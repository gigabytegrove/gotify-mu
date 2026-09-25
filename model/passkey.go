package model

import "time"

// PasskeyCredential stores one WebAuthn credential for a user.
type PasskeyCredential struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint       `gorm:"index" json:"userId"`
	Name         string     `gorm:"type:varchar(180)" json:"name"`
	CredentialID string     `gorm:"type:text;uniqueIndex" json:"-"`
	PublicKeyX   string     `gorm:"type:text" json:"-"`
	PublicKeyY   string     `gorm:"type:text" json:"-"`
	SignCount    uint32     `json:"signCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsed     *time.Time `json:"lastUsed,omitempty"`
}

// PasskeyChallenge is a short-lived registration or authentication challenge.
type PasskeyChallenge struct {
	ID        string    `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	Challenge string    `gorm:"type:varchar(128)" json:"-"`
	Kind      string    `gorm:"type:varchar(16);index" json:"-"`
	RPID      string    `gorm:"type:text" json:"-"`
	Origin    string    `gorm:"type:text" json:"-"`
	ExpiresAt time.Time `gorm:"index" json:"-"`
	CreatedAt time.Time `json:"-"`
}
