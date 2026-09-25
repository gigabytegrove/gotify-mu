package connectors

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/rs/zerolog/log"
)

type Publisher interface {
	Publish(applicationID uint, title, message string, priority int) (*model.Message, error)
}

type Database interface {
	GetEmailGateways() ([]*model.EmailGateway, error)
	GetEmailGatewaysForMessage(applicationID uint, priority int) ([]*model.EmailGateway, error)
	GetEmailGatewayByID(id uint) (*model.EmailGateway, error)
	UpdateEmailGatewayStatus(id uint, sentAt *time.Time, lastError string, errorAt *time.Time) error

	GetSMTPRoutes() ([]*model.SMTPRoute, error)
	MatchSMTPRoute(recipient string) (*model.SMTPRoute, error)

	GetRSSMonitors() ([]*model.RSSMonitor, error)
	GetRSSMonitorByID(id uint) (*model.RSSMonitor, error)
	GetDueRSSMonitors(now time.Time) ([]*model.RSSMonitor, error)
	UpdateRSSMonitorStatus(item *model.RSSMonitor) error

	GetSyslogRoutes() ([]*model.SyslogRoute, error)

	GetCalendarMonitors() ([]*model.CalendarMonitor, error)
	GetCalendarMonitorByID(id uint) (*model.CalendarMonitor, error)
	GetDueCalendarMonitors(now time.Time) ([]*model.CalendarMonitor, error)
	SaveCalendarMonitor(item *model.CalendarMonitor) error

	MarkConnectorItemSeen(connectorType string, connectorID uint, itemKey string, now time.Time) (bool, error)
	CleanupConnectorSeenItems(before time.Time) error
	TryAcquireAutomationLease(name, holder string, now time.Time, ttl time.Duration) (bool, error)
	ReleaseAutomationLease(name, holder string) error
}

type Manager struct {
	db Database
	publisher Publisher
	ctx context.Context
	cancel context.CancelFunc
	wg sync.WaitGroup
	instanceID string
	httpClient *http.Client
}

func New(db Database, publisher Publisher) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	host, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
	m := &Manager{
		db:db, publisher:publisher, ctx:ctx, cancel:cancel, instanceID:instanceID,
		httpClient:&http.Client{Timeout:20*time.Second},
	}
	m.wg.Add(3)
	go m.pollLoop()
	go m.smtpListenerLoop()
	go m.syslogListenerLoop()
	return m
}

func (m *Manager) Close() {
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) OnMessage(message *model.Message) {
	if message == nil { return }
	gateways, err := m.db.GetEmailGatewaysForMessage(message.ApplicationID, message.Priority)
	if err != nil {
		log.Error().Err(err).Uint("message_id", message.ID).Msg("Could not load email delivery rules")
		return
	}
	for _, gateway := range gateways {
		item := *gateway
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := m.sendEmail(&item, message); err != nil {
				now := time.Now()
				_ = m.db.UpdateEmailGatewayStatus(item.ID, nil, err.Error(), &now)
				log.Warn().Err(err).Uint("email_gateway_id", item.ID).Msg("Email delivery failed")
				return
			}
			now := time.Now()
			_ = m.db.UpdateEmailGatewayStatus(item.ID, &now, "", nil)
		}()
	}
}

func (m *Manager) TestEmailGateway(id uint) error {
	item, err := m.db.GetEmailGatewayByID(id)
	if err != nil { return err }
	if item == nil { return errors.New("email delivery connection not found") }
	test := &model.Message{
		ApplicationID:item.SourceApplicationID,
		Title:"Gotify MU email delivery test",
		Message:"This message confirms that Gotify MU can deliver email through this connection.",
		Priority:item.MinPriority,
		Date:time.Now(),
	}
	return m.sendEmail(item, test)
}

func splitAddresses(raw string) []string {
	var result []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r==',' || r==';' || r=='\n' }) {
		value := strings.TrimSpace(part)
		if value != "" { result = append(result, value) }
	}
	return result
}

