package model

// MonitaInfo Model
//
// swagger:model MonitaInfo
type MonitaInfo struct {
	// The current version.
	//
	// required: true
	Version string `json:"version"`
	// Whether OIDC authentication is enabled.
	Oidc bool `json:"oidc"`
	// Whether user registration is enabled.
	Register bool `json:"register"`
	// Whether local username/password authentication is enabled.
	LocalAuth bool `json:"localAuth"`
	// Configured OIDC provider display name.
	OIDCIDPName string `json:"oidcIdpName"`
	// Whether OIDC login should start automatically.
	OIDCAutoRedirect bool `json:"oidcAutoRedirect"`
	// Whether LDAP authentication is enabled.
	LDAP bool `json:"ldap"`
	// Configured LDAP provider display name.
	LDAPIDPName string `json:"ldapIdpName"`
}
