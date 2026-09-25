package automation

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
	"github.com/rs/zerolog/log"
)

// Notifier delivers a realtime Gotify-compatible message to one user.
type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
}

// Database is the storage contract required by the native integration engine.
type Database interface {
	CreateMessage(message *model.Message) error
	CreateMessageOnce(message *model.Message) (*model.Message, bool, error)
	GetMessageByID(id uint) (*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationRecipientUserIDs(applicationID uint) ([]uint, error)

	GetQuietHoursPolicy(userID uint) (*model.QuietHoursPolicy, error)
	GetDigestPolicy(userID uint) (*model.DigestPolicy, error)
	QueueDigestItem(item *model.DigestItem) error
	GetDigestItems(userID uint) ([]*model.DigestItem, error)
	DeleteDigestItems(userID uint) error
	GetDueDigestPolicies(now time.Time) ([]*model.DigestPolicy, error)
	SaveDigestPolicy(item *model.DigestPolicy) error

	GetScheduledNotifications() ([]*model.ScheduledNotification, error)
	GetDueScheduledNotifications(now time.Time) ([]*model.ScheduledNotification, error)
	SaveScheduledNotification(item *model.ScheduledNotification) error

	GetEscalationRulesForMessage(applicationID uint, priority int) ([]*model.EscalationRule, error)
	GetEscalationRuleByID(id uint) (*model.EscalationRule, error)
	QueueEscalation(item *model.EscalationState) error
	GetDueEscalations(now time.Time) ([]*model.EscalationState, error)
	SaveEscalationState(item *model.EscalationState) error
	IsMessageAcknowledged(messageID uint) (bool, error)

	GetMQTTIntegrations() ([]*model.MQTTIntegration, error)
	GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error)
	SaveMQTTIntegration(item *model.MQTTIntegration) error
	GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error)
	GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error)
	SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error

	GetRSSIntegrations() ([]*model.RSSIntegration, error)
	SaveRSSIntegration(item *model.RSSIntegration) error
	GetCalendarIntegrations() ([]*model.CalendarIntegration, error)
	GetEmailGatewaysForMessage(applicationID uint, priority int) ([]*model.EmailGateway, error)
	GetSMTPReceiver() (*model.SMTPReceiver, error)
	SaveSMTPReceiver(item *model.SMTPReceiver) error
	GetSMTPRouteByRecipient(recipient string) (*model.SMTPRoute, error)
	GetSyslogReceivers() ([]*model.SyslogReceiver, error)

	AcquireAutomationLease(key, owner string, now time.Time, ttl time.Duration) (bool, error)
	ReleaseAutomationLease(key, owner string) error
	CreateAutomationRun(item *model.AutomationRun) (bool, error)
	GetAutomationRunByTrigger(triggerKey string) (*model.AutomationRun, error)
	SaveAutomationRun(item *model.AutomationRun) error
	SaveIntegrationStatus(item *model.IntegrationStatus) error
	GetIntegrationStatus(kind string, objectID uint) (*model.IntegrationStatus, error)
}

// Engine runs scheduled work and persistent native integrations.
type Engine struct {
	db       Database
	notifier Notifier

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	integrationMu      sync.Mutex
	integrationCancels []context.CancelFunc
	integrationWG      sync.WaitGroup
	reload             chan struct{}
	instanceID         string
	emailQueue         chan model.Message
}

func New(db Database, notifier Notifier) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		db: db,
		notifier: notifier,
		ctx: ctx,
		cancel: cancel,
		reload: make(chan struct{}, 1),
		instanceID: newInstanceID(),
		emailQueue: make(chan model.Message, 256),
	}
	e.wg.Add(3)
	go e.schedulerLoop()
	go e.integrationLoop()
	go e.emailLoop()
	return e
}

func (e *Engine) Close() {
	e.cancel()
	e.stopIntegrations()
	e.wg.Wait()
}

func (e *Engine) ReloadIntegrations() {
	select {
	case e.reload <- struct{}{}:
	default:
	}
}

// Publish stores a normal Gotify MU message and applies native delivery policies.
func (e *Engine) Publish(applicationID uint, title, message string, priority int) (*model.Message, error) {
	app, err := e.db.GetApplicationByID(applicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("channel not found")
	}
	if strings.TrimSpace(title) == "" {
		title = app.Name
	}
	msg := &model.Message{
		ApplicationID: applicationID,
		Title: title,
		Message: message,
		Priority: priority,
		Date: time.Now(),
	}
	if _, err := e.storeAndDeliver(msg, true); err != nil {
		return nil, err
	}
	return msg, nil
}

// StoreAndDeliver stores a normal message and applies quiet hours, digests, and escalations.
func (e *Engine) StoreAndDeliver(msg *model.Message) (*model.MessageExternal, error) {
	return e.storeAndDeliver(msg, true)
}

func (e *Engine) StoreAndDeliverForUser(msg *model.Message, userID uint) (*model.MessageExternal, error) {
	if msg.Date.IsZero() { msg.Date = time.Now() }
	if err := e.db.CreateMessage(msg); err != nil { return nil, err }
	external := externalMessage(msg)
	if err := e.deliver(userID, msg, external); err != nil { return nil, err }
	if err := e.queueEscalations(msg); err != nil {
		log.Error().Err(err).Uint("message_id", msg.ID).Msg("Could not queue plugin escalation")
	}
	return external, nil
}


