package database

import (
	"errors"
	"strings"
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) EnrichMessageCollaboration(userID uint, messages []*model.Message) error {
	if len(messages) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(messages))
	index := make(map[uint]*model.Message, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
		index[message.ID] = message
		message.Collaboration = model.MessageCollaboration{}
	}

	var attachments []*model.MessageAttachment
	if err := d.DB.Where("message_id IN ?", ids).Order("id asc").Find(&attachments).Error; err != nil {
		return err
	}
	for _, item := range attachments {
		if message := index[item.MessageID]; message != nil {
			message.Collaboration.Attachments = append(message.Collaboration.Attachments, model.MessageAttachmentView{
				ID: item.ID,
				Filename: item.Filename,
				ContentType: item.ContentType,
				Size: item.Size,
				URL: "/message/" + uintString(item.MessageID) + "/attachment/" + uintString(item.ID),
			})
		}
	}

	type reactionRow struct {
		MessageID uint
		Emoji string
		Count int
		ReactedByMe bool
	}
	var reactions []reactionRow
	if err := d.DB.Table("message_reactions").
		Select("message_id, emoji, COUNT(*) AS count, MAX(CASE WHEN user_id = ? THEN 1 ELSE 0 END) AS reacted_by_me", userID).
		Where("message_id IN ?", ids).
		Group("message_id, emoji").
		Order("message_id asc, emoji asc").
		Scan(&reactions).Error; err != nil {
		return err
	}
	for _, item := range reactions {
		if message := index[item.MessageID]; message != nil {
			message.Collaboration.Reactions = append(message.Collaboration.Reactions, model.MessageReactionSummary{
				Emoji: item.Emoji,
				Count: item.Count,
				ReactedByMe: item.ReactedByMe,
			})
		}
	}

	type workflowRow struct {
		MessageID uint
		AssignedUserID uint
		AssignedUserName string
		Status string
		ResolvedBy uint
		ResolvedByName string
		ResolvedAt *time.Time
	}
	var workflows []workflowRow
	if err := d.DB.Table("message_workflows AS mw").
		Select("mw.message_id, mw.assigned_user_id, assigned.display_name AS assigned_user_name, mw.status, mw.resolved_by, resolver.display_name AS resolved_by_name, mw.resolved_at").
		Joins("LEFT JOIN users AS assigned ON assigned.id = mw.assigned_user_id").
		Joins("LEFT JOIN users AS resolver ON resolver.id = mw.resolved_by").
		Where("mw.message_id IN ?", ids).
		Scan(&workflows).Error; err != nil {
		return err
	}
	for _, item := range workflows {
		if message := index[item.MessageID]; message != nil {
			message.Collaboration.AssignedUserID = item.AssignedUserID
			message.Collaboration.AssignedUserName = item.AssignedUserName
			message.Collaboration.Status = item.Status
			message.Collaboration.ResolvedBy = item.ResolvedBy
			message.Collaboration.ResolvedByName = item.ResolvedByName
			message.Collaboration.ResolvedAt = item.ResolvedAt
		}
	}

	var readIDs []uint
	if err := d.DB.Model(&model.MessageRead{}).
		Where("user_id = ? AND message_id IN ?", userID, ids).
		Pluck("message_id", &readIDs).Error; err != nil {
		return err
	}
	for _, id := range readIDs {
		if message := index[id]; message != nil {
			message.Collaboration.Read = true
		}
	}

	var mentionIDs []uint
	if err := d.DB.Model(&model.MessageMention{}).
		Where("user_id = ? AND message_id IN ?", userID, ids).
		Pluck("message_id", &mentionIDs).Error; err != nil {
		return err
	}
	for _, id := range mentionIDs {
		if message := index[id]; message != nil {
			message.Collaboration.Mentioned = true
		}
	}

	type replyRow struct {
		ThreadRootMessageID uint
		Count int
	}
	var replies []replyRow
	if err := d.DB.Model(&model.Message{}).
		Select("thread_root_message_id, COUNT(*) AS count").
		Where("thread_root_message_id IN ? AND reply_to_message_id <> 0", ids).
		Group("thread_root_message_id").
		Scan(&replies).Error; err != nil {
		return err
	}
	for _, item := range replies {
		if message := index[item.ThreadRootMessageID]; message != nil {
			message.Collaboration.ReplyCount = item.Count
		}
	}
	return nil
}

func uintString(value uint) string {
	if value == 0 {
		return "0"
	}
	const digits = "0123456789"
	buf := [20]byte{}
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}

func (d *GormDatabase) AddMessageReaction(messageID, userID uint, emoji string) error {
	item := &model.MessageReaction{MessageID: messageID, UserID: userID, Emoji: strings.TrimSpace(emoji)}
	return d.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(item).Error
}

func (d *GormDatabase) DeleteMessageReaction(messageID, userID uint, emoji string) error {
	return d.DB.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, strings.TrimSpace(emoji)).
		Delete(&model.MessageReaction{}).Error
}

