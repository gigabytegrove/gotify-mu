package model

import "time"

// UserMFA stores local-account TOTP configuration and one-time recovery hashes.
type UserMFA struct {
	UserID         uint      `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	Enabled        bool      `json:"enabled"`
	Secret         string    `gorm:"type:text" json:"-"`
	RecoveryHashes string    `gorm:"type:text" json:"-"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type MFAStatus struct {
	Enabled           bool `json:"enabled"`
	RecoveryRemaining int  `json:"recoveryRemaining"`
}
