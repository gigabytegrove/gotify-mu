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
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gotify/server/v3/model"
	"github.com/rs/zerolog/log"
)

const maxMQTTPacketBytes = 1 << 20

// Notifier delivers a realtime Gotify-compatible message to one user.
type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
}

// Database is the storage contract required by the native integration engine.
type Database interface {
	CreateMessage(message *model.Message) error
	GetMessageByID(id uint) (*model.Message, error)
	GetMessageByAutomationKey(key string) (*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetApplicationRecipientUserIDs(applicationID uint) ([]uint, error)

	GetQuietHoursPolicy(userID uint) (*model.QuietHoursPolicy, error)
	GetDigestPolicy(userID uint) (*model.DigestPolicy, error)
	QueueDigestItem(item *model.DigestItem) error
	GetDigestItems(userID uint) ([]*model.DigestItem, error)
	DeleteDigestItems(userID uint) error
	DeleteDigestItemsByIDs(userID uint, ids []uint) error
	GetDueDigestPolicies(now time.Time) ([]*model.DigestPolicy, error)
	SaveDigestPolicy(item *model.DigestPolicy) error
	ClaimDigestPolicy(id uint, owner string, now, until time.Time) (bool, error)

	GetScheduledNotifications() ([]*model.ScheduledNotification, error)
	GetDueScheduledNotifications(now time.Time) ([]*model.ScheduledNotification, error)
	SaveScheduledNotification(item *model.ScheduledNotification) error
	ClaimScheduledNotification(id uint, owner string, now, until time.Time) (bool, error)

	GetEscalationRulesForMessage(applicationID uint, priority int) ([]*model.EscalationRule, error)
	GetEscalationRuleByID(id uint) (*model.EscalationRule, error)
	QueueEscalation(item *model.EscalationState) error
	GetDueEscalations(now time.Time) ([]*model.EscalationState, error)
	SaveEscalationState(item *model.EscalationState) error
	ClaimEscalation(id uint, owner string, now, until time.Time) (bool, error)
	IsMessageAcknowledged(messageID uint) (bool, error)

	GetMQTTIntegrations() ([]*model.MQTTIntegration, error)
	GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error)
	GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error)
	GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error)
	AcquireAutomationLease(name, owner string, now, until time.Time) (bool, error)
	ReleaseAutomationLease(name, owner string) error
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
	integrationLeader  atomic.Bool
	statusMu           sync.RWMutex
	integrationStatus  map[string]model.IntegrationRuntimeStatus
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
		integrationStatus: make(map[string]model.IntegrationRuntimeStatus),
	}
	e.wg.Add(2)
	go e.schedulerLoop()
	go e.integrationLoop()
	return e
}

func (e *Engine) Close() {
	e.cancel()
	e.stopIntegrations()
	if e.integrationLeader.Load() {
		if err := e.db.ReleaseAutomationLease("native-integrations", e.instanceID); err != nil {
			log.Warn().Err(err).Msg("Could not release native integration lease")
		}
	}
	e.wg.Wait()
}

func integrationStatusKey(kind string, id uint) string {
	return fmt.Sprintf("%s:%d", kind, id)
}

func (e *Engine) setIntegrationStatus(kind string, id uint, state, message string, connected, event, failed bool) {
	now := time.Now().UTC()
	key := integrationStatusKey(kind, id)
	e.statusMu.Lock()
	defer e.statusMu.Unlock()
	status := e.integrationStatus[key]
	status.Type = kind
	status.ID = id
	status.State = state
	status.Message = message
	if connected {
		status.LastConnectedAt = &now
	}
	if event {
		status.LastEventAt = &now
	}
	if failed {
		status.LastErrorAt = &now
	}
	e.integrationStatus[key] = status
}

func (e *Engine) IntegrationStatuses() []model.IntegrationRuntimeStatus {
	e.statusMu.RLock()
	defer e.statusMu.RUnlock()
	result := make([]model.IntegrationRuntimeStatus, 0, len(e.integrationStatus))
	for _, status := range e.integrationStatus {
		result = append(result, status)
	}
	return result
}

