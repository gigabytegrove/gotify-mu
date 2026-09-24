package model

import "time"

// ApplicationMembership grants a user access to an application/channel.
// Application.UserID remains the canonical owner for upstream compatibility;
// memberships add the many-to-many access model used by Gotify MU.
type ApplicationMembership struct {
	ApplicationID        uint `gorm:"primaryKey;autoIncrement:false"`
	UserID               uint `gorm:"primaryKey;autoIncrement:false;index"`
	ReceiveNotifications bool `gorm:"not null"`
	AutoAssigned         bool `gorm:"not null"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TableName keeps the table name stable across all supported GORM dialects.
func (ApplicationMembership) TableName() string { return "application_memberships" }
