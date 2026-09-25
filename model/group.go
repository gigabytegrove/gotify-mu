package model

import "time"

// UserGroup groups users for administration and future Channel assignment/policy rules.
type UserGroup struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(180);uniqueIndex:uix_user_groups_name" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UserGroupMembership links users to administrative groups.
type UserGroupMembership struct {
	GroupID   uint      `gorm:"primaryKey;autoIncrement:false" json:"groupId"`
	UserID    uint      `gorm:"primaryKey;autoIncrement:false;index" json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserGroupExternal includes member count for UI/admin use.
type UserGroupExternal struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MemberCount int64     `json:"memberCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UserGroupMemberExternal describes one group member.
type UserGroupMemberExternal struct {
	UserID      uint   `json:"userId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Admin       bool   `json:"admin"`
}


// ApplicationGroupGrant grants a User Group access to a Channel.
type ApplicationGroupGrant struct {
	ApplicationID        uint      `gorm:"primaryKey;autoIncrement:false;index" json:"applicationId"`
	GroupID              uint      `gorm:"primaryKey;autoIncrement:false;index" json:"groupId"`
	Role                 string    `gorm:"type:varchar(24);not null;default:'member'" json:"role"`
	ReceiveNotifications bool      `gorm:"not null" json:"receiveNotifications"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}
