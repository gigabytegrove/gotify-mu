package automation

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
	"github.com/rs/zerolog/log"
)

func sourceAllowed(raw string, ip net.IP) bool {
	if strings.TrimSpace(raw) == "" { return true }
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" { continue }
		if allowed := net.ParseIP(value); allowed != nil && allowed.Equal(ip) { return true }
		if _, network, err := net.ParseCIDR(value); err == nil && network.Contains(ip) { return true }
	}
	return false
}

var syslogPRI = regexp.MustCompile("^<(\\d{1,3})>")

func syslogPriority(raw string) (severity int, priority int, message string) {
	severity = 6
	if match := syslogPRI.FindStringSubmatch(raw); len(match) == 2 {
		if pri, err := strconv.Atoi(match[1]); err == nil { severity = pri % 8 }
		raw = syslogPRI.ReplaceAllString(raw, "")
	}
	priority = 2
	if severity <= 2 { priority = 8 } else if severity <= 4 { priority = 5 }
	return severity, priority, strings.TrimSpace(raw)
}

func (e *Engine) runSyslogLoop(ctx context.Context, integration *model.SyslogReceiver) {
	key := fmt.Sprintf("integration:syslog:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, key, 45*time.Second, func(leased context.Context) error {
			if strings.EqualFold(integration.Protocol, "tcp") {
				return e.runSyslogTCP(leased, integration)
			}
			return e.runSyslogUDP(leased, integration)
		})
		if ctx.Err() != nil { return }
		if errors.Is(err, errLeaseUnavailable) {
			e.setIntegrationStatus("syslog", integration.ID, "standby", "", false, false)
		} else if err != nil {
			e.setIntegrationStatus("syslog", integration.ID, "reconnecting", err.Error(), false, false)
		}
		select {
		case <-ctx.Done(): return
		case <-time.After(5*time.Second):
		}
	}
}

func (e *Engine) processSyslog(integration *model.SyslogReceiver, remote net.IP, raw string) {
	if !sourceAllowed(integration.AllowedCIDRs, remote) { return }
	severity, priority, message := syslogPriority(raw)
	if integration.MinSeverity >= 0 && severity > integration.MinSeverity { return }
	title := integration.Name
	if title == "" { title = "Syslog" }
	if _, err := e.Publish(integration.ApplicationID, title, message, priority); err != nil {
		log.Error().Err(err).Uint("integration_id", integration.ID).Msg("Syslog message could not be published")
	} else {
		e.setIntegrationStatus("syslog", integration.ID, "connected", "", false, true)
	}
}

func (e *Engine) runSyslogUDP(ctx context.Context, integration *model.SyslogReceiver) error {
	addr, err := net.ResolveUDPAddr("udp", integration.ListenAddress)
	if err != nil { return err }
	conn, err := net.ListenUDP("udp", addr)
	if err != nil { return err }
	defer conn.Close()
	e.setIntegrationStatus("syslog", integration.ID, "connected", "", true, false)
	go func(){ <-ctx.Done(); _ = conn.Close() }()
	buffer := make([]byte, 64*1024)
	for {
		n, remote, err := conn.ReadFromUDP(buffer)
		if err != nil { return err }
		e.processSyslog(integration, remote.IP, string(buffer[:n]))
	}
}

func (e *Engine) runSyslogTCP(ctx context.Context, integration *model.SyslogReceiver) error {
	listener, err := net.Listen("tcp", integration.ListenAddress)
	if err != nil { return err }
	defer listener.Close()
	e.setIntegrationStatus("syslog", integration.ID, "connected", "", true, false)
	go func(){ <-ctx.Done(); _ = listener.Close() }()
	for {
		conn, err := listener.Accept()
		if err != nil { return err }
		go func(c net.Conn) {
			defer c.Close()
			host, _, _ := net.SplitHostPort(c.RemoteAddr().String())
			ip := net.ParseIP(host)
			if !sourceAllowed(integration.AllowedCIDRs, ip) { return }
			scanner := bufio.NewScanner(io.LimitReader(c, 2<<20))
			scanner.Buffer(make([]byte, 4096), 64*1024)
			for scanner.Scan() { e.processSyslog(integration, ip, scanner.Text()) }
		}(conn)
	}
}

func (e *Engine) emailLoop() {
	defer e.wg.Done()
	for {
		select {
		case <-e.ctx.Done():
			return
		case msg := <-e.emailQueue:
			e.sendEmailGateways(&msg)
		}
	}
}

func (e *Engine) queueEmailGateways(msg *model.Message) {
	select {
	case e.emailQueue <- *msg:
	default:
		log.Warn().Uint("message_id", msg.ID).Msg("Email Gateway queue is full")
	}
}

