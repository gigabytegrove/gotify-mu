package database

import (
	"errors"
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/secretstore"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetWebhookRoutes() ([]*model.WebhookRoute, error) {
	var items []*model.WebhookRoute
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := d.decryptWebhookRoute(item); err != nil { return nil, err }
	}
	return items, nil
}

func (d *GormDatabase) GetWebhookRouteByID(id uint) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptWebhookRoute(item); err != nil { return nil, err }
	return item, nil
}

func (d *GormDatabase) GetWebhookRouteBySecret(secret string) (*model.WebhookRoute, error) {
	item := new(model.WebhookRoute)
	if err := d.DB.Where("secret_hash = ? AND enabled = ?", secretstore.Hash(secret), true).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	if err := d.decryptWebhookRoute(item); err != nil { return nil, err }
	if item.Secret != secret { return nil, nil }
	return item, nil
}

func (d *GormDatabase) SaveWebhookRoute(item *model.WebhookRoute) error {
	plain := item.Secret
	plainSigning := item.SigningSecret
	encrypted, err := d.secrets.Encrypt(plain)
	if err != nil { return err }
	encryptedSigning, err := d.secrets.Encrypt(plainSigning)
	if err != nil { return err }
	item.Secret = encrypted
	item.SigningSecret = encryptedSigning
	item.SecretHash = secretstore.Hash(plain)
	err = d.DB.Save(item).Error
	item.Secret = plain
	item.SigningSecret = plainSigning
	return err
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
	plain := item.Password
	encrypted, err := d.secrets.Encrypt(plain)
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
	plain := item.Token
	encrypted, err := d.secrets.Encrypt(plain)
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


func (d *GormDatabase) decryptWebhookRoute(item *model.WebhookRoute) error {
	if item == nil { return nil }
	plain, err := d.secrets.Decrypt(item.Secret)
	if err != nil { return err }
	signing, err := d.secrets.Decrypt(item.SigningSecret)
	if err != nil { return err }
	item.Secret = plain
	item.SigningSecret = signing
	return nil
}

func (d *GormDatabase) decryptMQTT(item *model.MQTTIntegration) error {
	if item == nil { return nil }
	plain, err := d.secrets.Decrypt(item.Password)
	if err != nil { return err }
	item.Password = plain
	return nil
}

func (d *GormDatabase) decryptHomeAssistant(item *model.HomeAssistantIntegration) error {
	if item == nil { return nil }
	plain, err := d.secrets.Decrypt(item.Token)
	if err != nil { return err }
	item.Token = plain
	return nil
}

func (d *GormDatabase) migrateIntegrationSecrets() error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var webhooks []*model.WebhookRoute
		if err := tx.Find(&webhooks).Error; err != nil { return err }
		for _, item := range webhooks {
			plain, err := d.secrets.Decrypt(item.Secret)
			if err != nil { return err }
			encrypted, err := d.secrets.Encrypt(plain)
			if err != nil { return err }
			hash := secretstore.Hash(plain)
			signingPlain, err := d.secrets.Decrypt(item.SigningSecret)
			if err != nil { return err }
			encryptedSigning, err := d.secrets.Encrypt(signingPlain)
			if err != nil { return err }
			if item.Secret != encrypted || item.SecretHash != hash || item.SigningSecret != encryptedSigning {
				if err := tx.Model(item).Updates(map[string]any{
					"secret": encrypted,
					"secret_hash": hash,
					"signing_secret": encryptedSigning,
				}).Error; err != nil { return err }
			}
		}

		var mqttItems []*model.MQTTIntegration
		if err := tx.Find(&mqttItems).Error; err != nil { return err }
		for _, item := range mqttItems {
			plain, err := d.secrets.Decrypt(item.Password)
			if err != nil { return err }
			encrypted, err := d.secrets.Encrypt(plain)
			if err != nil { return err }
			if item.Password != encrypted {
				if err := tx.Model(item).Update("password", encrypted).Error; err != nil { return err }
			}
		}

		var homeAssistantItems []*model.HomeAssistantIntegration
		if err := tx.Find(&homeAssistantItems).Error; err != nil { return err }
		for _, item := range homeAssistantItems {
			plain, err := d.secrets.Decrypt(item.Token)
			if err != nil { return err }
			encrypted, err := d.secrets.Encrypt(plain)
			if err != nil { return err }
			if item.Token != encrypted {
				if err := tx.Model(item).Update("token", encrypted).Error; err != nil { return err }
			}
		}
		return nil
	})
}


