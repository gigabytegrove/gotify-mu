package model

import "time"

// UserMFA stores TOTP enrollment and one-time recovery code hashes.
type UserMFA struct {
	UserID         uint       `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	Secret         string     `gorm:"type:text" json:"-"`
	RecoveryHashes string     `gorm:"type:text" json:"-"`
	Enabled        bool       `json:"enabled"`
	EnrolledAt     *time.Time `json:"enrolledAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// MFAStatus is safe to return to the current user or administrators.
type MFAStatus struct {
	Enabled        bool       `json:"enabled"`
	EnrolledAt     *time.Time `json:"enrolledAt,omitempty"`
	RecoveryCodes  int        `json:"recoveryCodes"`
}

// MFASetupResult is returned only when a new secret is generated.
type MFASetupResult struct {
	Secret        string   `json:"secret"`
	ProvisioningURI string `json:"provisioningUri"`
	RecoveryCodes []string `json:"recoveryCodes"`
}
