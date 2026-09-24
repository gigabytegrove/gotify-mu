package model

import "time"

// MessageDismissal records that one user removed a shared message from their
// own view. The underlying message remains available to other channel members.
type MessageDismissal struct {
	UserID    uint `gorm:"primaryKey;autoIncrement:false"`
	MessageID uint `gorm:"primaryKey;autoIncrement:false;index"`
	CreatedAt time.Time
}

// TableName keeps the table name stable across all supported GORM dialects.
func (MessageDismissal) TableName() string { return "message_dismissals" }