func (e *Engine) TestMQTT(id uint) error {
	integration, err := e.db.GetMQTTIntegrationByID(id)
	if err != nil {
		return err
	}
	if integration == nil {
		return errors.New("MQTT connection not found")
	}
	ctx, cancel := context.WithTimeout(e.ctx, 15*time.Second)
	defer cancel()
	conn, err := dialMQTT(ctx, integration.BrokerURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	clientID := integration.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("gotify-mu-test-%d", integration.ID)
	}
	if err := mqttConnect(conn, reader, clientID, integration.Username, integration.Password); err != nil {
		return err
	}
	return mqttSubscribe(conn, reader, integration.Topic)
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
	if msg.AutomationKey != nil && strings.TrimSpace(*msg.AutomationKey) != "" {
		existing, err := e.db.GetMessageByAutomationKey(*msg.AutomationKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			msg.ID = existing.ID
			return externalMessage(existing), nil
		}
	}
	if err := e.db.CreateMessage(msg); err != nil {
		if msg.AutomationKey != nil && strings.TrimSpace(*msg.AutomationKey) != "" {
			existing, lookupErr := e.db.GetMessageByAutomationKey(*msg.AutomationKey)
			if lookupErr == nil && existing != nil {
				msg.ID = existing.ID
				return externalMessage(existing), nil
			}
		}
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
	e.runDue(time.Now())
	for {
		select {
		case <-e.ctx.Done():
			return
		case now := <-ticker.C:
			e.runDue(now)
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
		claimed, err := e.db.ClaimScheduledNotification(item.ID, e.instanceID, now, now.Add(2*time.Minute))
		if err != nil {
			log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Could not claim scheduled notification")
			continue
		}
		if !claimed {
			continue
		}
		dueAt := now
		if item.NextRunAt != nil {
			dueAt = *item.NextRunAt
		}
		key := fmt.Sprintf("schedule:%d:%d", item.ID, dueAt.UTC().UnixNano())
		msg := &model.Message{
			ApplicationID: item.ApplicationID,
			Title: item.Title,
			Message: item.Message,
			Priority: item.Priority,
			Date: now,
			AutomationKey: &key,
		}
		if _, err := e.storeAndDeliver(msg, true); err != nil {
			log.Error().Err(err).Uint("schedule_id", item.ID).Msg("Scheduled notification failed")
			item.ClaimOwner = ""
			item.ClaimUntil = nil
			_ = e.db.SaveScheduledNotification(item)
			continue
		}
		runAt := now
		item.LastRunAt = &runAt
		if item.ScheduleType == "once" {
			item.Enabled = false
			item.NextRunAt = nil
		} else {
			item.NextRunAt = NextScheduleRun(item, now.Add(time.Second))
		}
		item.ClaimOwner = ""
		item.ClaimUntil = nil
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

	switch item.ScheduleType {
	case "once":
		if item.RunAt != nil && item.RunAt.After(now) {
			value := *item.RunAt
			return &value
		}
		return nil
	case "hourly":
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), localNow.Hour(), clamp(item.Minute, 0, 59), 0, 0, loc)
		if !candidate.After(localNow) {
			candidate = candidate.Add(time.Hour)
		}
		value := candidate.UTC()
		return &value
	case "daily":
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), clamp(item.Hour, 0, 23), clamp(item.Minute, 0, 59), 0, 0, loc)
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		value := candidate.UTC()
		return &value
	case "weekly":
		weekday := time.Weekday(clamp(item.Weekday, 0, 6))
		days := (int(weekday) - int(localNow.Weekday()) + 7) % 7
		candidate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), clamp(item.Hour, 0, 23), clamp(item.Minute, 0, 59), 0, 0, loc).AddDate(0, 0, days)
		if !candidate.After(localNow) {
			candidate = candidate.AddDate(0, 0, 7)
		}
		value := candidate.UTC()
		return &value
	default:
		return nil
	}
}

func (e *Engine) runDigests(now time.Time) {
	policies, err := e.db.GetDueDigestPolicies(now)
	if err != nil {
		log.Error().Err(err).Msg("Could not load digest policies")
		return
	}
	for _, policy := range policies {
		claimed, err := e.db.ClaimDigestPolicy(policy.ID, e.instanceID, now, now.Add(2*time.Minute))
		if err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not claim digest")
			continue
		}
		if !claimed {
			continue
		}
		items, err := e.db.GetDigestItems(policy.UserID)
		if err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not load digest items")
			policy.ClaimOwner = ""
			policy.ClaimUntil = nil
			_ = e.db.SaveDigestPolicy(policy)
			continue
		}

		var summary *model.MessageExternal
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
			summary = &model.MessageExternal{
				ID: 0,
				ApplicationID: last.ApplicationID,
				Title: fmt.Sprintf("%d notification digest", len(items)),
				Message: strings.Join(lines, "\n"),
				Priority: &highest,
				Date: now,
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
		policy.ClaimOwner = ""
		policy.ClaimUntil = nil

		// Commit the digest state before realtime delivery so a process crash
		// cannot cause the same digest batch to be sent twice. The underlying
		// notifications remain in normal message history even if delivery is
		// interrupted after this point.
		digestIDs := make([]uint, 0, len(items))
		for _, item := range items {
			digestIDs = append(digestIDs, item.ID)
		}
		if err := e.db.DeleteDigestItemsByIDs(policy.UserID, digestIDs); err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not clear digest")
			continue
		}
		if err := e.db.SaveDigestPolicy(policy); err != nil {
			log.Error().Err(err).Uint("user_id", policy.UserID).Msg("Could not update digest policy")
			continue
		}
		if summary != nil {
			e.notifier.Notify(policy.UserID, summary)
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
		claimed, err := e.db.ClaimEscalation(state.ID, e.instanceID, now, now.Add(2*time.Minute))
		if err != nil {
			log.Error().Err(err).Uint("escalation_id", state.ID).Msg("Could not claim escalation")
			continue
		}
		if !claimed {
			continue
		}
		acknowledged, err := e.db.IsMessageAcknowledged(state.MessageID)
		if err != nil {
			log.Error().Err(err).Uint("message_id", state.MessageID).Msg("Could not inspect acknowledgement")
			state.ClaimOwner = ""
			state.ClaimUntil = nil
			_ = e.db.SaveEscalationState(state)
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
					key := fmt.Sprintf("escalation:%d", state.ID)
					escalated := &model.Message{
						ApplicationID: rule.TargetApplicationID,
						Title: title,
						Message: body,
						Priority: msg.Priority,
						Date: now,
						AutomationKey: &key,
					}
					if _, publishErr := e.storeAndDeliver(escalated, false); publishErr != nil {
						log.Error().Err(publishErr).Uint("rule_id", rule.ID).Msg("Escalation delivery failed")
						state.ClaimOwner = ""
						state.ClaimUntil = nil
						_ = e.db.SaveEscalationState(state)
						continue
					}
					state.Completed = true
				}
			}
		} else {
			state.Completed = true
		}
		done := now
		state.DoneAt = &done
		state.ClaimOwner = ""
		state.ClaimUntil = nil
		if err := e.db.SaveEscalationState(state); err != nil {
			log.Error().Err(err).Uint("escalation_id", state.ID).Msg("Could not complete escalation")
		}
	}
}