func (e *Engine) publishWithKey(applicationID uint, title, message string, priority int, dedupKey string, parentMessageID uint) (*model.Message, error) {
	app, err := e.db.GetApplicationByID(applicationID)
	if err != nil { return nil, err }
	if app == nil { return nil, errors.New("channel not found") }
	if strings.TrimSpace(title) == "" { title = app.Name }
	msg := &model.Message{
		ApplicationID: applicationID,
		Title: title,
		Message: message,
		Priority: priority,
		Date: time.Now(),
		DedupKey: dedupKey,
		ParentMessageID: parentMessageID,
	}
	if _, err := e.storeAndDeliver(msg, true); err != nil { return nil, err }
	return msg, nil
}

func (e *Engine) storeAndDeliver(msg *model.Message, allowEscalation bool) (*model.MessageExternal, error) {
	if msg.Date.IsZero() {
		msg.Date = time.Now()
	}
	if msg.DedupKey != "" {
		stored, created, err := e.db.CreateMessageOnce(msg)
		if err != nil { return nil, err }
		if !created {
			return externalMessage(stored), nil
		}
		msg = stored
	} else if err := e.db.CreateMessage(msg); err != nil {
		return nil, err
	}
	recipients, err := e.db.GetApplicationRecipientUserIDs(msg.ApplicationID)
	if err != nil {
		return nil, err
	}
	external := externalMessage(msg)
	for _, userID := range recipients {
		if err := e.deliver(userID, msg, external); err != nil {
			log.Error().Err(err).Uint("user_id", userID).Uint("message_id", msg.ID).Msg("Could not apply delivery policy")
			e.notifier.Notify(userID, external)
		}
	}
	if allowEscalation {
		if err := e.queueEscalations(msg); err != nil {
			log.Error().Err(err).Uint("message_id", msg.ID).Msg("Could not queue escalation")
		}
	}
	e.queueEmailGateways(msg)
	return external, nil
}

func (e *Engine) deliver(userID uint, msg *model.Message, external *model.MessageExternal) error {
	digest, err := e.db.GetDigestPolicy(userID)
	if err != nil {
		return err
	}
	if digest != nil && digest.Enabled && msg.Priority < digest.ImmediatePriority {
		return e.db.QueueDigestItem(&model.DigestItem{
			UserID: userID,
			MessageID: msg.ID,
			ApplicationID: msg.ApplicationID,
			Title: msg.Title,
			Message: msg.Message,
			Priority: msg.Priority,
		})
	}

	quiet, err := e.db.GetQuietHoursPolicy(userID)
	if err != nil {
		return err
	}
	if quiet != nil && quiet.Enabled && msg.Priority < quiet.AllowPriority && quietNow(quiet, time.Now()) {
		return nil
	}

	e.notifier.Notify(userID, external)
	return nil
}

func quietNow(policy *model.QuietHoursPolicy, now time.Time) bool {
	loc := time.UTC
	if policy.Timezone != "" {
		if parsed, err := time.LoadLocation(policy.Timezone); err == nil {
			loc = parsed
		}
	}
	local := now.In(loc)
	minute := local.Hour()*60 + local.Minute()
	start := clamp(policy.StartMinute, 0, 1439)
	end := clamp(policy.EndMinute, 0, 1439)
	if start == end {
		return true
	}
	if start < end {
		return minute >= start && minute < end
	}
	return minute >= start || minute < end
}