func (d *GormDatabase) TryAcquireAutomationLease(name, owner string, now time.Time, ttl time.Duration) (bool, error) {
	expires := now.Add(ttl)
	acquired := false
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AutomationLease{}).
			Where("name = ? AND (owner = ? OR expires_at <= ?)", name, owner, now).
			Updates(map[string]any{
				"owner":      owner,
				"expires_at": expires,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			acquired = true
			return nil
		}

		lease := &model.AutomationLease{
			Name:      name,
			Owner:     owner,
			ExpiresAt: expires,
			UpdatedAt: now,
		}
		if err := tx.Create(lease).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil
			}
			return err
		}
		acquired = true
		return nil
	})
	return acquired, err
}


func (d *GormDatabase) CommitScheduledRun(
	item *model.ScheduledNotification,
	scheduledFor time.Time,
	runAt time.Time,
	nextRunAt *time.Time,
	disable bool,
) (*model.Message, error) {
	var result *model.Message
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		var existing model.ScheduledRun
		err := tx.Where("schedule_id = ? AND scheduled_for = ?", item.ID, scheduledFor).First(&existing).Error
		if err == nil {
			message := new(model.Message)
			if loadErr := tx.First(message, existing.MessageID).Error; loadErr != nil {
				return loadErr
			}
			result = message
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		message := &model.Message{
			ApplicationID: item.ApplicationID,
			Title:         item.Title,
			Message:       item.Message,
			Priority:      item.Priority,
			Date:          runAt,
		}
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		run := &model.ScheduledRun{
			ScheduleID:   item.ID,
			ScheduledFor: scheduledFor,
			MessageID:    message.ID,
		}
		if err := tx.Create(run).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"last_run_at": runAt,
			"next_run_at": nextRunAt,
		}
		if disable {
			updates["enabled"] = false
		}
		if err := tx.Model(&model.ScheduledNotification{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		result = message
		return nil
	})
	return result, err
}

func (d *GormDatabase) CommitEscalation(
	state *model.EscalationState,
	applicationID uint,
	title string,
	body string,
	priority int,
	now time.Time,
) (*model.Message, error) {
	var result *model.Message
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		current := new(model.EscalationState)
		if err := tx.First(current, state.ID).Error; err != nil {
			return err
		}
		if current.Completed {
			return nil
		}
		message := &model.Message{
			ApplicationID: applicationID,
			Title:         title,
			Message:       body,
			Priority:      priority,
			Date:          now,
		}
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		if err := tx.Model(current).Updates(map[string]any{
			"completed": true,
			"done_at":   now,
		}).Error; err != nil {
			return err
		}
		result = message
		return nil
	})
	return result, err
}

func (d *GormDatabase) CompleteEscalation(stateID uint, now time.Time) error {
	return d.DB.Model(&model.EscalationState{}).Where("id = ?", stateID).Updates(map[string]any{
		"completed": true,
		"done_at":   now,
	}).Error
}

func (d *GormDatabase) DeleteAutomationHistoryBefore(before time.Time) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("created_at < ?", before).Delete(&model.ScheduledRun{}).Error; err != nil {
			return err
		}
		if err := tx.Where("completed = ? AND done_at IS NOT NULL AND done_at < ?", true, before).
			Delete(&model.EscalationState{}).Error; err != nil {
			return err
		}
		return nil
	})
}


func (d *GormDatabase) GetMessageAcknowledgements(messageID uint) ([]*model.MessageAcknowledgementView, error) {
	type row struct {
		UserID         uint
		Username       string
		DisplayName    string
		AcknowledgedAt time.Time
	}
	var rows []row
	err := d.DB.Table("message_acknowledgements AS ma").
		Select("ma.user_id, users.name AS username, users.display_name, ma.acknowledged_at").
		Joins("JOIN users ON users.id = ma.user_id").
		Where("ma.message_id = ?", messageID).
		Order("ma.acknowledged_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*model.MessageAcknowledgementView, 0, len(rows))
	for _, item := range rows {
		out = append(out, &model.MessageAcknowledgementView{
			UserID:         item.UserID,
			Username:       item.Username,
			DisplayName:    item.DisplayName,
			AcknowledgedAt: item.AcknowledgedAt,
		})
	}
	return out, nil
}


func (d *GormDatabase) ConsumeWebhookReplay(routeID uint, signature string, now time.Time, ttl time.Duration) (bool, error) {
	accepted := false
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at <= ?", now).Delete(&model.WebhookReplay{}).Error; err != nil {
			return err
		}
		entry := &model.WebhookReplay{
			RouteID:   routeID,
			Signature: signature,
			ExpiresAt: now.Add(ttl),
		}
		if err := tx.Create(entry).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil
			}
			return err
		}
		accepted = true
		return nil
	})
	return accepted, err
}
