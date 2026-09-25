package model

import "time"

type MessageAttachment struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID   uint      `gorm:"index" json:"messageId"`
	Filename    string    `gorm:"type:text" json:"filename"`
	ContentType string    `gorm:"type:varchar(255)" json:"contentType"`
	Size        int64     `json:"size"`
	StorageName string    `gorm:"type:varchar(255);uniqueIndex" json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}

type MessageAttachmentView struct {
	ID          uint   `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
}

type MessageReaction struct {
	MessageID uint      `gorm:"primaryKey;autoIncrement:false;index" json:"messageId"`
	UserID    uint      `gorm:"primaryKey;autoIncrement:false;index" json:"userId"`
	Emoji     string    `gorm:"primaryKey;type:varchar(64);autoIncrement:false" json:"emoji"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessageReactionSummary struct {
	Emoji     string `json:"emoji"`
	Count     int    `json:"count"`
	ReactedByMe bool `json:"reactedByMe"`
}

type MessageWorkflow struct {
	MessageID      uint       `gorm:"primaryKey;autoIncrement:false" json:"messageId"`
	AssignedUserID uint       `gorm:"index" json:"assignedUserId,omitempty"`
	Status         string     `gorm:"type:varchar(24);index" json:"status"`
	ResolvedBy     uint       `json:"resolvedBy,omitempty"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type MessageRead struct {
	UserID    uint      `gorm:"primaryKey;autoIncrement:false;index" json:"userId"`
	MessageID uint      `gorm:"primaryKey;autoIncrement:false;index" json:"messageId"`
	ReadAt    time.Time `json:"readAt"`
}

type MessageMention struct {
	MessageID uint `gorm:"primaryKey;autoIncrement:false;index"`
	UserID    uint `gorm:"primaryKey;autoIncrement:false;index"`
}

type MessageTemplate struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:varchar(180)" json:"name"`
	UserID        uint      `gorm:"index" json:"userId"`
	ApplicationID uint      `gorm:"index" json:"applicationId,omitempty"`
	Title         string    `gorm:"type:text" json:"title"`
	Message       string    `gorm:"type:text" json:"message"`
	Priority      int       `json:"priority"`
	Extras        []byte    `gorm:"type:text" json:"-"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type MessageTemplateView struct {
	ID            uint           `json:"id"`
	Name          string         `json:"name"`
	ApplicationID uint           `json:"applicationId,omitempty"`
	Title         string         `json:"title"`
	Message       string         `json:"message"`
	Priority      int            `json:"priority"`
	Extras        map[string]any `json:"extras,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type MessageCollaboration struct {
	Attachments      []MessageAttachmentView  `json:"attachments,omitempty"`
	Reactions        []MessageReactionSummary `json:"reactions,omitempty"`
	AssignedUserID   uint                     `json:"assignedUserId,omitempty"`
	AssignedUserName string                   `json:"assignedUserName,omitempty"`
	Status           string                   `json:"status,omitempty"`
	ResolvedBy       uint                     `json:"resolvedBy,omitempty"`
	ResolvedByName   string                   `json:"resolvedByName,omitempty"`
	ResolvedAt       *time.Time               `json:"resolvedAt,omitempty"`
	Read             bool                     `json:"read,omitempty"`
	Mentioned        bool                     `json:"mentioned,omitempty"`
	ReplyCount       int                      `json:"replyCount,omitempty"`
}
