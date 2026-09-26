package model

import "time"

// PasskeyCredential stores one WebAuthn credential for a user.
type PasskeyCredential struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint       `gorm:"index" json:"userId"`
	Name         string     `gorm:"type:text" json:"name"`
	CredentialID string     `gorm:"type:varchar(1024);uniqueIndex" json:"credentialId"`
	PublicKeyX   []byte     `json:"-"`
	PublicKeyY   []byte     `json:"-"`
	SignCount    uint32     `json:"signCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
}

// WebAuthnChallenge stores a short-lived registration/login/elevation challenge.
type WebAuthnChallenge struct {
	Challenge  string    `gorm:"primaryKey;type:varchar(180)"`
	UserID     uint      `gorm:"index"`
	Purpose    string    `gorm:"type:varchar(24);index"`
	RPID       string    `gorm:"type:text"`
	Origin     string    `gorm:"type:text"`
	ClientName string    `gorm:"type:text"`
	ClientID   uint
	ExpiresAt  time.Time `gorm:"index"`
	CreatedAt  time.Time
}

type PasskeyView struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}
