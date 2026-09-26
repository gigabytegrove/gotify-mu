package database

import (
	"crypto/sha256"
	"errors"
	"encoding/hex"
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetWebhookRoutes() ([]*model.WebhookRoute, error) {
	var items []*model.WebhookRoute
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := d.decryptWebhook(item); err != nil { return nil, err }
	}
	return items, nil
}

func (d *GormDatabase) GetWebhookRouteByID(id uint) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptWebhook(item); err != nil { return nil, err }
	return item, nil
}

func (d *GormDatabase) GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	query := d.DB.Where("secret_hash = ? AND enabled = ?", secretHash(secret), true)
	if err := query.First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptWebhook(item); err != nil { return nil, err }
	if item.Secret != secret { return nil, nil }
	return item, nil
}

func (d *GormDatabase) SaveWebhookRoute(item *model.WebhookRoute) error {
	stored := *item
	stored.SecretHash = secretHash(item.Secret)
	if d.Secrets != nil {
		encrypted, err := d.Secrets.Encrypt(item.Secret)
		if err != nil { return err }
		stored.Secret = encrypted
	}
	if err := d.DB.Save(&stored).Error; err != nil { return err }
	item.ID, item.SecretHash, item.CreatedAt, item.UpdatedAt = stored.ID, stored.SecretHash, stored.CreatedAt, stored.UpdatedAt
	return nil
}
func (d *GormDatabase) DeleteWebhookRoute(id uint) error { return d.DB.Delete(&model.WebhookRoute{}, id).Error }

