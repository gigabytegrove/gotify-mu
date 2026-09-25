package database

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetWebhookRoutes() ([]*model.WebhookRoute, error) {
	var items []*model.WebhookRoute
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		plain, err := d.Secrets.Decrypt(item.Secret)
		if err != nil { return nil, err }
		item.Secret = plain
	}
	return items, nil
}

func (d *GormDatabase) GetWebhookRouteByID(id uint) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	plain, err := d.Secrets.Decrypt(item.Secret)
	if err != nil { return nil, err }
	item.Secret = plain
	return item, nil
}

func (d *GormDatabase) GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	hash := security.HashSecret(secret)
	if err := d.DB.Where("secret_hash = ? AND enabled = ?", hash, true).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	decrypted, err := d.Secrets.Decrypt(item.Secret)
	if err != nil { return nil, err }
	item.Secret = decrypted
	return item, nil
}

func (d *GormDatabase) SaveWebhookRoute(item *model.WebhookRoute) error {
	plain, err := d.Secrets.Decrypt(item.Secret)
	if err != nil { return err }
	item.SecretHash = security.HashSecret(plain)
	encrypted, err := d.Secrets.Encrypt(plain)
	if err != nil { return err }
	item.Secret = encrypted
	err = d.DB.Save(item).Error
	item.Secret = plain
	return err
}
func (d *GormDatabase) DeleteWebhookRoute(id uint) error { return d.DB.Delete(&model.WebhookRoute{}, id).Error }

func (d *GormDatabase) GetMQTTIntegrations() ([]*model.MQTTIntegration, error) {
	var items []*model.MQTTIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		plain, err := d.Secrets.Decrypt(item.Password)
		if err != nil { return nil, err }
		item.Password = plain
	}
	return items, nil
}
func (d *GormDatabase) GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error) {
	item := new(model.MQTTIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return nil, err }
	item.Password = plain
	return item, nil
}
func (d *GormDatabase) SaveMQTTIntegration(item *model.MQTTIntegration) error {
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return err }
	encrypted, err := d.Secrets.Encrypt(plain)
	if err != nil { return err }
	item.Password = encrypted
	err = d.DB.Save(item).Error
	item.Password = plain
	return err
}
func (d *GormDatabase) DeleteMQTTIntegration(id uint) error { return d.DB.Delete(&model.MQTTIntegration{}, id).Error }

func (d *GormDatabase) GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error) {
	var items []*model.HomeAssistantIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		plain, err := d.Secrets.Decrypt(item.Token)
		if err != nil { return nil, err }
		item.Token = plain
	}
	return items, nil
}
func (d *GormDatabase) GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error) {
	item := new(model.HomeAssistantIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	plain, err := d.Secrets.Decrypt(item.Token)
	if err != nil { return nil, err }
	item.Token = plain
	return item, nil
}
func (d *GormDatabase) SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error {
	plain, err := d.Secrets.Decrypt(item.Token)
	if err != nil { return err }
	encrypted, err := d.Secrets.Encrypt(plain)
	if err != nil { return err }
	item.Token = encrypted
	err = d.DB.Save(item).Error
	item.Token = plain
	return err
}
func (d *GormDatabase) DeleteHomeAssistantIntegration(id uint) error { return d.DB.Delete(&model.HomeAssistantIntegration{}, id).Error }

func (d *GormDatabase) GetScheduledNotifications() ([]*model.ScheduledNotification, error) {
	var items []*model.ScheduledNotification
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetScheduledNotificationByID(id uint) (*model.ScheduledNotification, error) {
	item := new(model.ScheduledNotification)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveScheduledNotification(item *model.ScheduledNotification) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteScheduledNotification(id uint) error { return d.DeleteScheduledNotificationWithRuns(id) }
func (d *GormDatabase) GetDueScheduledNotifications(now time.Time) ([]*model.ScheduledNotification, error) {
	var items []*model.ScheduledNotification
	return items, d.DB.Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).Find(&items).Error
}

func (d *GormDatabase) GetQuietHoursPolicy(userID uint) (*model.QuietHoursPolicy, error) {
	item := new(model.QuietHoursPolicy)
	if err := d.DB.Where("user_id = ?", userID).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveQuietHoursPolicy(item *model.QuietHoursPolicy) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled","start_minute","end_minute","timezone","allow_priority","mode","updated_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) GetDigestPolicy(userID uint) (*model.DigestPolicy, error) {
	item := new(model.DigestPolicy)
	if err := d.DB.Where("user_id = ?", userID).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveDigestPolicy(item *model.DigestPolicy) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled","interval_minutes","immediate_priority","last_sent_at","next_run_at","updated_at"}),
	}).Create(item).Error
}
func (d *GormDatabase) QueueDigestItem(item *model.DigestItem) error {
	return d.DB.Clauses(clause.OnConflict{DoNothing:true}).Create(item).Error
}
func (d *GormDatabase) GetDigestItems(userID uint) ([]*model.DigestItem, error) {
	var items []*model.DigestItem
	return items, d.DB.Where("user_id = ?", userID).Order("created_at asc").Find(&items).Error
}
func (d *GormDatabase) DeleteDigestItems(userID uint) error { return d.DB.Where("user_id = ?", userID).Delete(&model.DigestItem{}).Error }
func (d *GormDatabase) GetDueDigestPolicies(now time.Time) ([]*model.DigestPolicy, error) {
	var items []*model.DigestPolicy
	return items, d.DB.Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).Find(&items).Error
}