func (e *Engine) queueEscalations(msg *model.Message) error {
	rules, err := e.db.GetEscalationRulesForMessage(msg.ApplicationID, msg.Priority)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		delay := rule.DelayMinutes
		if delay < 1 {
			delay = 1
		}
		if err := e.db.QueueEscalation(&model.EscalationState{
			RuleID: rule.ID,
			MessageID: msg.ID,
			DueAt: time.Now().Add(time.Duration(delay) * time.Minute),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) schedulerLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		now := time.Now()
		acquired, err := e.db.AcquireAutomationLease("scheduler", e.instanceID, now, 25*time.Second)
		if err != nil {
			log.Error().Err(err).Msg("Could not acquire automation scheduler lease")
		} else if acquired {
			e.runDue(now)
			_ = e.db.ReleaseAutomationLease("scheduler", e.instanceID)
		}
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (e *Engine) runDue(now time.Time) {
	e.runSchedules(now)
	e.runDigests(now)
	e.runEscalations(now)
}

func (e *Engine) runSchedules(now time.Time) {
	items, err := e.db.GetDueScheduledNotifications(now)
	if err != nil {
		log.Error().Err(err).Msg("Could not load scheduled notifications")
		return
	}
	for _, item := range items {
		if item.NextRunAt == nil { continue }
		triggerTime := item.NextRunAt.UTC()

		if scheduleExpired(item, now) {
			item.Enabled = false
			item.NextRunAt = nil
			if err := e.db.SaveScheduledNotification(item); err != nil {
				log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not disable completed schedule")
			}
			continue
		}

		if item.MisfirePolicy == "skip" && now.Sub(triggerTime) > time.Minute {
			item.NextRunAt = NextScheduleRun(item, now)
			if item.NextRunAt == nil { item.Enabled = false }
			if err := e.db.SaveScheduledNotification(item); err != nil {
				log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not advance skipped schedule")
			}
			continue
		}

		triggerKey := fmt.Sprintf("schedule:%d:%d", item.ID, triggerTime.UnixNano())
		run, err := e.db.GetAutomationRunByTrigger(triggerKey)
		if err != nil {
			log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not inspect schedule run")
			continue
		}
		if run == nil {
			run = &model.AutomationRun{Kind:"schedule", ObjectID:item.ID, TriggerKey:triggerKey, Status:"running", StartedAt:now}
			created, createErr := e.db.CreateAutomationRun(run)
			if createErr != nil {
				log.Error().Err(createErr).Uint("schedule_id", item.ID).Msg("Could not create schedule run")
				continue
			}
			if !created {
				continue
			}
		} else if run.Status == "completed" {
			item.NextRunAt = NextScheduleRun(item, triggerTime.Add(time.Second))
			if item.NextRunAt == nil { item.Enabled = false }
			_ = e.db.SaveScheduledNotification(item)
			continue
		}

		msg, publishErr := e.publishWithKey(item.ApplicationID, item.Title, item.Message, item.Priority, triggerKey, 0)
		finished := time.Now()
		run.FinishedAt = &finished
		if publishErr != nil {
			run.Status = "failed"
			run.Error = publishErr.Error()
			_ = e.db.SaveAutomationRun(run)
			log.Error().Err(publishErr).Uint("schedule_id", item.ID).Msg("Scheduled notification failed")
			continue
		}
		run.Status = "completed"
		run.MessageID = msg.ID
		run.Error = ""
		_ = e.db.SaveAutomationRun(run)

		runAt := now
		item.LastRunAt = &runAt
		item.RunCount++
		if item.ScheduleType == "once" || (item.MaxRuns > 0 && item.RunCount >= item.MaxRuns) {
			item.Enabled = false
			item.NextRunAt = nil
		} else {
			item.NextRunAt = NextScheduleRun(item, triggerTime.Add(time.Second))
			if item.NextRunAt == nil { item.Enabled = false }
		}
		if err := e.db.SaveScheduledNotification(item); err != nil {
			log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not update schedule")
		}
	}
}

func NextScheduleRun(item *model.ScheduledNotification, now time.Time) *time.Time {
	loc := time.UTC
	if item.Timezone != "" {
		if parsed, err := time.LoadLocation(item.Timezone); err == nil {
			loc = parsed
		}
	}
	localNow := now.In(loc)

	var candidate time.Time
	switch item.ScheduleType {
	case "once":
		if item.RunAt == nil || !item.RunAt.After(now) { return nil }
		candidate = item.RunAt.In(loc)
	case "hourly":
		candidate = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), localNow.Hour(), clamp(item.Minute, 0, 59), 0, 0, loc)
		if !candidate.After(localNow) { candidate = candidate.Add(time.Hour) }
	case "daily":
		candidate = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), clamp(item.Hour, 0, 23), clamp(item.Minute, 0, 59), 0, 0, loc)
		if !candidate.After(localNow) { candidate = candidate.AddDate(0, 0, 1) }
	case "weekly":
		weekday := time.Weekday(clamp(item.Weekday, 0, 6))
		days := (int(weekday) - int(localNow.Weekday()) + 7) % 7
		candidate = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), clamp(item.Hour, 0, 23), clamp(item.Minute, 0, 59), 0, 0, loc).AddDate(0, 0, days)
		if !candidate.After(localNow) { candidate = candidate.AddDate(0, 0, 7) }
	case "cron":
		next, err := nextCronRun(item.CronExpression, localNow)
		if err != nil { return nil }
		candidate = next
	default:
		return nil
	}

	for isExcludedScheduleDate(item.ExcludeDates, candidate) {
		switch item.ScheduleType {
		case "once":
			return nil
		case "hourly":
			candidate = candidate.Add(time.Hour)
		case "daily":
			candidate = candidate.AddDate(0, 0, 1)
		case "weekly":
			candidate = candidate.AddDate(0, 0, 7)
		case "cron":
			next, err := nextCronRun(item.CronExpression, candidate)
			if err != nil { return nil }
			candidate = next
		}
	}

	value := candidate.UTC()
	if item.EndAt != nil && value.After(item.EndAt.UTC()) {
		return nil
	}
	if item.MaxRuns > 0 && item.RunCount >= item.MaxRuns {
		return nil
	}
	return &value
}

func scheduleExpired(item *model.ScheduledNotification, now time.Time) bool {
	if item.MaxRuns > 0 && item.RunCount >= item.MaxRuns { return true }
	return item.EndAt != nil && now.After(item.EndAt.UTC())
}

// ValidateExcludeDates accepts comma-separated YYYY-MM-DD dates.
func ValidateExcludeDates(raw string) error {
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" { continue }
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("invalid excluded date %q; use YYYY-MM-DD", value)
		}
	}
	return nil
}

func isExcludedScheduleDate(raw string, candidate time.Time) bool {
	if strings.TrimSpace(raw) == "" { return false }
	value := candidate.Format("2006-01-02")
	for _, part := range strings.Split(raw, ",") {
		if strings.TrimSpace(part) == value { return true }
	}
	return false
}

// ValidateCronExpression accepts standard five-field cron expressions:
// minute hour day-of-month month day-of-week.
func ValidateCronExpression(expression string) error {
	_, err := parseCronExpression(expression)
	return err
}

type cronSpec struct {
	minute map[int]bool
	hour   map[int]bool
	day    map[int]bool
	month  map[int]bool
	weekday map[int]bool
}

func parseCronExpression(expression string) (*cronSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expression))
	if len(fields) != 5 {
		return nil, errors.New("cron schedule must contain five fields: minute hour day month weekday")
	}
	minute, err := parseCronField(fields[0], 0, 59)
	if err != nil { return nil, fmt.Errorf("invalid cron minute: %w", err) }
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil { return nil, fmt.Errorf("invalid cron hour: %w", err) }
	day, err := parseCronField(fields[2], 1, 31)
	if err != nil { return nil, fmt.Errorf("invalid cron day: %w", err) }
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil { return nil, fmt.Errorf("invalid cron month: %w", err) }
	weekday, err := parseCronField(fields[4], 0, 7)
	if err != nil { return nil, fmt.Errorf("invalid cron weekday: %w", err) }
	if weekday[7] { weekday[0] = true; delete(weekday, 7) }
	return &cronSpec{minute:minute,hour:hour,day:day,month:month,weekday:weekday}, nil
}

