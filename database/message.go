package database

import (
	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func (d *GormDatabase) markAcknowledged(userID uint, messages []*model.Message) error {
	if len(messages) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}
	var acknowledged []uint
	if err := d.DB.Model(&model.MessageAcknowledgement{}).
		Where("user_id = ? AND message_id IN ?", userID, ids).
		Pluck("message_id", &acknowledged).Error; err != nil {
		return err
	}
	set := make(map[uint]struct{}, len(acknowledged))
	for _, id := range acknowledged {
		set[id] = struct{}{}
	}
	for _, message := range messages {
		_, message.Acknowledged = set[message.ID]
	}
	return nil
}

func visibleMessages(db *gorm.DB, userID uint) *gorm.DB {
	return db.Joins("JOIN application_memberships AS am ON am.application_id = messages.application_id AND am.user_id = ?", userID).
		Joins("LEFT JOIN message_dismissals AS md ON md.message_id = messages.id AND md.user_id = ?", userID).
		Where("md.message_id IS NULL")
}

func archivedMessages(db *gorm.DB, userID uint) *gorm.DB {
	return db.Joins("JOIN application_memberships AS am ON am.application_id = messages.application_id AND am.user_id = ?", userID).
		Joins("JOIN message_dismissals AS md ON md.message_id = messages.id AND md.user_id = ?", userID).
		Where("md.archived = ?", true)
}

func (d *GormDatabase) GetMessageByAutomationKey(key string) (*model.Message, error) {
	message := new(model.Message)
	if err := d.DB.Where("automation_key = ?", key).First(message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return message, nil
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
	if err == nil {
		err = d.markAcknowledged(userID, messages)
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
	if err == nil {
		err = d.markAcknowledged(userID, messages)
	}
	return messages, err
}

// GetArchivedMessagesByUserSince returns limited archived messages for one user.
func (d *GormDatabase) GetArchivedMessagesByUserSince(
	userID uint,
	limit int,
	since uint,
) ([]*model.Message, error) {
	var messages []*model.Message
	db := archivedMessages(d.DB, userID).Order("messages.id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if err == nil {
		err = d.markAcknowledged(userID, messages)
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
	if err == nil {
		err = d.markAcknowledged(userID, messages)
	}
	return messages, err
}

// GetArchivedMessagesByApplicationForUserSince returns one member's archived
// messages from an application.
func (d *GormDatabase) GetArchivedMessagesByApplicationForUserSince(
	userID, appID uint,
	limit int,
	since uint,
) ([]*model.Message, error) {
	var messages []*model.Message
	db := archivedMessages(d.DB, userID).
		Where("messages.application_id = ?", appID).
		Order("messages.id desc").
		Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := db.Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if err == nil {
		err = d.markAcknowledged(userID, messages)
	}
	return messages, err
}

// DismissMessageForUser hides a message from one user without affecting other members.
func (d *GormDatabase) DismissMessageForUser(userID, messageID uint) error {
	dismissal := model.MessageDismissal{UserID: userID, MessageID: messageID, Archived: false}
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"archived"}),
	}).Create(&dismissal).Error
}

// ArchiveMessageForUser hides a message from one user's active view while
// retaining it as a reversible archive.
func (d *GormDatabase) ArchiveMessageForUser(userID, messageID uint) error {
	archive := model.MessageDismissal{UserID: userID, MessageID: messageID, Archived: true}
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"archived"}),
	}).Create(&archive).Error
}

// UnarchiveMessageForUser restores one archived message to the active view.
func (d *GormDatabase) UnarchiveMessageForUser(userID, messageID uint) error {
	return d.DB.Where(
		"user_id = ? AND message_id = ? AND archived = ?",
		userID,
		messageID,
		true,
	).Delete(&model.MessageDismissal{}).Error
}

