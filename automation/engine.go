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
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
)

// Notifier delivers a realtime Gotify-compatible message to one user.
type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
}

// Database is the storage contract required by the native integration engine.
type Database interface {
	CreateMessage(message *model.Message) error
	CreateMessageOnce(message *model.Message) (bool, error)
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
	CreateScheduledNotificationRun(item *model.ScheduledNotificationRun) error
	SaveScheduledNotificationRun(item *model.ScheduledNotificationRun) error

	GetEscalationRulesForMessage(applicationID uint, priority int) ([]*model.EscalationRule, error)
	GetEscalationRuleByID(id uint) (*model.EscalationRule, error)
	ResolveEscalationTargetApplication(rule *model.EscalationRule) (*model.Application, error)
	QueueEscalation(item *model.EscalationState) error
	GetDueEscalations(now time.Time) ([]*model.EscalationState, error)
	SaveEscalationState(item *model.EscalationState) error
	IsMessageAcknowledged(messageID uint) (bool, error)
	QueueDeferredNotification(userID, messageID uint) error
	GetDeferredNotifications() ([]*model.DeferredNotification, error)
	DeleteDeferredNotification(userID, messageID uint) error
	GetOrCreateDigestApplication(userID uint) (*model.Application, error)
	TryAcquireAutomationLease(name, holder string, now time.Time, ttl time.Duration) (bool, error)
	ReleaseAutomationLease(name, holder string) error

	GetMQTTIntegrations() ([]*model.MQTTIntegration, error)
	GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error)
	UpdateMQTTIntegrationStatus(id uint, status string, connectedAt, messageAt *time.Time, lastError string, errorAt *time.Time, incrementReconnect bool) error
	GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error)
	GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error)
	UpdateHomeAssistantIntegrationStatus(id uint, status string, connectedAt, eventAt *time.Time, lastError string, errorAt *time.Time, incrementReconnect bool) error
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
}

func New(db Database, notifier Notifier) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	instanceBytes := make([]byte, 12)
	_, _ = rand.Read(instanceBytes)
	e := &Engine{
		db: db,
		notifier: notifier,
		ctx: ctx,
		cancel: cancel,
		reload: make(chan struct{}, 1),
		instanceID: hex.EncodeToString(instanceBytes),
	}
	e.wg.Add(2)
	go e.schedulerLoop()
	go e.integrationLoop()
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

