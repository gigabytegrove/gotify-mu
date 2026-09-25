package model

import "time"

// The User holds information about the credentials of a user and its application and client tokens.
type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	Name         string `gorm:"type:varchar(180);uniqueIndex:uix_users_name"`
	DisplayName  string `gorm:"type:text"`
	Pass         []byte
	Admin        bool
	CreatedAt    time.Time
	Applications []Application
	Clients      []Client
	Plugins      []PluginConf
	// Format: OIDC claims combined as "<iss>#<sub>".
	OIDCID *string `gorm:"column:oidc_id;type:text;uniqueIndex:uix_users_oidc_id,length:512"`
	// LDAPID stores the normalized directory DN for linked LDAP/Active Directory accounts.
	LDAPID *string `gorm:"column:ldap_id;type:text;uniqueIndex:uix_users_ldap_id,length:1024"`
}

// UserExternal Model
//
// The User holds information about permission and other stuff.
//
// swagger:model User
type UserExternal struct {
	// The user id.
	//
	// read only: true
	// required: true
	// example: 25
	ID uint `json:"id"`
	// The user name. For login.
	//
	// required: true
	// example: unicorn
	Name string `binding:"required" json:"name" query:"name" form:"name"`
	// Friendly display name. Login continues to use Name.
	DisplayName string `json:"displayName,omitempty" query:"displayName" form:"displayName"`
	// If the user is an administrator.
	//
	// required: true
	// example: true
	Admin bool `json:"admin" form:"admin" query:"admin"`
	// The date the user was created.
	//
	// read only: true
	// required: true
	// example: 2019-01-01T00:00:00Z
	CreatedAt time.Time `json:"createdAt"`
}

// CreateUserExternal Model
//
// Used for user creation.
//
// swagger:model CreateUserExternal
type CreateUserExternal struct {
	// The user name. For login.
	//
	// required: true
	// example: unicorn
	Name string `binding:"required" json:"name" query:"name" form:"name"`
	// Friendly display name.
	DisplayName string `json:"displayName,omitempty" query:"displayName" form:"displayName"`
	// If the user is an administrator.
	//
	// required: true
	// example: true
	Admin bool `json:"admin" form:"admin" query:"admin"`
	// The user password. For login.
	//
	// required: true
	// example: nrocinu
	Pass string `json:"pass,omitempty" form:"pass" query:"pass" binding:"required"`
}

// UpdateUserExternal Model
//
// Used for updating a user.
//
// swagger:model UpdateUserExternal
type UpdateUserExternal struct {
	// The user name. For login.
	//
	// required: true
	// example: unicorn
	Name string `binding:"required" json:"name" query:"name" form:"name"`
	// Friendly display name.
	DisplayName string `json:"displayName,omitempty" query:"displayName" form:"displayName"`
	// If the user is an administrator.
	//
	// required: true
	// example: true
	Admin bool `json:"admin" form:"admin" query:"admin"`
	// The user password. For login. Empty for using old password
	//
	// example: nrocinu
	Pass string `json:"pass,omitempty" form:"pass" query:"pass"`
}

// CurrentUserExternal Model
//
// swagger:model CurrentUser
type CurrentUserExternal struct {
	// The user id.
	//
	// read only: true
	// required: true
	// example: 25
	ID uint `json:"id"`
	// The user name. For login.
	//
	// required: true
	// example: unicorn
	Name string `json:"name"`
	// Friendly display name.
	DisplayName string `json:"displayName,omitempty"`
	// If the user is an administrator.
	//
	// required: true
	// example: true
	Admin bool `json:"admin"`
	// The date the user was created.
	//
	// read only: true
	// required: true
	// example: 2019-01-01T00:00:00Z
	CreatedAt time.Time `json:"createdAt"`
	// The client id of the current session.
	//
	// read only: true
	// example: 5
	ClientID uint `json:"clientId,omitempty"`
	// The time until which the session is elevated.
	//
	// read only: true
	ElevatedUntil *time.Time `json:"elevatedUntil,omitempty"`
	MFAEnabled  bool   `json:"mfaEnabled"`
	MFARequired bool   `json:"mfaRequired"`
	AuthProvider string `json:"authProvider,omitempty"`
}

// UserExternalPass Model
//
// The Password for updating the user.
//
// swagger:model UserPass
type UserExternalPass struct {
	// The user password. For login.
	//
	// required: true
	// example: nrocinu
	Pass string `json:"pass,omitempty" form:"pass" query:"pass" binding:"required"`
}