func parseCronField(raw string, min, max int) (map[int]bool, error) {
	result := make(map[int]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" { return nil, errors.New("empty field value") }
		step := 1
		base := part
		if slash := strings.IndexByte(part, '/'); slash >= 0 {
			base = part[:slash]
			parsed, err := strconv.Atoi(part[slash+1:])
			if err != nil || parsed < 1 { return nil, errors.New("invalid step") }
			step = parsed
		}
		start, end := min, max
		if base != "*" {
			if dash := strings.IndexByte(base, '-'); dash >= 0 {
				var err error
				start, err = strconv.Atoi(base[:dash])
				if err != nil { return nil, errors.New("invalid range start") }
				end, err = strconv.Atoi(base[dash+1:])
				if err != nil { return nil, errors.New("invalid range end") }
			} else {
				value, err := strconv.Atoi(base)
				if err != nil { return nil, errors.New("invalid value") }
				start, end = value, value
			}
		}
		if start < min || end > max || start > end { return nil, errors.New("value outside allowed range") }
		for value := start; value <= end; value += step { result[value] = true }
	}
	if len(result) == 0 { return nil, errors.New("field has no values") }
	return result, nil
}

func nextCronRun(expression string, after time.Time) (time.Time, error) {
	spec, err := parseCronExpression(expression)
	if err != nil { return time.Time{}, err }
	candidate := after.Truncate(time.Minute).Add(time.Minute)
	deadline := candidate.AddDate(5, 0, 0)
	for !candidate.After(deadline) {
		if spec.minute[candidate.Minute()] &&
			spec.hour[candidate.Hour()] &&
			spec.day[candidate.Day()] &&
			spec.month[int(candidate.Month())] &&
			spec.weekday[int(candidate.Weekday())] {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}, errors.New("cron expression has no matching time within five years")
}

func (e *Engine) runDigests(now time.Time) {
	policies, err := e.db.GetDueDigestPolicies(now)
	if err != nil {
		log.Error().Err(err).Msg("Could not load digest policies")
		return
	}
	for _, policy := range policies {
		items, err := e.db.GetDigestItems(policy.UserID)
		if err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not load digest items")
			continue
		}
		if len(items) > 0 {
			lines := make([]string, 0, len(items))
			highest := 0
			for _, item := range items {
				if item.Priority > highest {
					highest = item.Priority
				}
				line := item.Title
				if strings.TrimSpace(line) == "" {
					line = item.Message
				}
				if len(line) > 120 {
					line = line[:117] + "..."
				}
				lines = append(lines, "• "+line)
			}
			last := items[len(items)-1]
			e.notifier.Notify(policy.UserID, &model.MessageExternal{
				ID: last.MessageID,
				ApplicationID: last.ApplicationID,
				Title: fmt.Sprintf("%d notification digest", len(items)),
				Message: strings.Join(lines, "\n"),
				Priority: &highest,
				Date: now,
			})
			if err := e.db.DeleteDigestItems(policy.UserID); err != nil {
				log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not clear digest")
			}
		}
		lastSent := now
		policy.LastSentAt = &lastSent
		interval := policy.IntervalMinutes
		if interval < 15 {
			interval = 15
		}
		next := now.Add(time.Duration(interval) * time.Minute)
		policy.NextRunAt = &next
		if err := e.db.SaveDigestPolicy(policy); err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not update digest policy")
		}
	}
}

func (e *Engine) runEscalations(now time.Time) {
	states, err := e.db.GetDueEscalations(now)
	if err != nil {
		log.Error().Err(err).Msg("Could not load escalations")
		return
	}
	for _, state := range states {
		acknowledged, err := e.db.IsMessageAcknowledged(state.MessageID)
		if err != nil {
			log.Error().Err(err).Uint("message_id", state.MessageID).Msg("Could not inspect acknowledgement")
			continue
		}
		if !acknowledged {
			rule, err := e.db.GetEscalationRuleByID(state.RuleID)
			if err != nil || rule == nil || !rule.Enabled {
				state.Completed = true
			} else {
				msg, loadErr := e.db.GetMessageByID(state.MessageID)
				if loadErr != nil || msg == nil {
					state.Completed = true
				} else {
					title := msg.Title
					if title == "" {
						title = "Escalated notification"
					} else {
						title = "Escalated: " + title
					}
					body := msg.Message + "\n\nThis notification was escalated because it was not acknowledged."
					triggerKey := fmt.Sprintf("escalation:%d", state.ID)
					run, runErr := e.db.GetAutomationRunByTrigger(triggerKey)
					if runErr != nil {
						log.Error().Err(runErr).Uint("rule_id", rule.ID).Msg("Could not inspect escalation run")
						continue
					}
					if run == nil {
						run = &model.AutomationRun{Kind:"escalation", ObjectID:rule.ID, TriggerKey:triggerKey, Status:"running", StartedAt:now}
						if _, createErr := e.db.CreateAutomationRun(run); createErr != nil {
							log.Error().Err(createErr).Uint("rule_id", rule.ID).Msg("Could not create escalation run")
							continue
						}
					}
					escalated, publishErr := e.publishWithKey(rule.TargetApplicationID, title, body, msg.Priority, triggerKey, msg.ID)
					finished := time.Now()
					run.FinishedAt = &finished
					if publishErr != nil {
						run.Status = "failed"
						run.Error = publishErr.Error()
						_ = e.db.SaveAutomationRun(run)
						log.Error().Err(publishErr).Uint("rule_id", rule.ID).Msg("Escalation delivery failed")
						continue
					}
					run.Status = "completed"
					run.MessageID = escalated.ID
					run.Error = ""
					_ = e.db.SaveAutomationRun(run)
					state.Completed = true
				}
			}
		} else {
			state.Completed = true
		}
		done := now
		state.DoneAt = &done
		if err := e.db.SaveEscalationState(state); err != nil {
			log.Error().Err(err).Uint("escalation_id", state.ID).Msg("Could not complete escalation")
		}
	}
}