func (e *Engine) storeAndDeliver(msg *model.Message, allowEscalation bool) (*model.MessageExternal, error) {
	if msg.Date.IsZero() {
		msg.Date = time.Now()
	}
	created, err := e.db.CreateMessageOnce(msg)
	if err != nil { return nil, err }
	external := externalMessage(msg)
	if !created {
		return external, nil
	}
	recipients, err := e.db.GetApplicationRecipientUserIDs(msg.ApplicationID)
	if err != nil {
		return nil, err
	}
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
		if strings.EqualFold(quiet.Mode, "defer") {
			return e.db.QueueDeferredNotification(userID, msg.ID)
		}
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
	if msg.EscalationDepth >= 5 { return nil }
	rules, err := e.db.GetEscalationRulesForMessage(msg.ApplicationID, msg.Priority)
	if err != nil { return err }
	for _, rule := range rules {
		if e.escalationWouldCycle(msg, rule) { continue }
		delay := rule.DelayMinutes
		if delay < 1 { delay = 1 }
		if err := e.db.QueueEscalation(&model.EscalationState{
			RuleID:rule.ID,
			MessageID:msg.ID,
			DueAt:time.Now().Add(time.Duration(delay)*time.Minute),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) escalationWouldCycle(msg *model.Message, rule *model.EscalationRule) bool {
	targetType := strings.ToLower(strings.TrimSpace(rule.TargetType))
	if targetType != "" && targetType != "channel" { return false }
	targetID := rule.TargetApplicationID
	if targetID == 0 { targetID = rule.TargetID }
	if targetID == 0 { return true }

	current := msg
	for depth := 0; current != nil && depth < 8; depth++ {
		if current.ApplicationID == targetID { return true }
		if current.ParentMessageID == 0 { break }
		parent, err := e.db.GetMessageByID(current.ParentMessageID)
		if err != nil || parent == nil { break }
		current = parent
	}
	return false
}

func (e *Engine) schedulerLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	run := func(now time.Time) {
		acquired, err := e.db.TryAcquireAutomationLease("scheduler", e.instanceID, now, 25*time.Second)
		if err != nil {
			log.Error().Err(err).Msg("Could not acquire automation scheduler lease")
			return
		}
		if acquired { e.runDue(now) }
	}
	run(time.Now())
	for {
		select {
		case <-e.ctx.Done():
			_ = e.db.ReleaseAutomationLease("scheduler", e.instanceID)
			return
		case now := <-ticker.C:
			run(now)
		}
	}
}

func (e *Engine) runDue(now time.Time) {
	e.runSchedules(now)
	e.runDigests(now)
	e.runEscalations(now)
	e.runDeferred(now)
}

func (e *Engine) runSchedules(now time.Time) {
	items, err := e.db.GetDueScheduledNotifications(now)
	if err != nil {
		log.Error().Err(err).Msg("Could not load scheduled notifications")
		return
	}
	for _, item := range items {
		scheduledFor := now
		if item.NextRunAt != nil { scheduledFor = *item.NextRunAt }

		run := &model.ScheduledNotificationRun{
			ScheduleID:item.ID,
			ScheduledFor:scheduledFor,
			StartedAt:now,
			Status:"running",
		}
		if err := e.db.CreateScheduledNotificationRun(run); err != nil {
			log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not create schedule run history")
			continue
		}

		misfirePolicy := strings.ToLower(strings.TrimSpace(item.MisfirePolicy))
		if misfirePolicy == "" { misfirePolicy = "send" }
		if misfirePolicy == "skip" && now.Sub(scheduledFor) > 5*time.Minute {
			finished := now
			run.Status = "skipped"
			run.FinishedAt = &finished
			_ = e.db.SaveScheduledNotificationRun(run)
			item.LastStatus = "skipped"
			item.LastError = ""
			item.NextRunAt = NextScheduleRun(item, scheduledFor.Add(time.Second))
			if item.NextRunAt == nil { item.Enabled = false }
			_ = e.db.SaveScheduledNotification(item)
			continue
		}

		msg := &model.Message{
			ApplicationID:item.ApplicationID,
			Title:item.Title,
			Message:item.Message,
			Priority:item.Priority,
			Date:now,
			DeduplicationKey:fmt.Sprintf("schedule:%d:%d", item.ID, scheduledFor.UnixNano()),
		}
		external, publishErr := e.storeAndDeliver(msg, true)
		finished := time.Now()
		run.FinishedAt = &finished
		if publishErr != nil {
			run.Status = "failed"
			run.Error = publishErr.Error()
			item.LastStatus = "failed"
			item.LastError = publishErr.Error()
			retry := now.Add(time.Minute)
			item.NextRunAt = &retry
			_ = e.db.SaveScheduledNotificationRun(run)
			_ = e.db.SaveScheduledNotification(item)
			log.Error().Err(publishErr).Uint("schedule_id", item.ID).Msg("Scheduled notification failed")
			continue
		}
		run.Status = "completed"
		if external != nil { run.MessageID = external.ID }
		_ = e.db.SaveScheduledNotificationRun(run)

		runAt := now
		item.LastRunAt = &runAt
		item.RunCount++
		item.LastStatus = "completed"
		item.LastError = ""
		if item.ScheduleType == "once" || (item.MaxRuns > 0 && item.RunCount >= item.MaxRuns) {
			item.Enabled = false
			item.NextRunAt = nil
		} else {
			item.NextRunAt = NextScheduleRun(item, scheduledFor.Add(time.Second))
			if item.NextRunAt == nil { item.Enabled = false }
		}
		if err := e.db.SaveScheduledNotification(item); err != nil {
			log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not update schedule")
		}
	}
}

func scheduleLocation(item *model.ScheduledNotification) *time.Location {
	loc := time.UTC
	if item.Timezone != "" {
		if parsed, err := time.LoadLocation(item.Timezone); err == nil { loc = parsed }
	}
	return loc
}

func excludedScheduleDate(item *model.ScheduledNotification, candidate time.Time) bool {
	if strings.TrimSpace(item.ExcludedDates) == "" { return false }
	localDate := candidate.In(scheduleLocation(item)).Format("2006-01-02")
	for _, raw := range strings.FieldsFunc(item.ExcludedDates, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == ' ' || r == '\t'
	}) {
		if strings.TrimSpace(raw) == localDate { return true }
	}
	return false
}

func scheduleWithinLimits(item *model.ScheduledNotification, candidate time.Time) bool {
	if item.MaxRuns > 0 && item.RunCount >= item.MaxRuns { return false }
	if item.EndAt != nil && candidate.After(*item.EndAt) { return false }
	return true
}

func nextScheduleCandidate(item *model.ScheduledNotification, now time.Time) *time.Time {
	loc := scheduleLocation(item)
	localNow := now.In(loc)

	switch item.ScheduleType {
	case "once":
		if item.RunAt != nil && item.RunAt.After(now) {
			value := *item.RunAt
			return &value
		}
		return nil
	case "hourly":
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), localNow.Hour(), clamp(item.Minute, 0, 59), 0, 0, loc)
		if !candidate.After(localNow) { candidate = candidate.Add(time.Hour) }
		value := candidate.UTC()
		return &value
	case "daily":
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), clamp(item.Hour, 0, 23), clamp(item.Minute, 0, 59), 0, 0, loc)
		if !candidate.After(localNow) { candidate = candidate.AddDate(0, 0, 1) }
		value := candidate.UTC()
		return &value
	case "weekly":
		weekday := time.Weekday(clamp(item.Weekday, 0, 6))
		days := (int(weekday) - int(localNow.Weekday()) + 7) % 7
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), clamp(item.Hour, 0, 23), clamp(item.Minute, 0, 59), 0, 0, loc).AddDate(0, 0, days)
		if !candidate.After(localNow) { candidate = candidate.AddDate(0, 0, 7) }
		value := candidate.UTC()
		return &value
	case "cron":
		schedule, err := cron.Parse(strings.TrimSpace(item.CronExpression))
		if err != nil { return nil }
		candidate := schedule.Next(localNow)
		value := candidate.UTC()
		return &value
	default:
		return nil
	}
}