func (d *GormDatabase) GetEscalationRules() ([]*model.EscalationRule, error) {
	var items []*model.EscalationRule
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetEscalationRuleByID(id uint) (*model.EscalationRule, error) {
	item := new(model.EscalationRule)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveEscalationRule(item *model.EscalationRule) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteEscalationRule(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("rule_id = ?", id).Delete(&model.EscalationState{}).Error; err != nil { return err }
		return tx.Delete(&model.EscalationRule{}, id).Error
	})
}
func (d *GormDatabase) GetEscalationRulesForMessage(applicationID uint, priority int) ([]*model.EscalationRule, error) {
	var items []*model.EscalationRule
	return items, d.DB.Where("enabled = ? AND source_application_id = ? AND min_priority <= ?", true, applicationID, priority).Find(&items).Error
}
func (d *GormDatabase) QueueEscalation(item *model.EscalationState) error {
	return d.DB.Clauses(clause.OnConflict{DoNothing:true}).Create(item).Error
}
func (d *GormDatabase) GetDueEscalations(now time.Time) ([]*model.EscalationState, error) {
	var items []*model.EscalationState
	return items, d.DB.Where("completed = ? AND due_at <= ?", false, now).Find(&items).Error
}
func (d *GormDatabase) SaveEscalationState(item *model.EscalationState) error { return d.DB.Save(item).Error }

func (d *GormDatabase) SetMessageAcknowledgement(userID, messageID uint, acknowledged bool, now time.Time) error {
	if !acknowledged {
		return d.DB.Where("user_id = ? AND message_id = ?", userID, messageID).Delete(&model.MessageAcknowledgement{}).Error
	}
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"user_id"},{Name:"message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"acknowledged_at"}),
	}).Create(&model.MessageAcknowledgement{UserID:userID,MessageID:messageID,AcknowledgedAt:now}).Error
}
func (d *GormDatabase) IsMessageAcknowledgedByUser(userID, messageID uint) (bool, error) {
	var count int64
	err := d.DB.Model(&model.MessageAcknowledgement{}).Where("user_id = ? AND message_id = ?", userID, messageID).Count(&count).Error
	return count > 0, err
}
func (d *GormDatabase) IsMessageAcknowledged(messageID uint) (bool, error) {
	var count int64
	err := d.DB.Model(&model.MessageAcknowledgement{}).Where("message_id = ?", messageID).Count(&count).Error
	return count > 0, err
}
func (d *GormDatabase) DeleteMessageAcknowledgements(messageID uint) error {
	return d.DB.Where("message_id = ?", messageID).Delete(&model.MessageAcknowledgement{}).Error
}