func (e *Engine) sendEmailGateways(msg *model.Message) {
	gateways, err := e.db.GetEmailGatewaysForMessage(msg.ApplicationID, msg.Priority)
	if err != nil {
		log.Error().Err(err).Uint("message_id", msg.ID).Msg("Could not load Email Gateways")
		return
	}
	for _, gateway := range gateways {
		trigger := fmt.Sprintf("email:%d:%d", gateway.ID, msg.ID)
		if existing, _ := e.db.GetAutomationRunByTrigger(trigger); existing != nil { continue }
		run := &model.AutomationRun{
			Kind:"email", ObjectID:gateway.ID, TriggerKey:trigger,
			Status:"running", MessageID:msg.ID, StartedAt:time.Now(),
		}
		created, createErr := e.db.CreateAutomationRun(run)
		if createErr != nil || !created { continue }
		sendErr := sendEmailGateway(e.ctx, gateway, msg)
		done := time.Now()
		run.FinishedAt = &done
		if sendErr != nil {
			run.Status = "failed"
			run.Error = sendErr.Error()
		} else {
			run.Status = "completed"
		}
		_ = e.db.SaveAutomationRun(run)
	}
}

func sendEmailGateway(ctx context.Context, gateway *model.EmailGateway, msg *model.Message) error {
	password, err := security.Reveal(gateway.Password)
	if err != nil { return err }
	port := gateway.Port
	if port <= 0 {
		if gateway.UseTLS { port = 465 } else { port = 587 }
	}
	hostport := net.JoinHostPort(gateway.Host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout:15*time.Second}
	var conn net.Conn
	if gateway.UseTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", hostport, &tls.Config{MinVersion:tls.VersionTLS12, ServerName:gateway.Host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", hostport)
	}
	if err != nil { return err }
	defer conn.Close()

	client, err := smtp.NewClient(conn, gateway.Host)
	if err != nil { return err }
	defer client.Close()

	if gateway.StartTLS && !gateway.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok { return errors.New("SMTP server does not advertise STARTTLS") }
		if err := client.StartTLS(&tls.Config{MinVersion:tls.VersionTLS12, ServerName:gateway.Host}); err != nil { return err }
	}
	if gateway.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", gateway.Username, password, gateway.Host)); err != nil { return err }
	}
	if err := client.Mail(gateway.FromAddress); err != nil { return err }
	recipients := splitAddresses(gateway.ToAddresses)
	if len(recipients) == 0 { return errors.New("Email Gateway has no valid recipients") }
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil { return err }
	}
	writer, err := client.Data()
	if err != nil { return err }
	subject := strings.NewReplacer("\r"," ","\n"," ").Replace(msg.Title)
	payload := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		gateway.FromAddress, strings.Join(recipients, ", "), subject, msg.Message,
	)
	if _, err := io.WriteString(writer, payload); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func splitAddresses(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '\n' })
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if address, err := mail.ParseAddress(value); err == nil { out = append(out, address.Address) }
	}
	return out
}

func (e *Engine) runSMTPReceiverLoop(ctx context.Context, receiver *model.SMTPReceiver) {
	const key = "integration:smtp-receiver"
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, key, 45*time.Second, func(leased context.Context) error {
			return e.serveSMTP(leased, receiver)
		})
		if ctx.Err() != nil { return }
		if errors.Is(err, errLeaseUnavailable) {
			e.setIntegrationStatus("smtp-receiver", 1, "standby", "", false, false)
		} else if err != nil {
			e.setIntegrationStatus("smtp-receiver", 1, "reconnecting", err.Error(), false, false)
		}
		select {
		case <-ctx.Done(): return
		case <-time.After(5*time.Second):
		}
	}
}

func (e *Engine) serveSMTP(ctx context.Context, receiver *model.SMTPReceiver) error {
	listener, err := net.Listen("tcp", receiver.ListenAddress)
	if err != nil { return err }
	defer listener.Close()
	e.setIntegrationStatus("smtp-receiver", 1, "connected", "", true, false)
	go func(){ <-ctx.Done(); _ = listener.Close() }()
	for {
		conn, err := listener.Accept()
		if err != nil { return err }
		go e.handleSMTPConnection(receiver, conn)
	}
}

func smtpWrite(writer *textproto.Writer, code int, message string) {
	_ = writer.PrintfLine("%d %s", code, message)
}

func decodeSMTPAuth(value string) (string, string, bool) {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil { return "", "", false }
	parts := bytes.Split(raw, []byte{0})
	if len(parts) < 3 { return "", "", false }
	return string(parts[len(parts)-2]), string(parts[len(parts)-1]), true
}