func NextScheduleRun(item *model.ScheduledNotification, now time.Time) *time.Time {
	searchFrom := now
	for attempts := 0; attempts < 10000; attempts++ {
		candidate := nextScheduleCandidate(item, searchFrom)
		if candidate == nil || !scheduleWithinLimits(item, *candidate) { return nil }
		if !excludedScheduleDate(item, *candidate) { return candidate }
		searchFrom = candidate.Add(time.Second)
	}
	return nil
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
				if item.Priority > highest { highest = item.Priority }
				line := item.Title
				if strings.TrimSpace(line) == "" { line = item.Message }
				if len(line) > 120 { line = line[:117] + "..." }
				lines = append(lines, "• "+line)
			}
			app, appErr := e.db.GetOrCreateDigestApplication(policy.UserID)
			if appErr != nil {
				log.Error().Err(appErr).Uint("user_id", policy.UserID).Msg("Could not prepare digest history")
				continue
			}
			runKey := now.UnixNano()
			if policy.NextRunAt != nil { runKey = policy.NextRunAt.UnixNano() }
			summary := &model.Message{
				ApplicationID:app.ID,
				Title:fmt.Sprintf("%d notification digest", len(items)),
				Message:strings.Join(lines, "\n"),
				Priority:highest,
				Date:now,
				DeduplicationKey:fmt.Sprintf("digest:%d:%d", policy.UserID, runKey),
			}
			created, storeErr := e.db.CreateMessageOnce(summary)
			if storeErr != nil {
				log.Error().Err(storeErr).Uint("user_id", policy.UserID).Msg("Could not store digest")
				continue
			}
			if created {
				quiet, quietErr := e.db.GetQuietHoursPolicy(policy.UserID)
				if quietErr != nil {
					log.Error().Err(quietErr).Uint("user_id", policy.UserID).Msg("Could not evaluate digest quiet hours")
				} else if quiet == nil || !quiet.Enabled || highest >= quiet.AllowPriority || !quietNow(quiet, now) {
					e.notifier.Notify(policy.UserID, externalMessage(summary))
				} else if strings.EqualFold(quiet.Mode, "defer") {
					_ = e.db.QueueDeferredNotification(policy.UserID, summary.ID)
				}
			}
			if err := e.db.DeleteDigestItems(policy.UserID); err != nil {
				log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not clear digest")
				continue
			}
		}
		lastSent := now
		policy.LastSentAt = &lastSent
		interval := policy.IntervalMinutes
		if interval < 15 { interval = 15 }
		next := now.Add(time.Duration(interval) * time.Minute)
		policy.NextRunAt = &next
		if err := e.db.SaveDigestPolicy(policy); err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not update digest policy")
		}
	}
}