func (e *Engine) integrationLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	e.refreshIntegrationLeadership(true)
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.reload:
			if e.integrationLeader.Load() {
				e.restartIntegrations()
			}
		case <-ticker.C:
			e.refreshIntegrationLeadership(false)
		}
	}
}

func (e *Engine) refreshIntegrationLeadership(forceReload bool) {
	now := time.Now()
	acquired, err := e.db.AcquireAutomationLease(
		"native-integrations",
		e.instanceID,
		now,
		now.Add(45*time.Second),
	)
	if err != nil {
		log.Error().Err(err).Msg("Could not renew native integration lease")
		if e.integrationLeader.Load() {
			e.integrationLeader.Store(false)
			e.stopIntegrations()
		}
		return
	}
	if !acquired {
		if e.integrationLeader.Load() {
			e.integrationLeader.Store(false)
			e.stopIntegrations()
		}
		for _, item := range mustMQTT(e.db) {
			if item.Enabled {
				e.setIntegrationStatus("mqtt", item.ID, "standby", "Active on another server instance", false, false, false)
			}
		}
		for _, item := range mustHomeAssistant(e.db) {
			if item.Enabled {
				e.setIntegrationStatus("home-assistant", item.ID, "standby", "Active on another server instance", false, false, false)
			}
		}
		return
	}
	wasLeader := e.integrationLeader.Load()
	e.integrationLeader.Store(true)
	if !wasLeader || forceReload {
		e.restartIntegrations()
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
		e.setIntegrationStatus("mqtt", item.ID, "connecting", "Connecting", false, false, false)
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
		e.setIntegrationStatus("home-assistant", item.ID, "connecting", "Connecting", false, false, false)
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
	for {
		if ctx.Err() != nil {
			return
		}
		if err := e.runMQTT(ctx, integration); err != nil && ctx.Err() == nil {
			e.setIntegrationStatus("mqtt", integration.ID, "error", "Connection interrupted", false, false, true)
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
	e.setIntegrationStatus("mqtt", integration.ID, "connected", "Connected", true, false, false)

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
			e.setIntegrationStatus("mqtt", integration.ID, "connected", "Connected", false, true, false)
		}
		if qos == 1 && packetID != 0 {
			ack := []byte{0x40, 0x02, byte(packetID >> 8), byte(packetID)}
			_, _ = conn.Write(ack)
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
			if remaining > maxMQTTPacketBytes {
				return 0, nil, fmt.Errorf("MQTT packet exceeds %d byte limit", maxMQTTPacketBytes)
			}
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
	if qos > 1 {
		return "", nil, 0, qos, errors.New("MQTT QoS 2 publish packets are not supported")
	}
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
	for {
		if ctx.Err() != nil {
			return
		}
		if err := e.runHomeAssistant(ctx, integration); err != nil && ctx.Err() == nil {
			e.setIntegrationStatus("home-assistant", integration.ID, "error", "Connection interrupted", false, false, true)
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
	e.setIntegrationStatus("home-assistant", integration.ID, "connected", "Connected", true, false, false)

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
			e.setIntegrationStatus("home-assistant", integration.ID, "connected", "Connected", false, true, false)
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
		Acknowledged:         msg.Acknowledged,
		AcknowledgementCount: msg.AcknowledgementCount,
	}
}

func mustMQTT(db Database) []*model.MQTTIntegration {
	items, err := db.GetMQTTIntegrations()
	if err != nil {
		return nil
	}
	return items
}

func mustHomeAssistant(db Database) []*model.HomeAssistantIntegration {
	items, err := db.GetHomeAssistantIntegrations()
	if err != nil {
		return nil
	}
	return items
}

func newInstanceID() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("instance-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw)
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