func (e *Engine) integrationLoop() {
	defer e.wg.Done()
	e.restartIntegrations()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.reload:
			e.restartIntegrations()
		}
	}
}

func (e *Engine) stopIntegrations() {
	e.integrationMu.Lock()
	cancels := append([]context.CancelFunc(nil), e.integrationCancels...)
	e.integrationCancels = nil
	e.integrationMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	e.integrationWG.Wait()
}

func (e *Engine) restartIntegrations() {
	e.stopIntegrations()
	mqttItems, err := e.db.GetMQTTIntegrations()
	if err != nil {
		log.Error().Err(err).Msg("Could not load MQTT integrations")
	}
	haItems, haErr := e.db.GetHomeAssistantIntegrations()
	if haErr != nil {
		log.Error().Err(haErr).Msg("Could not load Home Assistant integrations")
	}
	rssItems, rssErr := e.db.GetRSSIntegrations()
	if rssErr != nil { log.Error().Err(rssErr).Msg("Could not load RSS integrations") }
	calendarItems, calendarErr := e.db.GetCalendarIntegrations()
	if calendarErr != nil { log.Error().Err(calendarErr).Msg("Could not load Calendar integrations") }
	syslogItems, syslogErr := e.db.GetSyslogReceivers()
	if syslogErr != nil { log.Error().Err(syslogErr).Msg("Could not load Syslog integrations") }
	smtpReceiver, smtpErr := e.db.GetSMTPReceiver()
	if smtpErr != nil { log.Error().Err(smtpErr).Msg("Could not load SMTP Receiver") }

	e.integrationMu.Lock()
	defer e.integrationMu.Unlock()
	for _, item := range mqttItems {
		if !item.Enabled {
			continue
		}
		password, revealErr := security.Reveal(item.Password)
		if revealErr != nil {
			log.Error().Err(revealErr).Uint("integration_id", item.ID).Msg("MQTT credentials could not be decrypted")
			continue
		}
		if item.Password != "" && !strings.HasPrefix(item.Password, "enc:v1:") {
			if protected, protectErr := security.Protect(item.Password); protectErr == nil {
				item.Password = protected
				if saveErr := e.db.SaveMQTTIntegration(item); saveErr != nil {
					log.Warn().Err(saveErr).Uint("integration_id", item.ID).Msg("Could not migrate MQTT credentials")
				}
			}
		}
		runtimeItem := *item
		runtimeItem.Password = password
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.MQTTIntegration) {
			defer e.integrationWG.Done()
			e.runMQTTLoop(ctx, &integration)
		}(runtimeItem)
	}
	for _, item := range haItems {
		if !item.Enabled {
			continue
		}
		token, revealErr := security.Reveal(item.Token)
		if revealErr != nil {
			log.Error().Err(revealErr).Uint("integration_id", item.ID).Msg("Home Assistant credentials could not be decrypted")
			continue
		}
		if item.Token != "" && !strings.HasPrefix(item.Token, "enc:v1:") {
			if protected, protectErr := security.Protect(item.Token); protectErr == nil {
				item.Token = protected
				if saveErr := e.db.SaveHomeAssistantIntegration(item); saveErr != nil {
					log.Warn().Err(saveErr).Uint("integration_id", item.ID).Msg("Could not migrate Home Assistant credentials")
				}
			}
		}
		runtimeItem := *item
		runtimeItem.Token = token
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.HomeAssistantIntegration) {
			defer e.integrationWG.Done()
			e.runHomeAssistantLoop(ctx, &integration)
		}(runtimeItem)
	}
	for _, item := range rssItems {
		if !item.Enabled { continue }
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.RSSIntegration) {
			defer e.integrationWG.Done()
			e.runRSSLoop(ctx, &integration)
		}(*item)
	}
	for _, item := range calendarItems {
		if !item.Enabled { continue }
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.CalendarIntegration) {
			defer e.integrationWG.Done()
			e.runCalendarLoop(ctx, &integration)
		}(*item)
	}
	for _, item := range syslogItems {
		if !item.Enabled { continue }
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.SyslogReceiver) {
			defer e.integrationWG.Done()
			e.runSyslogLoop(ctx, &integration)
		}(*item)
	}
	if smtpReceiver != nil && smtpReceiver.Enabled {
		password, revealErr := security.Reveal(smtpReceiver.Password)
		if revealErr != nil {
			log.Error().Err(revealErr).Msg("SMTP Receiver credentials could not be decrypted")
		} else {
			if smtpReceiver.Password != "" && !strings.HasPrefix(smtpReceiver.Password, "enc:v1:") {
				if protected, protectErr := security.Protect(smtpReceiver.Password); protectErr == nil {
					smtpReceiver.Password = protected
					if saveErr := e.db.SaveSMTPReceiver(smtpReceiver); saveErr != nil {
						log.Warn().Err(saveErr).Msg("Could not migrate SMTP Receiver credentials")
					}
				}
			}
			runtimeReceiver := *smtpReceiver
			runtimeReceiver.Password = password
			ctx, cancel := context.WithCancel(e.ctx)
			e.integrationCancels = append(e.integrationCancels, cancel)
			e.integrationWG.Add(1)
			go func(receiver model.SMTPReceiver) {
				defer e.integrationWG.Done()
				e.runSMTPReceiverLoop(ctx, &receiver)
			}(runtimeReceiver)
		}
	}
}