func (e *Engine) runDeferred(now time.Time) {
	items, err := e.db.GetDeferredNotifications()
	if err != nil {
		log.Error().Err(err).Msg("Could not load deferred notifications")
		return
	}
	for _, item := range items {
		quiet, quietErr := e.db.GetQuietHoursPolicy(item.UserID)
		if quietErr != nil {
			log.Error().Err(quietErr).Uint("user_id", item.UserID).Msg("Could not inspect deferred Quiet Hours")
			continue
		}
		if quiet != nil && quiet.Enabled && quietNow(quiet, now) { continue }
		msg, msgErr := e.db.GetMessageByID(item.MessageID)
		if msgErr != nil {
			log.Error().Err(msgErr).Uint("message_id", item.MessageID).Msg("Could not load deferred message")
			continue
		}
		if msg != nil { e.notifier.Notify(item.UserID, externalMessage(msg)) }
		if err := e.db.DeleteDeferredNotification(item.UserID, item.MessageID); err != nil {
			log.Error().Err(err).Uint("message_id", item.MessageID).Msg("Could not clear deferred notification")
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
		source, loadErr := e.db.GetMessageByID(state.MessageID)
		if loadErr != nil || source == nil {
			state.Completed = true
			done := now
			state.DoneAt = &done
			_ = e.db.SaveEscalationState(state)
			continue
		}

		rootID := source.RootMessageID
		if rootID == 0 { rootID = source.ID }
		acknowledged, ackErr := e.db.IsMessageAcknowledged(source.ID)
		if ackErr == nil && rootID != source.ID && !acknowledged {
			acknowledged, ackErr = e.db.IsMessageAcknowledged(rootID)
		}
		if ackErr != nil {
			log.Error().Err(ackErr).Uint("message_id", state.MessageID).Msg("Could not inspect acknowledgement")
			continue
		}
		if acknowledged {
			state.Completed = true
			done := now
			state.DoneAt = &done
			_ = e.db.SaveEscalationState(state)
			continue
		}

		rule, ruleErr := e.db.GetEscalationRuleByID(state.RuleID)
		if ruleErr != nil || rule == nil || !rule.Enabled {
			state.Completed = true
			done := now
			state.DoneAt = &done
			_ = e.db.SaveEscalationState(state)
			continue
		}

		targetApp, targetErr := e.db.ResolveEscalationTargetApplication(rule)
		if targetErr != nil || targetApp == nil {
			if targetErr == nil { targetErr = errors.New("escalation target is unavailable") }
			log.Error().Err(targetErr).Uint("rule_id", rule.ID).Msg("Escalation target could not be resolved")
			continue
		}

		title := source.Title
		if title == "" { title = "Escalated notification" } else { title = "Escalated: " + title }
		body := source.Message + "\n\nThis notification was escalated because it was not acknowledged."
		child := &model.Message{
			ApplicationID:targetApp.ID,
			Title:title,
			Message:body,
			Priority:source.Priority,
			Date:now,
			ParentMessageID:source.ID,
			RootMessageID:rootID,
			EscalationRuleID:rule.ID,
			EscalationDepth:source.EscalationDepth+1,
			DeduplicationKey:fmt.Sprintf("escalation:%d:%d", state.ID, state.RepeatCount),
		}
		allowChain := state.RepeatCount == 0
		external, publishErr := e.storeAndDeliver(child, allowChain)
		if publishErr != nil {
			log.Error().Err(publishErr).Uint("rule_id", rule.ID).Msg("Escalation delivery failed")
			continue
		}
		if external != nil { state.LastEscalatedMessageID = external.ID }
		state.RepeatCount++

		repeatMinutes := rule.RepeatMinutes
		if repeatMinutes > 0 && state.RepeatCount <= rule.MaxRepeats {
			state.DueAt = now.Add(time.Duration(repeatMinutes) * time.Minute)
			state.Completed = false
			state.DoneAt = nil
		} else {
			state.Completed = true
			done := now
			state.DoneAt = &done
		}
		if err := e.db.SaveEscalationState(state); err != nil {
			log.Error().Err(err).Uint("escalation_id", state.ID).Msg("Could not update escalation")
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

	e.integrationMu.Lock()
	defer e.integrationMu.Unlock()
	for _, item := range mqttItems {
		if !item.Enabled {
			continue
		}
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.MQTTIntegration) {
			defer e.integrationWG.Done()
			e.runMQTTLoop(ctx, &integration)
		}(*item)
	}
	for _, item := range haItems {
		if !item.Enabled {
			continue
		}
		ctx, cancel := context.WithCancel(e.ctx)
		e.integrationCancels = append(e.integrationCancels, cancel)
		e.integrationWG.Add(1)
		go func(integration model.HomeAssistantIntegration) {
			defer e.integrationWG.Done()
			e.runHomeAssistantLoop(ctx, &integration)
		}(*item)
	}
}

func (e *Engine) runMQTTLoop(ctx context.Context, integration *model.MQTTIntegration) {
	leaseName := fmt.Sprintf("mqtt:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, leaseName, func(leaseCtx context.Context) error {
			return e.runMQTT(leaseCtx, integration)
		})
		if err != nil && !errors.Is(err, errLeaseUnavailable) && ctx.Err() == nil {
			now := time.Now()
			_ = e.db.UpdateMQTTIntegrationStatus(integration.ID, "reconnecting", nil, nil, err.Error(), &now, true)
			log.Warn().Err(err).Uint("integration_id", integration.ID).Msg("MQTT connection interrupted")
		}
		select {
		case <-ctx.Done(): return
		case <-time.After(10 * time.Second):
		}
	}
}

