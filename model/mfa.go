package model

import "time"

type UserMFA struct {
	UserID        uint      `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	Secret        string    `gorm:"type:text" json:"-"`
	Enabled       bool      `gorm:"index" json:"enabled"`
	RecoveryCodes string    `gorm:"type:text" json:"-"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type MFAStatus struct {
	Enabled bool `json:"enabled"`
	RecoveryCodesRemaining int `json:"recoveryCodesRemaining"`
}