func smtpPath(arg, prefix string) string {
	value := strings.TrimSpace(arg)
	if strings.HasPrefix(strings.ToUpper(value), prefix) { value = strings.TrimSpace(value[len(prefix):]) }
	return strings.Trim(strings.TrimSpace(value), "<>")
}

func (e *Engine) handleSMTPConnection(receiver *model.SMTPReceiver, conn net.Conn) {
	defer conn.Close()
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	remote := net.ParseIP(host)
	if !sourceAllowed(receiver.AllowedCIDRs, remote) { return }
	_ = conn.SetDeadline(time.Now().Add(5*time.Minute))

	text := textproto.NewConn(conn)
	defer text.Close()
	smtpWrite(text.W, 220, "Gotify MU SMTP Receiver")

	authenticated := receiver.Username == ""
	var from string
	var recipients []string

	for {
		line, err := text.ReadLine()
		if err != nil { return }
		command, arg := line, ""
		if idx := strings.IndexByte(line, ' '); idx >= 0 {
			command, arg = line[:idx], strings.TrimSpace(line[idx+1:])
		}
		switch strings.ToUpper(command) {
		case "EHLO", "HELO":
			_ = text.W.PrintfLine("250-Gotify MU")
			if receiver.Username != "" { _ = text.W.PrintfLine("250-AUTH PLAIN") }
			_ = text.W.PrintfLine("250 SIZE %d", receiver.MaxMessageBytes)
		case "AUTH":
			fields := strings.Fields(arg)
			if len(fields) == 2 && strings.EqualFold(fields[0], "PLAIN") {
				user, pass, ok := decodeSMTPAuth(fields[1])
				stored, revealErr := security.Reveal(receiver.Password)
				if revealErr == nil && ok && user == receiver.Username && pass == stored {
					authenticated = true
					smtpWrite(text.W, 235, "Authentication successful")
				} else {
					smtpWrite(text.W, 535, "Authentication failed")
				}
			} else {
				smtpWrite(text.W, 504, "AUTH PLAIN required")
			}
		case "MAIL":
			if !authenticated { smtpWrite(text.W, 530, "Authentication required"); continue }
			from = smtpPath(arg, "FROM:")
			recipients = nil
			smtpWrite(text.W, 250, "OK")
		case "RCPT":
			if !authenticated || from == "" { smtpWrite(text.W, 503, "MAIL required"); continue }
			rcpt := smtpPath(arg, "TO:")
			route, routeErr := e.db.GetSMTPRouteByRecipient(rcpt)
			if routeErr != nil || route == nil {
				smtpWrite(text.W, 550, "No route for recipient")
				continue
			}
			recipients = append(recipients, rcpt)
			smtpWrite(text.W, 250, "OK")
		case "DATA":
			if len(recipients) == 0 { smtpWrite(text.W, 503, "RCPT required"); continue }
			smtpWrite(text.W, 354, "End data with <CR><LF>.<CR><LF>")
			reader := text.DotReader()
			limit := receiver.MaxMessageBytes
			if limit <= 0 { limit = 10 << 20 }
			data, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
			if readErr != nil || int64(len(data)) > limit {
				smtpWrite(text.W, 552, "Message too large")
				continue
			}
			e.publishInboundEmail(recipients, data)
			e.setIntegrationStatus("smtp-receiver", 1, "connected", "", false, true)
			smtpWrite(text.W, 250, "Accepted")
			from = ""
			recipients = nil
		case "RSET":
			from = ""
			recipients = nil
			smtpWrite(text.W, 250, "OK")
		case "NOOP":
			smtpWrite(text.W, 250, "OK")
		case "QUIT":
			smtpWrite(text.W, 221, "Bye")
			return
		default:
			smtpWrite(text.W, 502, "Command not implemented")
		}
	}
}

func (e *Engine) publishInboundEmail(recipients []string, data []byte) {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		log.Warn().Err(err).Msg("SMTP Receiver could not parse message")
		return
	}
	body, err := io.ReadAll(io.LimitReader(msg.Body, 2<<20))
	if err != nil { return }
	subject := msg.Header.Get("Subject")
	if subject == "" { subject = "Email" }
	from := msg.Header.Get("From")
	content := strings.TrimSpace(string(body))
	if from != "" { content = "From: " + from + "\n\n" + content }

	seen := map[uint]struct{}{}
	for _, recipient := range recipients {
		route, routeErr := e.db.GetSMTPRouteByRecipient(recipient)
		if routeErr != nil || route == nil { continue }
		if _, ok := seen[route.ApplicationID]; ok { continue }
		seen[route.ApplicationID] = struct{}{}
		if _, pubErr := e.Publish(route.ApplicationID, subject, content, 0); pubErr != nil {
			log.Error().Err(pubErr).Uint("route_id", route.ID).Msg("SMTP Receiver message could not be published")
		}
	}
}
