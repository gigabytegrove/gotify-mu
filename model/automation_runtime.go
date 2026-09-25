package model

import "time"

// AutomationLease elects one active worker for a scheduler or persistent integration.
// The key is stable across instances while Owner identifies the process holding the lease.
type AutomationLease struct {
	Key       string    `gorm:"primaryKey;type:varchar(220)" json:"key"`
	Owner     string    `gorm:"type:varchar(220);index" json:"owner"`
	ExpiresAt time.Time `gorm:"index" json:"expiresAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AutomationRun records scheduled/integration automation execution history.
type AutomationRun struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Kind       string     `gorm:"type:varchar(48);index" json:"kind"`
	ObjectID   uint       `gorm:"index" json:"objectId"`
	TriggerKey string     `gorm:"type:varchar(220);uniqueIndex" json:"triggerKey"`
	Status     string     `gorm:"type:varchar(32);index" json:"status"`
	MessageID  uint       `gorm:"index" json:"messageId,omitempty"`
	Error      string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt  time.Time  `gorm:"index" json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// IntegrationStatus exposes runtime connection health without mixing ephemeral state
// into the administrator's saved integration configuration.
type IntegrationStatus struct {
	Kind            string     `gorm:"primaryKey;type:varchar(48)" json:"kind"`
	ObjectID        uint       `gorm:"primaryKey;autoIncrement:false" json:"objectId"`
	State           string     `gorm:"type:varchar(32);index" json:"state"`
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"`
	LastActivityAt  *time.Time `json:"lastActivityAt,omitempty"`
	LastError       string     `gorm:"type:text" json:"lastError,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}
