package model

import "time"

// ApplicationMembership grants a user access to an application/channel.
// Application.UserID remains the canonical owner for upstream compatibility;
// memberships add the many-to-many access model used by Gotify MU.
type ApplicationMembership struct {
	ApplicationID        uint `gorm:"primaryKey;autoIncrement:false"`
	UserID               uint `gorm:"primaryKey;autoIncrement:false;index"`
	ReceiveNotifications bool   `gorm:"not null"`
	NotificationOverride *bool  `json:"-"`
	AutoAssigned         bool   `gorm:"not null"`
	Role                 string `gorm:"type:varchar(16)"`
	GroupRole                 string `gorm:"type:varchar(16)"`
	GroupAssigned             bool   `gorm:"not null"`
	GroupReceiveNotifications bool   `gorm:"not null"`
	EffectiveRole        string `gorm:"-" json:"-"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TableName keeps the table name stable across all supported GORM dialects.
func (ApplicationMembership) TableName() string { return "application_memberships" }


const (
	ChannelRoleReadOnly  = "readonly"
	ChannelRoleMember    = "member"
	ChannelRolePublisher = "publisher"
	ChannelRoleManager   = "manager"
	ChannelRoleOwner     = "owner"
)

func ChannelRoleRank(role string) int {
	switch role {
	case ChannelRoleOwner:
		return 5
	case ChannelRoleManager:
		return 4
	case ChannelRolePublisher:
		return 3
	case ChannelRoleMember:
		return 2
	case ChannelRoleReadOnly:
		return 1
	default:
		return 0
	}
}

func NormalizeChannelRole(role string) string {
	switch role {
	case ChannelRoleReadOnly, ChannelRoleMember, ChannelRolePublisher, ChannelRoleManager:
		return role
	default:
		return ChannelRoleMember
	}
}

func EffectiveChannelRole(owner bool, membership *ApplicationMembership) string {
	if owner { return ChannelRoleOwner }
	if membership == nil { return "" }
	best := ""
	if membership.Role != "" { best = NormalizeChannelRole(membership.Role) }
	if membership.AutoAssigned && ChannelRoleRank(best) < ChannelRoleRank(ChannelRoleMember) {
		best = ChannelRoleMember
	}
	if membership.GroupAssigned && ChannelRoleRank(membership.GroupRole) > ChannelRoleRank(best) {
		best = NormalizeChannelRole(membership.GroupRole)
	}
	if best == "" && !membership.GroupAssigned {
		best = ChannelRoleMember
	}
	return best
}

// ApplicationGroupAssignment grants a Group role on one Channel.
type ApplicationGroupAssignment struct {
	ApplicationID        uint      `gorm:"primaryKey;autoIncrement:false" json:"applicationId"`
	GroupID              uint      `gorm:"primaryKey;autoIncrement:false;index" json:"groupId"`
	Role                 string    `gorm:"type:varchar(16)" json:"role"`
	ReceiveNotifications bool      `gorm:"not null" json:"receiveNotifications"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type ApplicationGroupAssignmentExternal struct {
	GroupID              uint   `json:"groupId"`
	Name                 string `json:"name"`
	Role                 string `json:"role"`
	ReceiveNotifications bool   `json:"receiveNotifications"`
	MemberCount          int64  `json:"memberCount"`
}