func (d *GormDatabase) migrateIntegrationSecrets() error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var webhooks []*model.WebhookRoute
		if err := tx.Find(&webhooks).Error; err != nil { return err }
		for _, item := range webhooks {
			plain, err := d.Secrets.Decrypt(item.Secret)
			if err != nil { return err }
			encrypted, err := d.Secrets.Encrypt(plain)
			if err != nil { return err }
			hash := security.HashSecret(plain)
			if err := tx.Model(item).Updates(map[string]any{"secret": encrypted, "secret_hash": hash}).Error; err != nil { return err }
		}

		var mqtt []*model.MQTTIntegration
		if err := tx.Find(&mqtt).Error; err != nil { return err }
		for _, item := range mqtt {
			plain, err := d.Secrets.Decrypt(item.Password)
			if err != nil { return err }
			encrypted, err := d.Secrets.Encrypt(plain)
			if err != nil { return err }
			if err := tx.Model(item).Update("password", encrypted).Error; err != nil { return err }
		}

		var home []*model.HomeAssistantIntegration
		if err := tx.Find(&home).Error; err != nil { return err }
		for _, item := range home {
			plain, err := d.Secrets.Decrypt(item.Token)
			if err != nil { return err }
			encrypted, err := d.Secrets.Encrypt(plain)
			if err != nil { return err }
			if err := tx.Model(item).Update("token", encrypted).Error; err != nil { return err }
		}
		return nil
	})
}


func (d *GormDatabase) QueueDeferredNotification(userID, messageID uint) error {
	return d.DB.Clauses(clause.OnConflict{DoNothing:true}).Create(&model.DeferredNotification{
		UserID:userID, MessageID:messageID,
	}).Error
}

func (d *GormDatabase) GetDeferredNotifications() ([]*model.DeferredNotification, error) {
	var items []*model.DeferredNotification
	return items, d.DB.Order("created_at asc").Find(&items).Error
}

func (d *GormDatabase) DeleteDeferredNotification(userID, messageID uint) error {
	return d.DB.Where("user_id = ? AND message_id = ?", userID, messageID).Delete(&model.DeferredNotification{}).Error
}

func (d *GormDatabase) TryAcquireAutomationLease(name, holder string, now time.Time, ttl time.Duration) (bool, error) {
	expires := now.Add(ttl)
	result := d.DB.Model(&model.AutomationLease{}).
		Where("name = ? AND (holder = ? OR expires_at <= ?)", name, holder, now).
		Updates(map[string]any{"holder":holder, "expires_at":expires, "updated_at":now})
	if result.Error != nil { return false, result.Error }
	if result.RowsAffected > 0 { return true, nil }

	lease := &model.AutomationLease{Name:name, Holder:holder, ExpiresAt:expires, UpdatedAt:now}
	err := d.DB.Create(lease).Error
	if err == nil { return true, nil }
	if errors.Is(err, gorm.ErrDuplicatedKey) { return false, nil }
	return false, err
}

func (d *GormDatabase) ReleaseAutomationLease(name, holder string) error {
	return d.DB.Where("name = ? AND holder = ?", name, holder).Delete(&model.AutomationLease{}).Error
}