func (e *Engine) TestMQTT(id uint) error {
	integration, err := e.db.GetMQTTIntegrationByID(id)
	if err != nil { return err }
	if integration == nil { return errors.New("MQTT connection not found") }
	password, err := security.Reveal(integration.Password)
	if err != nil { return err }
	ctx, cancel := context.WithTimeout(e.ctx, 12*time.Second)
	defer cancel()
	conn, err := dialMQTT(ctx, integration.BrokerURL)
	if err != nil { return err }
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)
	clientID := integration.ClientID
	if clientID == "" { clientID = "gotify-mu-test-" + strconv.FormatUint(uint64(integration.ID), 10) }
	if err := mqttConnect(conn, reader, clientID, integration.Username, password); err != nil { return err }
	if err := mqttSubscribe(conn, reader, integration.Topic); err != nil { return err }
	return nil
}

func (e *Engine) TestHomeAssistant(id uint) error {
	integration, err := e.db.GetHomeAssistantIntegrationByID(id)
	if err != nil { return err }
	if integration == nil { return errors.New("Home Assistant connection not found") }
	token, err := security.Reveal(integration.Token)
	if err != nil { return err }
	parsed, err := url.Parse(integration.BaseURL)
	if err != nil { return err }
	switch parsed.Scheme {
	case "https": parsed.Scheme = "wss"
	case "http": parsed.Scheme = "ws"
	default: return errors.New("Home Assistant URL must use http or https")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/websocket"
	ctx, cancel := context.WithTimeout(e.ctx, 12*time.Second)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, parsed.String(), nil)
	if err != nil { return err }
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var hello map[string]any
	if err := conn.ReadJSON(&hello); err != nil { return err }
	if hello["type"] != "auth_required" { return errors.New("unexpected Home Assistant authentication response") }
	if err := conn.WriteJSON(map[string]any{"type":"auth","access_token":token}); err != nil { return err }
	var response map[string]any
	if err := conn.ReadJSON(&response); err != nil { return err }
	if response["type"] != "auth_ok" { return errors.New("Home Assistant authentication failed") }
	return nil
}

func (e *Engine) runMQTTLoop(ctx context.Context, integration *model.MQTTIntegration) {
	key := fmt.Sprintf("integration:mqtt:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, key, 45*time.Second, func(leased context.Context) error {
			e.setIntegrationStatus("mqtt", integration.ID, "connecting", "", false, false)
			return e.runMQTT(leased, integration)
		})
		if ctx.Err() != nil { return }
		if errors.Is(err, errLeaseUnavailable) {
			e.setIntegrationStatus("mqtt", integration.ID, "standby", "", false, false)
		} else if err != nil {
			e.setIntegrationStatus("mqtt", integration.ID, "reconnecting", err.Error(), false, false)
			log.Warn().Err(err).Uint("integration_id", integration.ID).Msg("MQTT connection interrupted")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (e *Engine) runMQTT(ctx context.Context, integration *model.MQTTIntegration) error {
	conn, err := dialMQTT(ctx, integration.BrokerURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stopClose:
		}
	}()
	defer close(stopClose)

	reader := bufio.NewReader(conn)
	pendingQoS2 := make(map[uint16]struct{})
	clientID := integration.ClientID
	if clientID == "" {
		clientID = "gotify-mu-" + strconv.FormatUint(uint64(integration.ID), 10)
	}
	if err := mqttConnect(conn, reader, clientID, integration.Username, integration.Password); err != nil {
		return err
	}
	if err := mqttSubscribe(conn, reader, integration.Topic); err != nil {
		return err
	}
	e.setIntegrationStatus("mqtt", integration.ID, "connected", "", true, false)

	for {
		if ctx.Err() != nil {
			return nil
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		header, body, err := readMQTTPacket(reader)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			if _, writeErr := conn.Write([]byte{0xC0, 0x00}); writeErr != nil {
				return writeErr
			}
			continue
		}
		if err != nil {
			return err
		}
		packetType := header >> 4
		if packetType == 6 {
			if len(body) < 2 { return errors.New("invalid MQTT PUBREL packet") }
			packetID := binary.BigEndian.Uint16(body[:2])
			_, _ = conn.Write([]byte{0x70, 0x02, byte(packetID >> 8), byte(packetID)})
			delete(pendingQoS2, packetID)
			continue
		}
		if packetType != 3 { continue }

		topic, payload, packetID, qos, err := decodePublish(header, body)
		if err != nil { return err }
		shouldPublish := true
		if qos == 2 {
			if _, seen := pendingQoS2[packetID]; seen { shouldPublish = false }
		}
		if shouldPublish {
			title, message, priority := integration.Name, string(payload), 0
			var object map[string]any
			if json.Unmarshal(payload, &object) == nil {
				if value, ok := object["title"].(string); ok && value != "" { title = value }
				if value, ok := object["message"].(string); ok { message = value }
				if value, ok := numberAsInt(object["priority"]); ok { priority = value }
			}
			if title == "" { title = topic }
			if _, err := e.Publish(integration.ApplicationID, title, message, priority); err != nil {
				log.Error().Err(err).Uint("integration_id", integration.ID).Msg("MQTT message could not be published")
			} else {
				e.setIntegrationStatus("mqtt", integration.ID, "connected", "", false, true)
			}
		}
		switch qos {
		case 1:
			if packetID != 0 {
				_, _ = conn.Write([]byte{0x40, 0x02, byte(packetID >> 8), byte(packetID)})
			}
		case 2:
			if packetID != 0 {
				pendingQoS2[packetID] = struct{}{}
				_, _ = conn.Write([]byte{0x50, 0x02, byte(packetID >> 8), byte(packetID)})
			}
		}
	}
}

