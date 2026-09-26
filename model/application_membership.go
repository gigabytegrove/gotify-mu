package model

import "time"

// ApplicationMembership grants a user access to an application/channel.
// Application.UserID remains the canonical owner for upstream compatibility;
// memberships add the many-to-many access model used by Gotify MU.
type ApplicationMembership struct {
	ApplicationID        uint `gorm:"primaryKey;autoIncrement:false"`
	UserID               uint `gorm:"primaryKey;autoIncrement:false;index"`
	ReceiveNotifications bool   `gorm:"not null"`
	AutoAssigned         bool   `gorm:"not null"`
	Role                 string `gorm:"type:varchar(24);not null;default:member" json:"role"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TableName keeps the table name stable across all supported GORM dialects.
func (ApplicationMembership) TableName() string { return "application_memberships" }

const (
	ChannelRoleOwner = "owner"
	ChannelRoleManager = "manager"
	ChannelRolePublisher = "publisher"
	ChannelRoleMember = "member"
	ChannelRoleReadOnly = "read_only"
)

func ValidChannelRole(role string) bool {
	switch role {
	case ChannelRoleOwner, ChannelRoleManager, ChannelRolePublisher, ChannelRoleMember, ChannelRoleReadOnly:
		return true
	default:
		return false
	}
}

type ApplicationGroupAssignment struct {
	ApplicationID        uint      `gorm:"primaryKey;autoIncrement:false" json:"applicationId"`
	GroupID              uint      `gorm:"primaryKey;autoIncrement:false;index" json:"groupId"`
	Role                 string    `gorm:"type:varchar(24);not null;default:member" json:"role"`
	ReceiveNotifications bool      `gorm:"not null;default:true" json:"receiveNotifications"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}