func (m *Manager) sendEmail(item *model.EmailGateway, message *model.Message) error {
	host := strings.TrimSpace(item.SMTPHost)
	if host == "" { return errors.New("SMTP host is required") }
	port := item.SMTPPort
	if port == 0 {
		if strings.EqualFold(item.TLSMode, "tls") { port = 465 } else { port = 587 }
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	var client *smtp.Client
	var conn net.Conn
	var err error
	switch strings.ToLower(strings.TrimSpace(item.TLSMode)) {
	case "tls":
		conn, err = tls.Dial("tcp", address, &tls.Config{MinVersion:tls.VersionTLS12,ServerName:host})
		if err != nil { return err }
		client, err = smtp.NewClient(conn, host)
	default:
		client, err = smtp.Dial(address)
	}
	if err != nil {
		if conn != nil { _ = conn.Close() }
		return err
	}
	defer client.Close()
	if strings.EqualFold(item.TLSMode, "starttls") {
		if err := client.StartTLS(&tls.Config{MinVersion:tls.VersionTLS12,ServerName:host}); err != nil { return err }
	}
	if item.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", item.Username, item.Password, host)); err != nil { return err }
	}
	from := strings.TrimSpace(item.FromAddress)
	if from == "" { return errors.New("from address is required") }
	to := splitAddresses(item.ToAddresses)
	if len(to)==0 { return errors.New("at least one recipient is required") }
	if err := client.Mail(from); err != nil { return err }
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil { return err }
	}
	writer, err := client.Data()
	if err != nil { return err }
	body := "From: "+from+"\r\n"+
		"To: "+strings.Join(to,", ")+"\r\n"+
		"Subject: "+sanitizeHeader(message.Title)+"\r\n"+
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n"+
		message.Message+"\r\n"
	if _, err := io.WriteString(writer, body); err != nil { _=writer.Close(); return err }
	if err := writer.Close(); err != nil { return err }
	return client.Quit()
}

func sanitizeHeader(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
}

func (m *Manager) pollLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(30*time.Second)
	defer ticker.Stop()
	m.poll(time.Now())
	for {
		select {
		case <-m.ctx.Done(): return
		case now := <-ticker.C: m.poll(now)
		}
	}
}

func (m *Manager) poll(now time.Time) {
	rssItems, err := m.db.GetDueRSSMonitors(now)
	if err != nil { log.Error().Err(err).Msg("Could not load RSS monitors") }
	for _, item := range rssItems {
		if item.IntervalMinutes < 1 { item.IntervalMinutes = 15 }
		if item.LastCheckedAt != nil && item.LastCheckedAt.Add(time.Duration(item.IntervalMinutes)*time.Minute).After(now) { continue }
		acquired, leaseErr := m.db.TryAcquireAutomationLease(fmt.Sprintf("rss:%d",item.ID),m.instanceID,now,2*time.Minute)
		if leaseErr != nil || !acquired { continue }
		m.pollRSS(item, now)
		_ = m.db.ReleaseAutomationLease(fmt.Sprintf("rss:%d",item.ID),m.instanceID)
	}
	calItems, err := m.db.GetDueCalendarMonitors(now)
	if err != nil { log.Error().Err(err).Msg("Could not load calendar monitors") }
	for _, item := range calItems {
		if item.IntervalMinutes < 1 { item.IntervalMinutes = 15 }
		if item.LastCheckedAt != nil && item.LastCheckedAt.Add(time.Duration(item.IntervalMinutes)*time.Minute).After(now) { continue }
		acquired, leaseErr := m.db.TryAcquireAutomationLease(fmt.Sprintf("ical:%d",item.ID),m.instanceID,now,2*time.Minute)
		if leaseErr != nil || !acquired { continue }
		m.pollCalendar(item, now)
		_ = m.db.ReleaseAutomationLease(fmt.Sprintf("ical:%d",item.ID),m.instanceID)
	}
	_ = m.db.CleanupConnectorSeenItems(now.AddDate(0,-6,0))
}