var errLeaseUnavailable = errors.New("automation lease unavailable")

func (e *Engine) runWithLease(ctx context.Context, name string, work func(context.Context) error) error {
	acquired, err := e.db.TryAcquireAutomationLease(name, e.instanceID, time.Now(), 30*time.Second)
	if err != nil { return err }
	if !acquired { return errLeaseUnavailable }

	leaseCtx, cancel := context.WithCancel(ctx)
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-stop:
				return
			case now := <-ticker.C:
				ok, leaseErr := e.db.TryAcquireAutomationLease(name, e.instanceID, now, 30*time.Second)
				if leaseErr != nil || !ok {
					cancel()
					return
				}
			}
		}
	}()

	workErr := work(leaseCtx)
	close(stop)
	cancel()
	_ = e.db.ReleaseAutomationLease(name, e.instanceID)
	return workErr
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
	connectedAt := time.Now()
	_ = e.db.UpdateMQTTIntegrationStatus(integration.ID, "connected", &connectedAt, nil, "", nil, false)

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
		if packetType != 3 {
			continue
		}
		topic, payload, packetID, qos, err := decodePublish(header, body)
		if err != nil {
			return err
		}
		title, message, priority := integration.Name, string(payload), 0
		var object map[string]any
		if json.Unmarshal(payload, &object) == nil {
			if value, ok := object["title"].(string); ok && value != "" {
				title = value
			}
			if value, ok := object["message"].(string); ok {
				message = value
			}
			if value, ok := numberAsInt(object["priority"]); ok {
				priority = value
			}
		}
		if title == "" {
			title = topic
		}
		if _, err := e.Publish(integration.ApplicationID, title, message, priority); err != nil {
			log.Error().Err(err).Uint("integration_id", integration.ID).Msg("MQTT message could not be published")
		} else {
			messageAt := time.Now()
			_ = e.db.UpdateMQTTIntegrationStatus(integration.ID, "connected", nil, &messageAt, "", nil, false)
		}
		if qos == 1 && packetID != 0 {
			ack := []byte{0x40, 0x02, byte(packetID >> 8), byte(packetID)}
			_, _ = conn.Write(ack)
		}
	}
}

