package model

import "time"

// DirectoryConfig configures LDAP/Active Directory authentication.
type DirectoryConfig struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Enabled            bool      `json:"enabled"`
	URL                string    `gorm:"type:text" json:"url"`
	StartTLS           bool      `json:"startTls"`
	BindDN             string    `gorm:"type:text" json:"bindDn"`
	BindPassword       string    `gorm:"type:text" json:"-"`
	UserBaseDN         string    `gorm:"type:text" json:"userBaseDn"`
	UserAttribute      string    `gorm:"type:varchar(120)" json:"userAttribute"`
	DisplayNameAttribute string  `gorm:"type:varchar(120)" json:"displayNameAttribute"`
	AdminGroupDN       string    `gorm:"type:text" json:"adminGroupDn"`
	AutoRegister       bool      `json:"autoRegister"`
	LinkByUsername     bool      `json:"linkByUsername"`
	CACertificatePEM   string    `gorm:"type:text" json:"caCertificatePem,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type DirectoryConfigView struct {
	ID                   uint      `json:"id"`
	Enabled              bool      `json:"enabled"`
	URL                  string    `json:"url"`
	StartTLS             bool      `json:"startTls"`
	BindDN               string    `json:"bindDn"`
	BindPasswordConfigured bool    `json:"bindPasswordConfigured"`
	UserBaseDN           string    `json:"userBaseDn"`
	UserAttribute        string    `json:"userAttribute"`
	DisplayNameAttribute string    `json:"displayNameAttribute"`
	AdminGroupDN         string    `json:"adminGroupDn"`
	AutoRegister         bool      `json:"autoRegister"`
	LinkByUsername       bool      `json:"linkByUsername"`
	CACertificatePEM     string    `json:"caCertificatePem,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}