type rssDocument struct {
	Channel struct {
		Title string `xml:"title"`
		Items []struct {
			Title string `xml:"title"`
			Link string `xml:"link"`
			GUID string `xml:"guid"`
			Description string `xml:"description"`
			PubDate string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}
type atomDocument struct {
	Title string `xml:"title"`
	Entries []struct {
		ID string `xml:"id"`
		Title string `xml:"title"`
		Summary string `xml:"summary"`
		Content string `xml:"content"`
		Updated string `xml:"updated"`
		Links []struct{ Href string `xml:"href,attr"` } `xml:"link"`
	} `xml:"entry"`
}

type feedItem struct { Key, Title, Body string; When time.Time }

func (m *Manager) fetchURL(raw string, etag, modified string) ([]byte, string, string, bool, error) {
	request, err := http.NewRequestWithContext(m.ctx,http.MethodGet,raw,nil)
	if err != nil { return nil,"","",false,err }
	request.Header.Set("User-Agent","Gotify-MU/0.5")
	if etag!="" { request.Header.Set("If-None-Match",etag) }
	if modified!="" { request.Header.Set("If-Modified-Since",modified) }
	response, err := m.httpClient.Do(request)
	if err != nil { return nil,"","",false,err }
	defer response.Body.Close()
	if response.StatusCode==http.StatusNotModified { return nil,etag,modified,true,nil }
	if response.StatusCode<200 || response.StatusCode>=300 { return nil,"","",false,fmt.Errorf("HTTP %d",response.StatusCode) }
	body, err := io.ReadAll(io.LimitReader(response.Body,5<<20))
	if err != nil { return nil,"","",false,err }
	return body,response.Header.Get("ETag"),response.Header.Get("Last-Modified"),false,nil
}

func parseFeed(body []byte) ([]feedItem,error) {
	var rss rssDocument
	if err:=xml.Unmarshal(body,&rss);err==nil && len(rss.Channel.Items)>0 {
		result:=make([]feedItem,0,len(rss.Channel.Items))
		for _, item:=range rss.Channel.Items {
			key:=strings.TrimSpace(item.GUID); if key=="" { key=strings.TrimSpace(item.Link) }; if key=="" { key=item.Title+"|"+item.PubDate }
			bodyText:=strings.TrimSpace(item.Description)
			if item.Link!="" { bodyText += "\n"+item.Link }
			result=append(result,feedItem{Key:key,Title:item.Title,Body:bodyText})
		}
		return result,nil
	}
	var atom atomDocument
	if err:=xml.Unmarshal(body,&atom);err!=nil { return nil,err }
	result:=make([]feedItem,0,len(atom.Entries))
	for _, item:=range atom.Entries {
		link:=""; if len(item.Links)>0 { link=item.Links[0].Href }
		key:=strings.TrimSpace(item.ID); if key=="" { key=link }; if key=="" { key=item.Title+"|"+item.Updated }
		bodyText:=strings.TrimSpace(item.Summary); if bodyText=="" { bodyText=strings.TrimSpace(item.Content) }; if link!="" { bodyText+="\n"+link }
		result=append(result,feedItem{Key:key,Title:item.Title,Body:bodyText})
	}
	return result,nil
}

func (m *Manager) pollRSS(item *model.RSSMonitor, now time.Time) {
	body,etag,modified,notModified,err:=m.fetchURL(item.URL,item.ETag,item.LastModified)
	item.LastCheckedAt=&now
	if err!=nil {
		item.Status="error"; item.LastError=err.Error(); item.LastErrorAt=&now
		_ = m.db.UpdateRSSMonitorStatus(item); return
	}
	item.Status="ready"; item.LastError=""; item.LastErrorAt=nil
	if etag!="" { item.ETag=etag }; if modified!="" { item.LastModified=modified }
	if notModified { _=m.db.UpdateRSSMonitorStatus(item); return }
	feed,err:=parseFeed(body)
	if err!=nil { item.Status="error";item.LastError=err.Error();item.LastErrorAt=&now;_=m.db.UpdateRSSMonitorStatus(item);return }
	for i:=len(feed)-1;i>=0;i-- {
		entry:=feed[i]
		fresh,err:=m.db.MarkConnectorItemSeen("rss",item.ID,entry.Key,now)
		if err!=nil || !fresh { continue }
		title:=entry.Title; if title=="" { title=item.Name }
		if _,err:=m.publisher.Publish(item.ApplicationID,title,entry.Body,0);err==nil {
			t:=now; item.LastItemAt=&t
		}
	}
	_ = m.db.UpdateRSSMonitorStatus(item)
}

type calendarEvent struct { UID, Summary, Description, Location string; Start time.Time }

func unfoldICal(raw string) []string {
	raw=strings.ReplaceAll(raw,"\r\n","\n")
	lines:=strings.Split(raw,"\n")
	var out []string
	for _,line:=range lines {
		if (strings.HasPrefix(line," ")||strings.HasPrefix(line,"\t")) && len(out)>0 {
			out[len(out)-1]+=strings.TrimLeft(line," \t")
		} else { out=append(out,line) }
	}
	return out
}
func parseICalTime(value string) (time.Time,error) {
	value=strings.TrimSpace(value)
	layouts:=[]string{"20060102T150405Z","20060102T150405","20060102"}
	for _,layout:=range layouts { if t,err:=time.Parse(layout,value);err==nil{return t,nil} }
	return time.Time{},errors.New("unsupported calendar date "+value)
}
func parseCalendar(body []byte) []calendarEvent {
	lines:=unfoldICal(string(body))
	var result []calendarEvent
	var current *calendarEvent
	for _,line:=range lines {
		switch strings.TrimSpace(line) {
		case "BEGIN:VEVENT": current=&calendarEvent{}
		case "END:VEVENT": if current!=nil && !current.Start.IsZero() { result=append(result,*current) };current=nil
		default:
			if current==nil {continue}
			parts:=strings.SplitN(line,":",2);if len(parts)!=2{continue}
			key:=strings.ToUpper(strings.SplitN(parts[0],";",2)[0]);value:=strings.TrimSpace(parts[1])
			switch key {
			case "UID":current.UID=value
			case "SUMMARY":current.Summary=value
			case "DESCRIPTION":current.Description=strings.ReplaceAll(value,"\\n","\n")
			case "LOCATION":current.Location=value
			case "DTSTART":if t,err:=parseICalTime(value);err==nil{current.Start=t}
			}
		}
	}
	return result
}
func (m *Manager) pollCalendar(item *model.CalendarMonitor, now time.Time) {
	body,_,_,_,err:=m.fetchURL(item.URL,"","")
	item.LastCheckedAt=&now
	if err!=nil { item.Status="error";item.LastError=err.Error();item.LastErrorAt=&now;_=m.db.SaveCalendarMonitor(item);return }
	item.Status="ready";item.LastError="";item.LastErrorAt=nil
	ahead:=item.NotifyBeforeMinutes;if ahead<0{ahead=0}
	windowEnd:=now.Add(time.Duration(ahead)*time.Minute)
	for _,event:=range parseCalendar(body) {
		if event.Start.Before(now.Add(-time.Minute)) || event.Start.After(windowEnd) { continue }
		key:=event.UID+"|"+event.Start.UTC().Format(time.RFC3339)
		if event.UID=="" { key=event.Summary+"|"+event.Start.UTC().Format(time.RFC3339) }
		fresh,seenErr:=m.db.MarkConnectorItemSeen("ical",item.ID,key,now)
		if seenErr!=nil||!fresh{continue}
		text:=event.Description
		if event.Location!="" { if text!=""{text+="\n"};text+="Location: "+event.Location }
		if text!=""{text+="\n"};text+="Starts: "+event.Start.Format(time.RFC1123)
		if _,publishErr:=m.publisher.Publish(item.ApplicationID,event.Summary,text,0);publishErr==nil{t:=now;item.LastEventAt=&t}
	}
	_ = m.db.SaveCalendarMonitor(item)
}

func remoteIP(address net.Addr) net.IP {
	host,_,err:=net.SplitHostPort(address.String());if err!=nil{return net.ParseIP(address.String())};return net.ParseIP(host)
}
func ipAllowed(ip net.IP, cidrs string) bool {
	if strings.TrimSpace(cidrs)=="" { return true }
	for _,raw:=range strings.FieldsFunc(cidrs,func(r rune)bool{return r==','||r==';'||r==' '||r=='\n'||r=='\t'}) {
		_,network,err:=net.ParseCIDR(strings.TrimSpace(raw));if err==nil&&network.Contains(ip){return true}
	}
	return false
}

func (m *Manager) smtpListenerLoop() {
	defer m.wg.Done()
	listen:=strings.TrimSpace(os.Getenv("GOTIFY_MU_SMTP_LISTEN"));if listen==""{listen=":2525"}
	for {
		if m.ctx.Err()!=nil{return}
		acquired,err:=m.db.TryAcquireAutomationLease("smtp-receiver",m.instanceID,time.Now(),30*time.Second)
		if err!=nil||!acquired { select{case<-m.ctx.Done():return;case<-time.After(10*time.Second):continue} }
		listener,err:=net.Listen("tcp",listen)
		if err!=nil { _=m.db.ReleaseAutomationLease("smtp-receiver",m.instanceID);select{case<-m.ctx.Done():return;case<-time.After(10*time.Second):continue} }
		done:=make(chan struct{})
		go func(){select{case<-m.ctx.Done():_=listener.Close();case<-done:}}()
		for {
			_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(10*time.Second))
			conn,acceptErr:=listener.Accept()
			if netErr,ok:=acceptErr.(net.Error);ok&&netErr.Timeout() {
				acquired,_=m.db.TryAcquireAutomationLease("smtp-receiver",m.instanceID,time.Now(),30*time.Second)
				if !acquired||m.ctx.Err()!=nil{break};continue
			}
			if acceptErr!=nil { break }
			go m.handleSMTP(conn)
		}
		close(done);_=listener.Close();_=m.db.ReleaseAutomationLease("smtp-receiver",m.instanceID)
	}
}

func (m *Manager) handleSMTP(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5*time.Minute))
	reader:=bufio.NewReader(conn);writer:=bufio.NewWriter(conn)
	reply:=func(code int,text string){fmt.Fprintf(writer,"%d %s\r\n",code,text);_=writer.Flush()}
	reply(220,"Gotify MU SMTP Receiver")
	var recipient,user,pass string
	authenticated:=false
	for {
		line,err:=reader.ReadString('\n');if err!=nil{return}
		line=strings.TrimSpace(line);upper:=strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper,"EHLO")||strings.HasPrefix(upper,"HELO"):
			fmt.Fprint(writer,"250-Gotify MU\r\n250 AUTH PLAIN\r\n");_=writer.Flush()
		case strings.HasPrefix(upper,"AUTH PLAIN"):
			encoded:=strings.TrimSpace(strings.TrimPrefix(line,"AUTH PLAIN"))
			raw,decodeErr:=base64.StdEncoding.DecodeString(encoded)
			if decodeErr!=nil{reply(535,"Authentication failed");continue}
			parts:=strings.Split(string(raw),"\x00")
			if len(parts)>=3{user=parts[len(parts)-2];pass=parts[len(parts)-1];authenticated=true;reply(235,"Authenticated")}else{reply(535,"Authentication failed")}
		case strings.HasPrefix(upper,"MAIL FROM:"):reply(250,"OK")
		case strings.HasPrefix(upper,"RCPT TO:"):
			recipient=strings.Trim(strings.TrimSpace(line[len("RCPT TO:"):]),"<>")
			route,routeErr:=m.db.MatchSMTPRoute(recipient)
			if routeErr!=nil||route==nil||!ipAllowed(remoteIP(conn.RemoteAddr()),route.AllowedCIDRs){reply(550,"Recipient unavailable");recipient="";continue}
			if route.Username!=""&&(!authenticated||user!=route.Username||pass!=route.Password){reply(530,"Authentication required");recipient="";continue}
			reply(250,"OK")
		case upper=="DATA":
			if recipient==""{reply(503,"Valid recipient required");continue}
			reply(354,"End data with <CRLF>.<CRLF>")
			var data strings.Builder
			for {
				part,readErr:=reader.ReadString('\n');if readErr!=nil{return}
				if strings.TrimSpace(part)=="."{break}
				if strings.HasPrefix(part,".."){part=part[1:]}
				if data.Len()+(len(part))>5<<20{reply(552,"Message too large");return}
				data.WriteString(part)
			}
			route,_:=m.db.MatchSMTPRoute(recipient);if route==nil{reply(550,"Recipient unavailable");continue}
			msg,parseErr:=mail.ReadMessage(strings.NewReader(data.String()))
			if parseErr!=nil{reply(554,"Invalid message");continue}
			body,readErr:=io.ReadAll(io.LimitReader(msg.Body,4<<20));if readErr!=nil{reply(451,"Read failed");continue}
			title:=msg.Header.Get("Subject");if title==""{title=route.Name}
			content:=strings.TrimSpace(string(body));from:=msg.Header.Get("From");if from!=""{content="From: "+from+"\n\n"+content}
			if _,publishErr:=m.publisher.Publish(route.ApplicationID,title,content,0);publishErr!=nil{reply(451,"Delivery failed");continue}
			reply(250,"Accepted");recipient=""
		case upper=="RSET":recipient="";reply(250,"OK")
		case upper=="NOOP":reply(250,"OK")
		case upper=="QUIT":reply(221,"Bye");return
		default:reply(502,"Command not implemented")
		}
	}
}