func createDismissals(tx *gorm.DB, userID uint, messageIDs []uint) error {
	if len(messageIDs) == 0 {
		return nil
	}
	dismissals := make([]model.MessageDismissal, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		dismissals = append(dismissals, model.MessageDismissal{
			UserID: userID, MessageID: messageID, Archived: false,
		})
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&dismissals).Error
}

func createArchives(tx *gorm.DB, userID uint, messageIDs []uint) error {
	if len(messageIDs) == 0 {
		return nil
	}
	archives := make([]model.MessageDismissal, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		archives = append(archives, model.MessageDismissal{
			UserID:    userID,
			MessageID: messageID,
			Archived:  true,
		})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"archived"}),
	}).Create(&archives).Error
}

// ArchiveMessagesByUser archives all currently visible messages for one user.
func (d *GormDatabase) ArchiveMessagesByUser(userID uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var ids []uint
		if err := visibleMessages(tx.Model(&model.Message{}), userID).
			Pluck("messages.id", &ids).Error; err != nil {
			return err
		}
		return createArchives(tx, userID, ids)
	})
}

// ArchiveMessagesByApplicationForUser archives a channel's visible messages for one member.
func (d *GormDatabase) ArchiveMessagesByApplicationForUser(
	userID, applicationID uint,
) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var ids []uint
		if err := visibleMessages(tx.Model(&model.Message{}), userID).
			Where("messages.application_id = ?", applicationID).
			Pluck("messages.id", &ids).Error; err != nil {
			return err
		}
		return createArchives(tx, userID, ids)
	})
}

// UnarchiveMessagesByUser restores all archived messages for one user.
func (d *GormDatabase) UnarchiveMessagesByUser(userID uint) error {
	return d.DB.Where("user_id = ? AND archived = ?", userID, true).
		Delete(&model.MessageDismissal{}).Error
}

// UnarchiveMessagesByApplicationForUser restores one channel's archived messages for one user.
func (d *GormDatabase) UnarchiveMessagesByApplicationForUser(
	userID, applicationID uint,
) error {
	subQuery := d.DB.Model(&model.Message{}).
		Select("id").
		Where("application_id = ?", applicationID)
	return d.DB.Where(
		"user_id = ? AND archived = ? AND message_id IN (?)",
		userID,
		true,
		subQuery,
	).Delete(&model.MessageDismissal{}).Error
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
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("message_id = ?", id).Delete(&model.MessageDismissal{}).Error; err != nil { return err }
		if err := tx.Where("message_id = ?", id).Delete(&model.MessageAcknowledgement{}).Error; err != nil { return err }
		if err := tx.Where("message_id = ?", id).Delete(&model.DigestItem{}).Error; err != nil { return err }
		if err := tx.Where("message_id = ?", id).Delete(&model.EscalationState{}).Error; err != nil { return err }
		return tx.Where("id = ?", id).Delete(&model.Message{}).Error
	})
}

// DeleteMessagesByApplication deletes all messages from an application.
func (d *GormDatabase) DeleteMessagesByApplication(applicationID uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		subQuery := tx.Model(&model.Message{}).Select("id").Where("application_id = ?", applicationID)
		if err := tx.Where("message_id IN (?)", subQuery).Delete(&model.MessageDismissal{}).Error; err != nil { return err }
		if err := tx.Where("message_id IN (?)", subQuery).Delete(&model.MessageAcknowledgement{}).Error; err != nil { return err }
		if err := tx.Where("message_id IN (?)", subQuery).Delete(&model.DigestItem{}).Error; err != nil { return err }
		if err := tx.Where("message_id IN (?)", subQuery).Delete(&model.EscalationState{}).Error; err != nil { return err }
		return tx.Where("application_id = ?", applicationID).Delete(&model.Message{}).Error
	})
}

// DeleteMessagesByUser deletes all messages from a user.
func (d *GormDatabase) DeleteMessagesByUser(userID uint) error {
	app, _ := d.GetApplicationsByUser(userID)
	for _, app := range app {
		d.DeleteMessagesByApplication(app.ID)
	}
	return nil
}
