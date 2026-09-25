package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetWebhookRoutes() ([]*model.WebhookRoute, error) {
	var items []*model.WebhookRoute
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}

func (d *GormDatabase) GetWebhookRouteByID(id uint) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	verifier := security.WebhookVerifier(secret)
	if err := d.DB.Where("(secret = ? OR secret = ?) AND enabled = ?", secret, verifier, true).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if item.Secret == secret {
		item.Secret = verifier
		if err := d.DB.Model(item).Update("secret", verifier).Error; err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (d *GormDatabase) SaveWebhookRoute(item *model.WebhookRoute) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteWebhookRoute(id uint) error { return d.DB.Delete(&model.WebhookRoute{}, id).Error }

func (d *GormDatabase) GetMQTTIntegrations() ([]*model.MQTTIntegration, error) {
	var items []*model.MQTTIntegration
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error) {
	item := new(model.MQTTIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveMQTTIntegration(item *model.MQTTIntegration) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteMQTTIntegration(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kind = ? AND object_id = ?", "mqtt", id).Delete(&model.IntegrationStatus{}).Error; err != nil { return err }
		if err := tx.Where("key = ?", fmt.Sprintf("integration:mqtt:%d", id)).Delete(&model.AutomationLease{}).Error; err != nil { return err }
		return tx.Delete(&model.MQTTIntegration{}, id).Error
	})
}

func (d *GormDatabase) GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error) {
	var items []*model.HomeAssistantIntegration
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error) {
	item := new(model.HomeAssistantIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteHomeAssistantIntegration(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kind = ? AND object_id = ?", "home-assistant", id).Delete(&model.IntegrationStatus{}).Error; err != nil { return err }
		if err := tx.Where("key = ?", fmt.Sprintf("integration:home-assistant:%d", id)).Delete(&model.AutomationLease{}).Error; err != nil { return err }
		return tx.Delete(&model.HomeAssistantIntegration{}, id).Error
	})
}

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
func (d *GormDatabase) DeleteScheduledNotification(id uint) error { return d.DB.Delete(&model.ScheduledNotification{}, id).Error }
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
		DoUpdates: clause.AssignmentColumns([]string{"enabled","start_minute","end_minute","timezone","allow_priority","behavior","updated_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) QueueDeferredNotification(item *model.DeferredNotification) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"user_id"},{Name:"message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"application_id","due_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) GetDueDeferredNotifications(now time.Time) ([]*model.DeferredNotification, error) {
	var items []*model.DeferredNotification
	return items, d.DB.Where("due_at <= ?", now).Order("due_at asc").Find(&items).Error
}

func (d *GormDatabase) DeleteDeferredNotification(userID, messageID uint) error {
	return d.DB.Where("user_id = ? AND message_id = ?", userID, messageID).Delete(&model.DeferredNotification{}).Error
}

func (d *GormDatabase) DeleteDeferredNotificationsForUser(userID uint) error {
	return d.DB.Where("user_id = ?", userID).Delete(&model.DeferredNotification{}).Error
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


func (d *GormDatabase) AcquireAutomationLease(key, owner string, now time.Time, ttl time.Duration) (bool, error) {
	expires := now.Add(ttl)
	result := d.DB.Model(&model.AutomationLease{}).
		Where("key = ? AND (expires_at < ? OR owner = ?)", key, now, owner).
		Updates(map[string]any{"owner": owner, "expires_at": expires, "updated_at": now})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		return true, nil
	}
	item := &model.AutomationLease{Key:key, Owner:owner, ExpiresAt:expires, UpdatedAt:now}
	if err := d.DB.Create(item).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *GormDatabase) ReleaseAutomationLease(key, owner string) error {
	return d.DB.Where("key = ? AND owner = ?", key, owner).Delete(&model.AutomationLease{}).Error
}

func (d *GormDatabase) CreateAutomationRun(item *model.AutomationRun) (bool, error) {
	if err := d.DB.Create(item).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *GormDatabase) SaveAutomationRun(item *model.AutomationRun) error {
	return d.DB.Save(item).Error
}

func (d *GormDatabase) GetAutomationRunByTrigger(triggerKey string) (*model.AutomationRun, error) {
	item := new(model.AutomationRun)
	if err := d.DB.Where("trigger_key = ?", triggerKey).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}


func (d *GormDatabase) GetAutomationRuns(kind string, objectID uint, limit int) ([]*model.AutomationRun, error) {
	if limit <= 0 || limit > 500 { limit = 100 }
	query := d.DB.Model(&model.AutomationRun{})
	if kind != "" { query = query.Where("kind = ?", kind) }
	if objectID != 0 { query = query.Where("object_id = ?", objectID) }
	var items []*model.AutomationRun
	err := query.Order("started_at desc, id desc").Limit(limit).Find(&items).Error
	return items, err
}

func (d *GormDatabase) DeleteAutomationRunsBefore(before time.Time) error {
	return d.DB.Where("started_at < ?", before).Delete(&model.AutomationRun{}).Error
}

func (d *GormDatabase) SaveIntegrationStatus(item *model.IntegrationStatus) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"kind"},{Name:"object_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"state","last_connected_at","last_activity_at","last_error","updated_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) GetIntegrationStatuses() ([]*model.IntegrationStatus, error) {
	var items []*model.IntegrationStatus
	return items, d.DB.Order("kind asc, object_id asc").Find(&items).Error
}

func (d *GormDatabase) GetIntegrationStatus(kind string, objectID uint) (*model.IntegrationStatus, error) {
	item := new(model.IntegrationStatus)
	if err := d.DB.Where("kind = ? AND object_id = ?", kind, objectID).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}


func (d *GormDatabase) DeleteIntegrationStatus(kind string, objectID uint) error {
	return d.DB.Where("kind = ? AND object_id = ?", kind, objectID).Delete(&model.IntegrationStatus{}).Error
}

func (d *GormDatabase) GetMessageAcknowledgements(messageID uint) ([]model.MessageAcknowledgementExternal, error) {
	type row struct {
		UserID uint
		Name string
		DisplayName string
		AcknowledgedAt time.Time
	}
	var rows []row
	if err := d.DB.Table("message_acknowledgements AS ma").
		Select("ma.user_id, users.name, users.display_name, ma.acknowledged_at").
		Joins("JOIN users ON users.id = ma.user_id").
		Where("ma.message_id = ?", messageID).
		Order("ma.acknowledged_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]model.MessageAcknowledgementExternal, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.MessageAcknowledgementExternal{
			UserID:row.UserID, Name:row.Name, DisplayName:row.DisplayName, AcknowledgedAt:row.AcknowledgedAt,
		})
	}
	return items, nil
}