func (m *Manager) syslogListenerLoop() {
	defer m.wg.Done()
	listen:=strings.TrimSpace(os.Getenv("GOTIFY_MU_SYSLOG_LISTEN"));if listen==""{listen=":5514"}
	for {
		if m.ctx.Err()!=nil{return}
		acquired,err:=m.db.TryAcquireAutomationLease("syslog-receiver",m.instanceID,time.Now(),30*time.Second)
		if err!=nil||!acquired{select{case<-m.ctx.Done():return;case<-time.After(10*time.Second):continue}}
		packet,err:=net.ListenPacket("udp",listen)
		if err!=nil{_=m.db.ReleaseAutomationLease("syslog-receiver",m.instanceID);select{case<-m.ctx.Done():return;case<-time.After(10*time.Second):continue}}
		buffer:=make([]byte,65535)
		for {
			_=packet.SetReadDeadline(time.Now().Add(10*time.Second))
			n,address,readErr:=packet.ReadFrom(buffer)
			if netErr,ok:=readErr.(net.Error);ok&&netErr.Timeout(){
				acquired,_=m.db.TryAcquireAutomationLease("syslog-receiver",m.instanceID,time.Now(),30*time.Second)
				if !acquired||m.ctx.Err()!=nil{break};continue
			}
			if readErr!=nil{break}
			m.handleSyslog(remoteIP(address),string(buffer[:n]))
		}
		_=packet.Close();_=m.db.ReleaseAutomationLease("syslog-receiver",m.instanceID)
	}
}

