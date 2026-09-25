package model

import "time"

// ServiceCredential is a non-human credential with explicit API scopes.
type ServiceCredential struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint       `gorm:"index" json:"userId"`
	ApplicationID *uint      `gorm:"index" json:"applicationId,omitempty"`
	Name          string     `gorm:"type:varchar(180)" json:"name"`
	TokenHash     string     `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	Scopes        string     `gorm:"type:text" json:"-"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastUsed      *time.Time `json:"lastUsed,omitempty"`
	ExpiresAt     *time.Time `gorm:"index" json:"expiresAt,omitempty"`
}

// ServiceCredentialExternal is safe for management API responses.
type ServiceCredentialExternal struct {
	ID            uint       `json:"id"`
	UserID        uint       `json:"userId"`
	ApplicationID *uint      `json:"applicationId,omitempty"`
	Name          string     `json:"name"`
	Scopes        []string   `json:"scopes"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastUsed      *time.Time `json:"lastUsed,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Token         string     `json:"token,omitempty"`
}