func dialMQTT(ctx context.Context, raw string) (net.Conn, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host := parsed.Host
	if !strings.Contains(host, ":") {
		if parsed.Scheme == "mqtts" || parsed.Scheme == "tls" {
			host += ":8883"
		} else {
			host += ":1883"
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	switch parsed.Scheme {
	case "mqtts", "tls":
		return tls.DialWithDialer(dialer, "tcp", host, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: parsed.Hostname()})
	case "mqtt", "tcp", "":
		return dialer.DialContext(ctx, "tcp", host)
	default:
		return nil, fmt.Errorf("unsupported MQTT scheme %q", parsed.Scheme)
	}
}

func mqttConnect(conn net.Conn, reader *bufio.Reader, clientID, username, password string) error {
	flags := byte(0x02)
	if username != "" {
		flags |= 0x80
	}
	if password != "" {
		flags |= 0x40
	}
	var variable []byte
	variable = appendMQTTString(variable, "MQTT")
	variable = append(variable, 0x04, flags, 0x00, 0x3c)
	var payload []byte
	payload = appendMQTTString(payload, clientID)
	if username != "" {
		payload = appendMQTTString(payload, username)
	}
	if password != "" {
		payload = appendMQTTString(payload, password)
	}
	packet := []byte{0x10}
	packet = append(packet, encodeRemainingLength(len(variable)+len(payload))...)
	packet = append(packet, variable...)
	packet = append(packet, payload...)
	if _, err := conn.Write(packet); err != nil {
		return err
	}
	header, body, err := readMQTTPacket(reader)
	if err != nil {
		return err
	}
	if header>>4 != 2 || len(body) < 2 || body[1] != 0 {
		return errors.New("MQTT broker rejected connection")
	}
	return nil
}

func mqttSubscribe(conn net.Conn, reader *bufio.Reader, topic string) error {
	if strings.TrimSpace(topic) == "" {
		return errors.New("MQTT topic is required")
	}
	var body []byte
	body = append(body, 0x00, 0x01)
	body = appendMQTTString(body, topic)
	body = append(body, 0x00)
	packet := []byte{0x82}
	packet = append(packet, encodeRemainingLength(len(body))...)
	packet = append(packet, body...)
	if _, err := conn.Write(packet); err != nil {
		return err
	}
	header, response, err := readMQTTPacket(reader)
	if err != nil {
		return err
	}
	if header>>4 != 9 || len(response) < 3 || response[2] == 0x80 {
		return errors.New("MQTT subscription was rejected")
	}
	return nil
}

const maxMQTTPacketBytes = 2 * 1024 * 1024

func readMQTTPacket(reader *bufio.Reader) (byte, []byte, error) {
	header, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	multiplier, remaining := 1, 0
	for i := 0; i < 4; i++ {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, nil, err
		}
		remaining += int(value&127) * multiplier
		if remaining > maxMQTTPacketBytes {
			return 0, nil, fmt.Errorf("MQTT packet exceeds %d byte limit", maxMQTTPacketBytes)
		}
		if value&128 == 0 {
			body := make([]byte, remaining)
			_, err = io.ReadFull(reader, body)
			return header, body, err
		}
		multiplier *= 128
	}
	return 0, nil, errors.New("invalid MQTT remaining length")
}

func decodePublish(header byte, body []byte) (string, []byte, uint16, byte, error) {
	if len(body) < 2 {
		return "", nil, 0, 0, errors.New("invalid MQTT publish packet")
	}
	length := int(binary.BigEndian.Uint16(body[:2]))
	if len(body) < 2+length {
		return "", nil, 0, 0, errors.New("invalid MQTT topic length")
	}
	topic := string(body[2 : 2+length])
	offset := 2 + length
	qos := (header >> 1) & 0x03
	var packetID uint16
	if qos > 0 {
		if len(body) < offset+2 {
			return "", nil, 0, 0, errors.New("invalid MQTT packet id")
		}
		packetID = binary.BigEndian.Uint16(body[offset : offset+2])
		offset += 2
	}
	return topic, body[offset:], packetID, qos, nil
}

func appendMQTTString(target []byte, value string) []byte {
	length := len(value)
	target = append(target, byte(length>>8), byte(length))
	return append(target, []byte(value)...)
}

func encodeRemainingLength(length int) []byte {
	var result []byte
	for {
		value := byte(length % 128)
		length /= 128
		if length > 0 {
			value |= 0x80
		}
		result = append(result, value)
		if length == 0 {
			return result
		}
	}
}