// TestMQTTConnection verifies broker authentication and topic subscription without consuming notifications.
func (e *Engine) TestMQTTConnection(id uint) error {
	integration, err := e.db.GetMQTTIntegrationByID(id)
	if err != nil { return err }
	if integration == nil { return errors.New("MQTT connection not found") }
	ctx, cancel := context.WithTimeout(e.ctx, 15*time.Second)
	defer cancel()
	conn, err := dialMQTT(ctx, integration.BrokerURL)
	if err != nil { return err }
	defer conn.Close()
	reader := bufio.NewReader(conn)
	clientID := integration.ClientID
	if clientID == "" { clientID = fmt.Sprintf("gotify-mu-test-%d", integration.ID) }
	if err := mqttConnect(conn, reader, clientID, integration.Username, integration.Password); err != nil { return err }
	if err := mqttSubscribe(conn, reader, integration.Topic); err != nil { return err }
	return nil
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
	request.Header.Set("Authorization", "Bearer "+integration.Token)
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
	leaseName := fmt.Sprintf("home-assistant:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, leaseName, func(leaseCtx context.Context) error {
			return e.runHomeAssistant(leaseCtx, integration)
		})
		if err != nil && !errors.Is(err, errLeaseUnavailable) && ctx.Err() == nil {
			now := time.Now()
			_ = e.db.UpdateHomeAssistantIntegrationStatus(integration.ID, "reconnecting", nil, nil, err.Error(), &now, true)
			log.Warn().Err(err).Uint("integration_id", integration.ID).Msg("Home Assistant connection interrupted")
		}
		select {
		case <-ctx.Done(): return
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
	connectedAt := time.Now()
	_ = e.db.UpdateHomeAssistantIntegrationStatus(integration.ID, "connected", &connectedAt, nil, "", nil, false)

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
			eventAt := time.Now()
			_ = e.db.UpdateHomeAssistantIntegrationStatus(integration.ID, "connected", nil, &eventAt, "", nil, false)
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
		ParentMessageID: msg.ParentMessageID,
		RootMessageID: msg.RootMessageID,
		EscalationRuleID: msg.EscalationRuleID,
		EscalationDepth: msg.EscalationDepth,
		Acknowledged: msg.Acknowledged,
		AcknowledgedByAnyone: msg.AcknowledgedByAnyone,
		AcknowledgementCount: msg.AcknowledgementCount,
		LastAcknowledgedBy: msg.LastAcknowledgedBy,
		LastAcknowledgedAt: msg.LastAcknowledgedAt,
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

func clamp(value, min, max int) int {
	if value < min { return min }
	if value > max { return max }
	return value
}
