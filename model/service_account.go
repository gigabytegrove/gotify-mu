package model

import "time"

type ServiceAccount struct {
	ID                    uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                  string     `gorm:"type:varchar(180);uniqueIndex" json:"name"`
	Scopes                string     `gorm:"type:text" json:"scopes"`
	AllowedApplicationIDs string     `gorm:"type:text" json:"allowedApplicationIds"`
	Enabled               bool       `json:"enabled"`
	CreatedBy             uint       `gorm:"index" json:"createdBy"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
	LastUsedAt            *time.Time `json:"lastUsedAt,omitempty"`
}

type ServiceAccountToken struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	AccountID   uint       `gorm:"index" json:"accountId"`
	TokenHash   string     `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	TokenPrefix string     `gorm:"type:varchar(24);index" json:"tokenPrefix"`
	Name        string     `gorm:"type:text" json:"name"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   *time.Time `gorm:"index" json:"expiresAt,omitempty"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

type ServiceAccountTokenResult struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token"`
	Prefix    string     `json:"prefix"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}