func parseSyslogPRI(message string)(facility,severity int,body string){
	facility=-1;severity=7;body=message
	if strings.HasPrefix(message,"<"){if end:=strings.Index(message,">");end>1{if pri,err:=strconv.Atoi(message[1:end]);err==nil{facility=pri/8;severity=pri%8;body=strings.TrimSpace(message[end+1:])}}}
	return
}
func (m *Manager) handleSyslog(ip net.IP, raw string) {
	facility,severity,body:=parseSyslogPRI(raw)
	routes,err:=m.db.GetSyslogRoutes();if err!=nil{return}
	for _,route:=range routes{
		if !route.Enabled||!ipAllowed(ip,route.AllowedCIDRs){continue}
		if route.Facility>=0&&facility!=route.Facility{continue}
		max:=route.MaxSeverity;if max<0||max>7{max=7};if severity>max{continue}
		priority:=0;if severity<=3{priority=8}else if severity<=4{priority=4}
		_,_=m.publisher.Publish(route.ApplicationID,route.Name,body,priority)
	}
}

func ValidateConnectorURL(raw string) error {
	parsed,err:=url.Parse(strings.TrimSpace(raw));if err!=nil{return err}
	if parsed.Scheme!="http"&&parsed.Scheme!="https"{return errors.New("URL must use http or https")}
	if parsed.Host==""{return errors.New("URL must include a host")}
	return nil
}