func (d *GormDatabase) SaveMessageWorkflow(item *model.MessageWorkflow) error {
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"assigned_user_id", "status", "resolved_by", "resolved_at", "updated_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) GetMessageWorkflow(messageID uint) (*model.MessageWorkflow, error) {
	item := new(model.MessageWorkflow)
	if err := d.DB.First(item, "message_id = ?", messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) MarkMessageRead(userID, messageID uint, at time.Time) error {
	item := &model.MessageRead{UserID: userID, MessageID: messageID, ReadAt: at}
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"read_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) MarkMessageUnread(userID, messageID uint) error {
	return d.DB.Where("user_id = ? AND message_id = ?", userID, messageID).Delete(&model.MessageRead{}).Error
}

func (d *GormDatabase) ReplaceMessageMentions(messageID uint, userIDs []uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("message_id = ?", messageID).Delete(&model.MessageMention{}).Error; err != nil {
			return err
		}
		for _, userID := range userIDs {
			if userID == 0 {
				continue
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.MessageMention{
				MessageID: messageID,
				UserID: userID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *GormDatabase) GetThreadMessagesForUser(userID, rootID uint) ([]*model.Message, error) {
	var items []*model.Message
	err := visibleMessages(d.DB, userID).
		Where("messages.id = ? OR messages.thread_root_message_id = ?", rootID, rootID).
		Order("messages.id asc").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	if err = d.markAcknowledged(userID, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (d *GormDatabase) CreateMessageAttachment(item *model.MessageAttachment) error {
	return d.DB.Create(item).Error
}

func (d *GormDatabase) GetMessageAttachment(id uint) (*model.MessageAttachment, error) {
	item := new(model.MessageAttachment)
	if err := d.DB.First(item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) DeleteMessageAttachment(id uint) error {
	return d.DB.Delete(&model.MessageAttachment{}, id).Error
}

func (d *GormDatabase) GetMessageTemplates(userID uint) ([]*model.MessageTemplate, error) {
	var items []*model.MessageTemplate
	return items, d.DB.Where("user_id = ?", userID).Order("name asc, id asc").Find(&items).Error
}

func (d *GormDatabase) GetMessageTemplateByID(userID, id uint) (*model.MessageTemplate, error) {
	item := new(model.MessageTemplate)
	if err := d.DB.First(item, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) SaveMessageTemplate(item *model.MessageTemplate) error {
	return d.DB.Save(item).Error
}

func (d *GormDatabase) DeleteMessageTemplate(userID, id uint) error {
	return d.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.MessageTemplate{}).Error
}

func (d *GormDatabase) GetSavedMessageSearches(userID uint) ([]*model.SavedMessageSearch, error) {
	var items []*model.SavedMessageSearch
	return items, d.DB.Where("user_id = ?", userID).Order("name asc, id asc").Find(&items).Error
}

func (d *GormDatabase) GetSavedMessageSearchByID(userID, id uint) (*model.SavedMessageSearch, error) {
	item := new(model.SavedMessageSearch)
	if err := d.DB.First(item, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) SaveSavedMessageSearch(item *model.SavedMessageSearch) error {
	return d.DB.Save(item).Error
}

func (d *GormDatabase) DeleteSavedMessageSearch(userID, id uint) error {
	return d.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.SavedMessageSearch{}).Error
}

func (d *GormDatabase) SearchMessages(userID uint, filter model.MessageSearchFilter) ([]*model.Message, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := visibleMessages(d.DB, userID).
		Select("messages.*").
		Order("messages.id desc").
		Limit(limit)
	if filter.Query != "" {
		needle := "%" + strings.ToLower(strings.TrimSpace(filter.Query)) + "%"
		query = query.Where("LOWER(messages.title) LIKE ? OR LOWER(messages.message) LIKE ?", needle, needle)
	}
	if filter.ApplicationID != 0 {
		query = query.Where("messages.application_id = ?", filter.ApplicationID)
	}
	if filter.MinPriority != nil {
		query = query.Where("messages.priority >= ?", *filter.MinPriority)
	}
	if filter.MaxPriority != nil {
		query = query.Where("messages.priority <= ?", *filter.MaxPriority)
	}
	if filter.Sender != "" {
		query = query.Where("LOWER(messages.sender_name) LIKE ?", "%"+strings.ToLower(filter.Sender)+"%")
	}
	if filter.From != nil {
		query = query.Where("messages.date >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("messages.date <= ?", *filter.To)
	}
	if filter.Status != "" {
		query = query.Joins("LEFT JOIN message_workflows AS mw ON mw.message_id = messages.id").
			Where("COALESCE(mw.status, '') = ?", filter.Status)
	}
	switch filter.Acknowledged {
	case "yes":
		query = query.Joins("JOIN message_acknowledgements AS ma_search ON ma_search.message_id = messages.id")
	case "no":
		query = query.Joins("LEFT JOIN message_acknowledgements AS ma_search ON ma_search.message_id = messages.id").
			Where("ma_search.message_id IS NULL")
	}
	var items []*model.Message
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	if err := d.markAcknowledged(userID, items); err != nil {
		return nil, err
	}
	return items, nil
}


func (d *GormDatabase) GetAttachmentStorageNames() ([]string, error) {
	var names []string
	err := d.DB.Model(&model.MessageAttachment{}).Where("storage_name <> ''").Pluck("storage_name", &names).Error
	return names, err
}
