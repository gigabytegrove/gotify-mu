package database

import (
	"crypto/subtle"
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
		plain, err := d.decryptCredential(item.Secret)
		if err != nil {
			return nil, err
		}
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
	plain, err := d.decryptCredential(item.Secret)
	if err != nil {
		return nil, err
	}
	item.Secret = plain
	return item, nil
}

func (d *GormDatabase) GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	if err := d.DB.Where("secret_hash = ? AND enabled = ?", hashWebhookSecret(secret), true).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	plain, err := d.decryptCredential(item.Secret)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(plain), []byte(secret)) != 1 {
		return nil, nil
	}
	item.Secret = plain
	return item, nil
}

func (d *GormDatabase) SaveWebhookRoute(item *model.WebhookRoute) error {
	copyItem := *item
	copyItem.SecretHash = hashWebhookSecret(item.Secret)
	encrypted, err := d.encryptCredential(item.Secret)
	if err != nil {
		return err
	}
	copyItem.Secret = encrypted
	if err := d.DB.Save(&copyItem).Error; err != nil {
		return err
	}
	item.ID, item.CreatedAt, item.UpdatedAt, item.SecretHash = copyItem.ID, copyItem.CreatedAt, copyItem.UpdatedAt, copyItem.SecretHash
	return nil
}
func (d *GormDatabase) DeleteWebhookRoute(id uint) error { return d.DB.Delete(&model.WebhookRoute{}, id).Error }

func (d *GormDatabase) GetMQTTIntegrations() ([]*model.MQTTIntegration, error) {
	var items []*model.MQTTIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		plain, err := d.decryptCredential(item.Password)
		if err != nil {
			return nil, err
		}
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
	plain, err := d.decryptCredential(item.Password)
	if err != nil {
		return nil, err
	}
	item.Password = plain
	return item, nil
}
func (d *GormDatabase) SaveMQTTIntegration(item *model.MQTTIntegration) error {
	copyItem := *item
	encrypted, err := d.encryptCredential(item.Password)
	if err != nil {
		return err
	}
	copyItem.Password = encrypted
	if err := d.DB.Save(&copyItem).Error; err != nil {
		return err
	}
	item.ID, item.CreatedAt, item.UpdatedAt = copyItem.ID, copyItem.CreatedAt, copyItem.UpdatedAt
	return nil
}
func (d *GormDatabase) DeleteMQTTIntegration(id uint) error { return d.DB.Delete(&model.MQTTIntegration{}, id).Error }

func (d *GormDatabase) GetHomeAssistantIntegrations() ([]*model.HomeAssistantIntegration, error) {
	var items []*model.HomeAssistantIntegration
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		plain, err := d.decryptCredential(item.Token)
		if err != nil {
			return nil, err
		}
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
	plain, err := d.decryptCredential(item.Token)
	if err != nil {
		return nil, err
	}
	item.Token = plain
	return item, nil
}
func (d *GormDatabase) SaveHomeAssistantIntegration(item *model.HomeAssistantIntegration) error {
	copyItem := *item
	encrypted, err := d.encryptCredential(item.Token)
	if err != nil {
		return err
	}
	copyItem.Token = encrypted
	if err := d.DB.Save(&copyItem).Error; err != nil {
		return err
	}
	item.ID, item.CreatedAt, item.UpdatedAt = copyItem.ID, copyItem.CreatedAt, copyItem.UpdatedAt
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
	return items, d.DB.Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (claim_until IS NULL OR claim_until <= ?)", true, now, now).Find(&items).Error
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
		DoUpdates: clause.AssignmentColumns([]string{"enabled","interval_minutes","immediate_priority","last_sent_at","next_run_at","claim_owner","claim_until","updated_at"}),
	}).Create(item).Error
}
func (d *GormDatabase) QueueDigestItem(item *model.DigestItem) error {
	return d.DB.Clauses(clause.OnConflict{DoNothing:true}).Create(item).Error
}
func (d *GormDatabase) GetDigestItems(userID uint) ([]*model.DigestItem, error) {
	var items []*model.DigestItem
	return items, d.DB.Where("user_id = ?", userID).Order("created_at asc").Find(&items).Error
}
func (d *GormDatabase) DeleteDigestItems(userID uint) error {
	return d.DB.Where("user_id = ?", userID).Delete(&model.DigestItem{}).Error
}
func (d *GormDatabase) DeleteDigestItemsByIDs(userID uint, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return d.DB.Where("user_id = ? AND id IN ?", userID, ids).Delete(&model.DigestItem{}).Error
}
func (d *GormDatabase) GetDueDigestPolicies(now time.Time) ([]*model.DigestPolicy, error) {
	var items []*model.DigestPolicy
	return items, d.DB.Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (claim_until IS NULL OR claim_until <= ?)", true, now, now).Find(&items).Error
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
	return items, d.DB.Where("completed = ? AND due_at <= ? AND (claim_until IS NULL OR claim_until <= ?)", false, now, now).Find(&items).Error
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


func (d *GormDatabase) ClaimScheduledNotification(id uint, owner string, now, until time.Time) (bool, error) {
	result := d.DB.Model(&model.ScheduledNotification{}).
		Where("id = ? AND (claim_until IS NULL OR claim_until <= ? OR claim_owner = ?)", id, now, owner).
		Updates(map[string]any{"claim_owner": owner, "claim_until": until})
	return result.RowsAffected == 1, result.Error
}

func (d *GormDatabase) ClaimDigestPolicy(id uint, owner string, now, until time.Time) (bool, error) {
	result := d.DB.Model(&model.DigestPolicy{}).
		Where("id = ? AND (claim_until IS NULL OR claim_until <= ? OR claim_owner = ?)", id, now, owner).
		Updates(map[string]any{"claim_owner": owner, "claim_until": until})
	return result.RowsAffected == 1, result.Error
}

func (d *GormDatabase) ClaimEscalation(id uint, owner string, now, until time.Time) (bool, error) {
	result := d.DB.Model(&model.EscalationState{}).
		Where("id = ? AND completed = ? AND (claim_until IS NULL OR claim_until <= ? OR claim_owner = ?)", id, false, now, owner).
		Updates(map[string]any{"claim_owner": owner, "claim_until": until})
	return result.RowsAffected == 1, result.Error
}

func (d *GormDatabase) AcquireAutomationLease(name, owner string, now, until time.Time) (bool, error) {
	result := d.DB.Model(&model.AutomationLease{}).
		Where("name = ? AND (expires_at <= ? OR owner = ?)", name, now, owner).
		Updates(map[string]any{"owner": owner, "expires_at": until, "updated_at": now})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	lease := &model.AutomationLease{Name: name, Owner: owner, ExpiresAt: until, UpdatedAt: now}
	result = d.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(lease)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (d *GormDatabase) ReleaseAutomationLease(name, owner string) error {
	return d.DB.Where("name = ? AND owner = ?", name, owner).Delete(&model.AutomationLease{}).Error
}
