package model

import "time"

// SecurityPolicy is the singleton administrator-controlled security and retention policy.
type SecurityPolicy struct {
	ID                       uint      `gorm:"primaryKey;autoIncrement:false" json:"id"`
	MinPasswordLength        int       `json:"minPasswordLength"`
	SessionLifetimeHours     int       `json:"sessionLifetimeHours"`
	ElevationMinutes         int       `json:"elevationMinutes"`
	AuditRetentionDays       int       `json:"auditRetentionDays"`
	AutomationRetentionDays  int       `json:"automationRetentionDays"`
	AllowNativePluginUploads bool      `json:"allowNativePluginUploads"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

func DefaultSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		ID: 1,
		MinPasswordLength: 12,
		SessionLifetimeHours: 168,
		ElevationMinutes: 60,
		AuditRetentionDays: 180,
		AutomationRetentionDays: 90,
		AllowNativePluginUploads: false,
	}
}