func (d *GormDatabase) GetMQTTIntegrations() ([]*model.MQTTIntegration, error) {
	var items []*model.MQTTIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		if err := d.decryptMQTT(item); err != nil { return nil, err }
	}
	return items, nil
}
func (d *GormDatabase) GetMQTTIntegrationByID(id uint) (*model.MQTTIntegration, error) {
	item := new(model.MQTTIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptMQTT(item); err != nil { return nil, err }
	return item, nil
}
func (d *GormDatabase) SaveMQTTIntegration(item *model.MQTTIntegration) error {
	stored := *item
	if d.Secrets != nil && item.Password != "" {
		encrypted, err := d.Secrets.Encrypt(item.Password)
		if err != nil { return err }
		stored.Password = encrypted
	}
	if err := d.DB.Save(&stored).Error; err != nil { return err }
	item.ID, item.CreatedAt, item.UpdatedAt = stored.ID, stored.CreatedAt, stored.UpdatedAt
	return nil
}
func (d *GormDatabase) DeleteMQTTIntegration(id uint) error { return d.DB.Delete(&model.MQTTIntegration{}, id).Error }

func (d *GormDatabase) GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error) {
	var items []*model.HomeAssistantIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		if err := d.decryptHomeAssistant(item); err != nil { return nil, err }
	}
	return items, nil
}
func (d *GormDatabase) GetHomeAssistantIntegrationByID(id uint) (*model.HomeAssistantIntegration, error) {
	item := new(model.HomeAssistantIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptHomeAssistant(item); err != nil { return nil, err }
	return item, nil
}
func (d *GormDatabase) SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error {
	stored := *item
	if d.Secrets != nil && item.Token != "" {
		encrypted, err := d.Secrets.Encrypt(item.Token)
		if err != nil { return err }
		stored.Token = encrypted
	}
	if err := d.DB.Save(&stored).Error; err != nil { return err }
	item.ID, item.CreatedAt, item.UpdatedAt = stored.ID, stored.CreatedAt, stored.UpdatedAt
	return nil
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
		DoUpdates: clause.AssignmentColumns([]string{"enabled","start_minute","end_minute","timezone","allow_priority","updated_at"}),
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


func secretHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (d *GormDatabase) decryptWebhook(item *model.WebhookRoute) error {
	if d.Secrets == nil || item == nil { return nil }
	plain, err := d.Secrets.Decrypt(item.Secret)
	if err != nil { return err }
	item.Secret = plain
	return nil
}

func (d *GormDatabase) decryptMQTT(item *model.MQTTIntegration) error {
	if d.Secrets == nil || item == nil || item.Password == "" { return nil }
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return err }
	item.Password = plain
	return nil
}

func (d *GormDatabase) decryptHomeAssistant(item *model.HomeAssistantIntegration) error {
	if d.Secrets == nil || item == nil || item.Token == "" { return nil }
	plain, err := d.Secrets.Decrypt(item.Token)
	if err != nil { return err }
	item.Token = plain
	return nil
}

func (d *GormDatabase) encryptStoredIntegrationSecrets() error {
	if d.Secrets == nil { return nil }

	var webhooks []*model.WebhookRoute
	if err := d.DB.Find(&webhooks).Error; err != nil { return err }
	for _, item := range webhooks {
		plain, err := d.Secrets.Decrypt(item.Secret)
		if err != nil { return err }
		encrypted, err := d.Secrets.Encrypt(plain)
		if err != nil { return err }
		if err := d.DB.Model(item).Updates(map[string]any{
			"secret": encrypted,
			"secret_hash": secretHash(plain),
		}).Error; err != nil { return err }
	}

	var mqtt []*model.MQTTIntegration
	if err := d.DB.Find(&mqtt).Error; err != nil { return err }
	for _, item := range mqtt {
		if item.Password == "" { continue }
		plain, err := d.Secrets.Decrypt(item.Password)
		if err != nil { return err }
		encrypted, err := d.Secrets.Encrypt(plain)
		if err != nil { return err }
		if err := d.DB.Model(item).Update("password", encrypted).Error; err != nil { return err }
	}

	var homeAssistant []*model.HomeAssistantIntegration
	if err := d.DB.Find(&homeAssistant).Error; err != nil { return err }
	for _, item := range homeAssistant {
		if item.Token == "" { continue }
		plain, err := d.Secrets.Decrypt(item.Token)
		if err != nil { return err }
		encrypted, err := d.Secrets.Encrypt(plain)
		if err != nil { return err }
		if err := d.DB.Model(item).Update("token", encrypted).Error; err != nil { return err }
	}
	return nil
}


func (d *GormDatabase) TryAcquireAutomationLease(name, owner string, now time.Time, ttl time.Duration) (bool, error) {
	until := now.Add(ttl)
	result := d.DB.Model(&model.AutomationLease{}).
		Where("name = ? AND (lease_until < ? OR owner = ?)", name, now, owner).
		Updates(map[string]any{"owner": owner, "lease_until": until, "updated_at": now})
	if result.Error != nil { return false, result.Error }
	if result.RowsAffected > 0 { return true, nil }

	item := &model.AutomationLease{Name:name, Owner:owner, LeaseUntil:until, UpdatedAt:now}
	if err := d.DB.Create(item).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *GormDatabase) ReleaseAutomationLease(name, owner string, now time.Time) error {
	return d.DB.Model(&model.AutomationLease{}).
		Where("name = ? AND owner = ?", name, owner).
		Updates(map[string]any{"lease_until": now, "updated_at": now}).Error
}


func (d *GormDatabase) CreateScheduledMessage(item *model.ScheduledNotification, message *model.Message, now time.Time, nextRunAt *time.Time, enabled bool) (bool, error) {
	created := false
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.ScheduledNotification{}).
			Where("id = ? AND enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", item.ID, true, now).
			Updates(map[string]any{
				"last_run_at": now,
				"next_run_at": nextRunAt,
				"enabled": enabled,
				"updated_at": now,
			})
		if result.Error != nil { return result.Error }
		if result.RowsAffected == 0 { return nil }
		if err := tx.Create(message).Error; err != nil { return err }
		created = true
		return nil
	})
	return created, err
}

func (d *GormDatabase) CompleteEscalationWithMessage(state *model.EscalationState, message *model.Message, now time.Time) (bool, error) {
	created := false
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		var acknowledgements int64
		if err := tx.Model(&model.MessageAcknowledgement{}).
			Where("message_id = ?", state.MessageID).Count(&acknowledgements).Error; err != nil {
			return err
		}
		if acknowledgements > 0 {
			return tx.Model(&model.EscalationState{}).
				Where("id = ? AND completed = ?", state.ID, false).
				Updates(map[string]any{"completed":true, "done_at":now}).Error
		}

		result := tx.Model(&model.EscalationState{}).
			Where("id = ? AND completed = ?", state.ID, false).
			Updates(map[string]any{"completed":true, "done_at":now})
		if result.Error != nil { return result.Error }
		if result.RowsAffected == 0 { return nil }
		if err := tx.Create(message).Error; err != nil { return err }
		created = true
		return nil
	})
	return created, err
}
