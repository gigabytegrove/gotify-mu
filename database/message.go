package database

import (
	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func visibleMessages(db *gorm.DB, userID uint) *gorm.DB {
	return db.Joins("JOIN application_memberships AS am ON am.application_id = messages.application_id AND am.user_id = ?", userID).
		Joins("LEFT JOIN message_dismissals AS md ON md.message_id = messages.id AND md.user_id = ?", userID).
		Where("md.message_id IS NULL")
}

// GetMessageByID returns the messages for the given id or nil.
func (d *GormDatabase) GetMessageByID(id uint) (*model.Message, error) {
	msg := new(model.Message)
	err := d.DB.Find(msg, id).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if msg.ID == id {
		return msg, err
	}
	return nil, err
}

// CreateMessage creates a message.
func (d *GormDatabase) CreateMessage(message *model.Message) error {
	return d.DB.Create(message).Error
}

// GetMessagesByUser returns all messages from a user.
func (d *GormDatabase) GetMessagesByUser(userID uint) ([]*model.Message, error) {
	var messages []*model.Message
	err := visibleMessages(d.DB, userID).Order("messages.id desc").Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByUserSince returns limited messages from a user.
// If since is 0 it will be ignored.
func (d *GormDatabase) GetMessagesByUserSince(userID uint, limit int, since uint) ([]*model.Message, error) {
	var messages []*model.Message
	db := visibleMessages(d.DB, userID).Order("messages.id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByApplication returns all messages from an application.
func (d *GormDatabase) GetMessagesByApplication(tokenID uint) ([]*model.Message, error) {
	var messages []*model.Message
	err := d.DB.Where("application_id = ?", tokenID).Order("messages.id desc").Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByApplicationSince returns limited messages from an application.
// If since is 0 it will be ignored.
func (d *GormDatabase) GetMessagesByApplicationSince(appID uint, limit int, since uint) ([]*model.Message, error) {
	var messages []*model.Message
	db := d.DB.Where("application_id = ?", appID).Order("messages.id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByApplicationForUserSince returns one member's non-dismissed
// messages from an application.
func (d *GormDatabase) GetMessagesByApplicationForUserSince(userID, appID uint, limit int, since uint) ([]*model.Message, error) {
	var messages []*model.Message
	db := visibleMessages(d.DB, userID).Where("messages.application_id = ?", appID).
		Order("messages.id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// DismissMessageForUser hides a message from one user without affecting other members.
func (d *GormDatabase) DismissMessageForUser(userID, messageID uint) error {
	dismissal := model.MessageDismissal{UserID: userID, MessageID: messageID}
	return d.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&dismissal).Error
}

func createDismissals(tx *gorm.DB, userID uint, messageIDs []uint) error {
	if len(messageIDs) == 0 {
		return nil
	}
	dismissals := make([]model.MessageDismissal, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		dismissals = append(dismissals, model.MessageDismissal{UserID: userID, MessageID: messageID})
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&dismissals).Error
}

// DismissMessagesByUser hides all currently visible messages for one user.
func (d *GormDatabase) DismissMessagesByUser(userID uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var ids []uint
		if err := visibleMessages(tx.Model(&model.Message{}), userID).Pluck("messages.id", &ids).Error; err != nil {
			return err
		}
		return createDismissals(tx, userID, ids)
	})
}

// DismissMessagesByApplicationForUser hides a channel's messages for one member.
func (d *GormDatabase) DismissMessagesByApplicationForUser(userID, applicationID uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var ids []uint
		if err := tx.Model(&model.Message{}).Where("application_id = ?", applicationID).Pluck("id", &ids).Error; err != nil {
			return err
		}
		return createDismissals(tx, userID, ids)
	})
}

// DeleteMessageByID deletes a message by its id.
func (d *GormDatabase) DeleteMessageByID(id uint) error {
	if err := d.DB.Where("message_id = ?", id).Delete(&model.MessageDismissal{}).Error; err != nil {
		return err
	}
	return d.DB.Where("id = ?", id).Delete(&model.Message{}).Error
}

// DeleteMessagesByApplication deletes all messages from an application.
func (d *GormDatabase) DeleteMessagesByApplication(applicationID uint) error {
	subQuery := d.DB.Model(&model.Message{}).Select("id").Where("application_id = ?", applicationID)
	if err := d.DB.Where("message_id IN (?)", subQuery).Delete(&model.MessageDismissal{}).Error; err != nil {
		return err
	}
	return d.DB.Where("application_id = ?", applicationID).Delete(&model.Message{}).Error
}

// DeleteMessagesByUser deletes all messages from a user.
func (d *GormDatabase) DeleteMessagesByUser(userID uint) error {
	app, _ := d.GetApplicationsByUser(userID)
	for _, app := range app {
		d.DeleteMessagesByApplication(app.ID)
	}
	return nil
}
