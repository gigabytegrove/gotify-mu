package database

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func hashAutomationSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (d *GormDatabase) encryptValue(value string) (string, error) {
	if d.Secrets == nil {
		return value, nil
	}
	return d.Secrets.Encrypt(value)
}

func (d *GormDatabase) decryptValue(value string) (string, error) {
	if d.Secrets == nil {
		return value, nil
	}
	return d.Secrets.Decrypt(value)
}

func (d *GormDatabase) decryptWebhook(item *model.WebhookRoute) error {
	value, err := d.decryptValue(item.Secret)
	if err != nil { return err }
	item.Secret = value
	return nil
}

func (d *GormDatabase) decryptMQTT(item *model.MQTTIntegration) error {
	value, err := d.decryptValue(item.Password)
	if err != nil { return err }
	item.Password = value
	return nil
}

func (d *GormDatabase) decryptHomeAssistant(item *model.HomeAssistantIntegration) error {
	value, err := d.decryptValue(item.Token)
	if err != nil { return err }
	item.Token = value
	return nil
}

func (d *GormDatabase) encryptLegacyAutomationSecrets() error {
	var webhooks []*model.WebhookRoute
	if err := d.DB.Find(&webhooks).Error; err != nil { return err }
	for _, item := range webhooks {
		plain, err := d.decryptValue(item.Secret)
		if err != nil { return err }
		encrypted, err := d.encryptValue(plain)
		if err != nil { return err }
		hash := hashAutomationSecret(plain)
		if item.Secret != encrypted || item.SecretHash != hash {
			if err := d.DB.Model(item).Updates(map[string]any{"secret": encrypted, "secret_hash": hash}).Error; err != nil { return err }
		}
	}

	var mqtt []*model.MQTTIntegration
	if err := d.DB.Find(&mqtt).Error; err != nil { return err }
	for _, item := range mqtt {
		plain, err := d.decryptValue(item.Password)
		if err != nil { return err }
		encrypted, err := d.encryptValue(plain)
		if err != nil { return err }
		if item.Password != encrypted {
			if err := d.DB.Model(item).Update("password", encrypted).Error; err != nil { return err }
		}
	}

	var homeAssistant []*model.HomeAssistantIntegration
	if err := d.DB.Find(&homeAssistant).Error; err != nil { return err }
	for _, item := range homeAssistant {
		plain, err := d.decryptValue(item.Token)
		if err != nil { return err }
		encrypted, err := d.encryptValue(plain)
		if err != nil { return err }
		if item.Token != encrypted {
			if err := d.DB.Model(item).Update("token", encrypted).Error; err != nil { return err }
		}
	}
	return nil
}

func (d *GormDatabase) GetWebhookRoutes() ([]*model.WebhookRoute, error) {
	var items []*model.WebhookRoute
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return items, err }
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
	hash := hashAutomationSecret(secret)
	if err := d.DB.Where("secret_hash = ? AND enabled = ?", hash, true).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptWebhook(item); err != nil { return nil, err }
	if subtle.ConstantTimeCompare([]byte(item.Secret), []byte(secret)) != 1 { return nil, nil }
	return item, nil
}

func (d *GormDatabase) SaveWebhookRoute(item *model.WebhookRoute) error {
	copy := *item
	copy.SecretHash = hashAutomationSecret(copy.Secret)
	encrypted, err := d.encryptValue(copy.Secret)
	if err != nil { return err }
	copy.Secret = encrypted
	if err := d.DB.Save(&copy).Error; err != nil { return err }
	item.ID, item.CreatedAt, item.UpdatedAt = copy.ID, copy.CreatedAt, copy.UpdatedAt
	item.SecretHash = copy.SecretHash
	return nil
}
func (d *GormDatabase) DeleteWebhookRoute(id uint) error { return d.DB.Delete(&model.WebhookRoute{}, id).Error }

func (d *GormDatabase) GetMQTTIntegrations() ([]*model.MQTTIntegration, error) {
	var items []*model.MQTTIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return items, err }
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
	copy := *item
	encrypted, err := d.encryptValue(copy.Password)
	if err != nil { return err }
	copy.Password = encrypted
	if err := d.DB.Save(&copy).Error; err != nil { return err }
	item.ID, item.CreatedAt, item.UpdatedAt = copy.ID, copy.CreatedAt, copy.UpdatedAt
	return nil
}
func (d *GormDatabase) DeleteMQTTIntegration(id uint) error { return d.DB.Delete(&model.MQTTIntegration{}, id).Error }

func (d *GormDatabase) GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error) {
	var items []*model.HomeAssistantIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return items, err }
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
	copy := *item
	encrypted, err := d.encryptValue(copy.Token)
	if err != nil { return err }
	copy.Token = encrypted
	if err := d.DB.Save(&copy).Error; err != nil { return err }
	item.ID, item.CreatedAt, item.UpdatedAt = copy.ID, copy.CreatedAt, copy.UpdatedAt
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