// SendHomeAssistantEvent sends an event through a configured Home Assistant connection.
func (e *Engine) SendHomeAssistantEvent(id uint, eventType string, data map[string]any) error {
	integration, err := e.db.GetHomeAssistantIntegrationByID(id)
	if err != nil {
		return err
	}
	if integration == nil {
		return errors.New("Home Assistant connection not found")
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return errors.New("event type is required")
	}
	base := strings.TrimRight(integration.BaseURL, "/")
	endpoint := base + "/api/events/" + url.PathEscape(eventType)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(e.ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	token, revealErr := security.Reveal(integration.Token)
	if revealErr != nil {
		return revealErr
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Home Assistant returned HTTP %d", response.StatusCode)
	}
	return nil
}

func (e *Engine) runHomeAssistantLoop(ctx context.Context, integration *model.HomeAssistantIntegration) {
	key := fmt.Sprintf("integration:home-assistant:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, key, 45*time.Second, func(leased context.Context) error {
			e.setIntegrationStatus("home-assistant", integration.ID, "connecting", "", false, false)
			return e.runHomeAssistant(leased, integration)
		})
		if ctx.Err() != nil { return }
		if errors.Is(err, errLeaseUnavailable) {
			e.setIntegrationStatus("home-assistant", integration.ID, "standby", "", false, false)
		} else if err != nil {
			e.setIntegrationStatus("home-assistant", integration.ID, "reconnecting", err.Error(), false, false)
			log.Warn().Err(err).Uint("integration_id", integration.ID).Msg("Home Assistant connection interrupted")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (e *Engine) runHomeAssistant(ctx context.Context, integration *model.HomeAssistantIntegration) error {
	parsed, err := url.Parse(integration.BaseURL)
	if err != nil {
		return err
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	default:
		return errors.New("Home Assistant URL must use http or https")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/websocket"

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, parsed.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stopClose:
		}
	}()
	defer close(stopClose)

	var hello map[string]any
	if err := conn.ReadJSON(&hello); err != nil {
		return err
	}
	if hello["type"] != "auth_required" {
		return errors.New("unexpected Home Assistant authentication response")
	}
	if err := conn.WriteJSON(map[string]any{"type":"auth","access_token":integration.Token}); err != nil {
		return err
	}
	var authResponse map[string]any
	if err := conn.ReadJSON(&authResponse); err != nil {
		return err
	}
	if authResponse["type"] != "auth_ok" {
		return errors.New("Home Assistant authentication failed")
	}
	e.setIntegrationStatus("home-assistant", integration.ID, "connected", "", true, false)

	subscribe := map[string]any{"id":1,"type":"subscribe_events"}
	if strings.TrimSpace(integration.EventType) != "" {
		subscribe["event_type"] = integration.EventType
	}
	if err := conn.WriteJSON(subscribe); err != nil {
		return err
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		var envelope map[string]any
		if err := conn.ReadJSON(&envelope); err != nil {
			return err
		}
		if envelope["type"] != "event" {
			continue
		}
		event, ok := envelope["event"].(map[string]any)
		if !ok {
			continue
		}
		eventType, _ := event["event_type"].(string)
		data := event["data"]
		encoded, _ := json.MarshalIndent(data, "", "  ")
		title := integration.Name
		if title == "" {
			title = "Home Assistant"
		}
		if eventType != "" {
			title += ": " + eventType
		}
		if _, err := e.Publish(integration.ApplicationID, title, string(encoded), 0); err != nil {
			log.Error().Err(err).Uint("integration_id", integration.ID).Msg("Home Assistant event could not be published")
		} else {
			e.setIntegrationStatus("home-assistant", integration.ID, "connected", "", false, true)
		}
	}
}

func externalMessage(msg *model.Message) *model.MessageExternal {
	priority := msg.Priority
	return &model.MessageExternal{
		ID: msg.ID,
		ApplicationID: msg.ApplicationID,
		Message: msg.Message,
		Title: msg.Title,
		Priority: &priority,
		Date: msg.Date,
		SenderUserID: msg.SenderUserID,
		SenderName: msg.SenderName,
		Acknowledged: msg.Acknowledged,
		AcknowledgementCount: msg.AckCount,
		AcknowledgedBy: msg.Acknowledgements,
		ParentMessageID: msg.ParentMessageID,
	}
}

func numberAsInt(value any) (int, bool) {
	switch number := value.(type) {
	case float64:
		return int(number), true
	case json.Number:
		parsed, err := number.Int64()
		return int(parsed), err == nil
	case int:
		return number, true
	default:
		return 0, false
	}
}

var errLeaseUnavailable = errors.New("automation lease is held by another instance")

func (e *Engine) runWithLease(parent context.Context, key string, ttl time.Duration, fn func(context.Context) error) error {
	acquired, err := e.db.AcquireAutomationLease(key, e.instanceID, time.Now(), ttl)
	if err != nil { return err }
	if !acquired { return errLeaseUnavailable }
	defer e.db.ReleaseAutomationLease(key, e.instanceID)

	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- fn(ctx) }()

	renewEvery := ttl / 3
	if renewEvery < 5*time.Second { renewEvery = 5*time.Second }
	ticker := time.NewTicker(renewEvery)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-parent.Done():
			cancel()
			return <-done
		case <-ticker.C:
			ok, renewErr := e.db.AcquireAutomationLease(key, e.instanceID, time.Now(), ttl)
			if renewErr != nil || !ok {
				cancel()
				if renewErr != nil { return renewErr }
				return errLeaseUnavailable
			}
		}
	}
}

func (e *Engine) setIntegrationStatus(kind string, objectID uint, state, lastError string, connected, activity bool) {
	status, err := e.db.GetIntegrationStatus(kind, objectID)
	if err != nil {
		log.Warn().Err(err).Str("kind", kind).Uint("integration_id", objectID).Msg("Could not load integration status")
		return
	}
	if status == nil {
		status = &model.IntegrationStatus{Kind:kind, ObjectID:objectID}
	}
	now := time.Now()
	status.State = state
	status.LastError = lastError
	status.UpdatedAt = now
	if connected { status.LastConnectedAt = &now }
	if activity { status.LastActivityAt = &now }
	if err := e.db.SaveIntegrationStatus(status); err != nil {
		log.Warn().Err(err).Str("kind", kind).Uint("integration_id", objectID).Msg("Could not save integration status")
	}
}

func newInstanceID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("instance-%d", time.Now().UnixNano())
	}
	return "instance-" + hex.EncodeToString(raw)
}

func clamp(value, min, max int) int {
	if value < min { return min }
	if value > max { return max }
	return value
}