func (d *GormDatabase) GetMessageAcknowledgements(messageID uint) ([]*model.MessageAcknowledgementView, error) {
	var rows []model.MessageAcknowledgementView
	err := d.DB.Table("message_acknowledgements AS ma").
		Select("ma.user_id, users.name AS username, users.display_name, ma.acknowledged_at").
		Joins("LEFT JOIN users ON users.id = ma.user_id").
		Where("ma.message_id = ?", messageID).
		Order("ma.acknowledged_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (d *GormDatabase) GetOrCreateDigestApplication(userID uint) (*model.Application, error) {
	app := new(model.Application)
	err := d.DB.Where("user_id = ? AND internal = ? AND name = ?", userID, true, "Notification Digest").First(app).Error
	if err == nil { return app, nil }
	if !errors.Is(err, gorm.ErrRecordNotFound) { return nil, err }

	publicToken, _ := auth.GenerateApplicationToken()
	app = &model.Application{
		UserID:userID,
		Name:"Notification Digest",
		Description:"Periodic notification summaries",
		Token:publicToken,
		Internal:true,
	}
	if err := d.CreateApplication(app); err != nil {
		// A concurrent worker may have created the same logical app. Prefer the
		// existing one when it can be found.
		existing := new(model.Application)
		if findErr := d.DB.Where("user_id = ? AND internal = ? AND name = ?", userID, true, "Notification Digest").First(existing).Error; findErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("create digest application: %w", err)
	}
	return app, nil
}


func (d *GormDatabase) UpdateMQTTIntegrationStatus(id uint, status string, connectedAt, messageAt *time.Time, lastError string, errorAt *time.Time, incrementReconnect bool) error {
	updates := map[string]any{"status":status, "last_error":lastError, "last_error_at":errorAt}
	if connectedAt != nil { updates["last_connected_at"] = connectedAt }
	if messageAt != nil { updates["last_message_at"] = messageAt }
	if incrementReconnect { updates["reconnect_count"] = gorm.Expr("reconnect_count + ?", 1) }
	return d.DB.Model(&model.MQTTIntegration{}).Where("id = ?", id).Updates(updates).Error
}

func (d *GormDatabase) UpdateHomeAssistantIntegrationStatus(id uint, status string, connectedAt, eventAt *time.Time, lastError string, errorAt *time.Time, incrementReconnect bool) error {
	updates := map[string]any{"status":status, "last_error":lastError, "last_error_at":errorAt}
	if connectedAt != nil { updates["last_connected_at"] = connectedAt }
	if eventAt != nil { updates["last_event_at"] = eventAt }
	if incrementReconnect { updates["reconnect_count"] = gorm.Expr("reconnect_count + ?", 1) }
	return d.DB.Model(&model.HomeAssistantIntegration{}).Where("id = ?", id).Updates(updates).Error
}


func (d *GormDatabase) CreateScheduledNotificationRun(item *model.ScheduledNotificationRun) error {
	return d.DB.Create(item).Error
}

func (d *GormDatabase) SaveScheduledNotificationRun(item *model.ScheduledNotificationRun) error {
	return d.DB.Save(item).Error
}

func (d *GormDatabase) GetScheduledNotificationRuns(scheduleID uint, limit int) ([]*model.ScheduledNotificationRun, error) {
	if limit <= 0 || limit > 200 { limit = 50 }
	var items []*model.ScheduledNotificationRun
	err := d.DB.Where("schedule_id = ?", scheduleID).
		Order("scheduled_for desc, id desc").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (d *GormDatabase) DeleteScheduledNotificationRuns(scheduleID uint) error {
	return d.DB.Where("schedule_id = ?", scheduleID).Delete(&model.ScheduledNotificationRun{}).Error
}

func (d *GormDatabase) DeleteScheduledNotificationWithRuns(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("schedule_id = ?", id).Delete(&model.ScheduledNotificationRun{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ScheduledNotification{}, id).Error
	})
}


func (d *GormDatabase) ResolveEscalationTargetApplication(rule *model.EscalationRule) (*model.Application, error) {
	targetType := strings.ToLower(strings.TrimSpace(rule.TargetType))
	if targetType == "" || targetType == "channel" {
		targetID := rule.TargetApplicationID
		if targetID == 0 { targetID = rule.TargetID }
		return d.GetApplicationByID(targetID)
	}
	if targetType != "user" && targetType != "group" {
		return nil, fmt.Errorf("unsupported escalation target type %q", rule.TargetType)
	}
	if rule.TargetID == 0 {
		return nil, errors.New("escalation target is required")
	}

	mapping := new(model.EscalationTargetApplication)
	err := d.DB.Where("target_type = ? AND target_id = ?", targetType, rule.TargetID).First(mapping).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { return nil, err }

	var app *model.Application
	if err == nil {
		app, err = d.GetApplicationByID(mapping.ApplicationID)
		if err != nil { return nil, err }
	}

	if app == nil {
		name := "Escalations"
		description := "Escalated notifications"
		ownerID := uint(0)
		if targetType == "user" {
			user, userErr := d.GetUserByID(rule.TargetID)
			if userErr != nil { return nil, userErr }
			if user == nil { return nil, errors.New("escalation user not found") }
			ownerID = user.ID
			name = "Escalations · " + user.Name
			description = "Escalated notifications for " + user.Name
		} else {
			group, groupErr := d.GetUserGroupByID(rule.TargetID)
			if groupErr != nil { return nil, groupErr }
			if group == nil { return nil, errors.New("escalation Group not found") }
			name = "Escalations · " + group.Name
			description = "Escalated notifications for Group " + group.Name
		}

		publicToken, _ := auth.GenerateApplicationToken()
		app = &model.Application{
			UserID:ownerID,
			Name:name,
			Description:description,
			Token:publicToken,
			Internal:true,
		}
		if err := d.CreateApplication(app); err != nil { return nil, err }
		mapping = &model.EscalationTargetApplication{
			TargetType:targetType,
			TargetID:rule.TargetID,
			ApplicationID:app.ID,
		}
		if err := d.DB.Save(mapping).Error; err != nil {
			_ = d.DeleteApplicationByID(app.ID)
			return nil, err
		}
	}

	if targetType == "user" {
		if err := d.UpsertApplicationMembership(&model.ApplicationMembership{
			ApplicationID:app.ID,
			UserID:rule.TargetID,
			ReceiveNotifications:true,
		}); err != nil { return nil, err }
		return app, nil
	}

	members, err := d.GetUserGroupMembers(rule.TargetID)
	if err != nil { return nil, err }
	desired := make(map[uint]struct{}, len(members))
	for _, user := range members {
		desired[user.ID] = struct{}{}
		if err := d.UpsertApplicationMembership(&model.ApplicationMembership{
			ApplicationID:app.ID,
			UserID:user.ID,
			ReceiveNotifications:true,
		}); err != nil { return nil, err }
	}
	current, err := d.GetApplicationMemberships(app.ID)
	if err != nil { return nil, err }
	for _, membership := range current {
		if _, ok := desired[membership.UserID]; !ok {
			if err := d.DeleteApplicationMembership(app.ID, membership.UserID); err != nil { return nil, err }
		}
	}
	return app, nil
}

func (d *GormDatabase) DeleteEscalationTargetApplication(targetType string, targetID uint) error {
	mapping := new(model.EscalationTargetApplication)
	err := d.DB.Where("target_type = ? AND target_id = ?", targetType, targetID).First(mapping).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil }
	if err != nil { return err }
	if err := d.DB.Where("target_type = ? AND target_id = ?", targetType, targetID).
		Delete(&model.EscalationTargetApplication{}).Error; err != nil {
		return err
	}
	if mapping.ApplicationID != 0 {
		return d.DeleteApplicationByID(mapping.ApplicationID)
	}
	return nil
}


func (d *GormDatabase) CleanupAutomationHistory(before time.Time) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("completed = ? AND done_at IS NOT NULL AND done_at < ?", true, before).
			Delete(&model.EscalationState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("finished_at IS NOT NULL AND finished_at < ?", before).
			Delete(&model.ScheduledNotificationRun{}).Error; err != nil {
			return err
		}
		return tx.Where("created_at < ?", before).Delete(&model.ConnectorSeenItem{}).Error
	})
}
