package model

import "time"

// ServiceAccount is a non-interactive credential with explicit API scopes and Channel restrictions.
type ServiceAccount struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string     `gorm:"type:text" json:"name"`
	UserID     uint       `gorm:"index" json:"userId"`
	TokenHash  string     `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	Scopes     string     `gorm:"type:text" json:"scopes"`
	ChannelIDs string     `gorm:"type:text" json:"channelIds"`
	LastUsed   *time.Time `json:"lastUsed,omitempty"`
	ExpiresAt  *time.Time `gorm:"index" json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}
