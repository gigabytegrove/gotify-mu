package model

import "time"

// SystemSetting stores one administrator-controlled server setting.
type SystemSetting struct {
	Key       string    `gorm:"primaryKey;type:varchar(120)" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SecurityPolicy is the effective server-side security policy.
type SecurityPolicy struct {
	MinimumPasswordLength      int  `json:"minimumPasswordLength"`
	SessionInactivityMinutes   int  `json:"sessionInactivityMinutes"`
	ElevationMinutes           int  `json:"elevationMinutes"`
	RequireMFAForAdmins        bool `json:"requireMfaForAdmins"`
	RequireMFAForAllLocalUsers bool `json:"requireMfaForAllLocalUsers"`
	AuditRetentionDays         int  `json:"auditRetentionDays"`
}

// OperationsSummary exposes non-secret administrative health and capacity data.
type OperationsSummary struct {
	Users                  int64  `json:"users"`
	Channels               int64  `json:"channels"`
	Messages               int64  `json:"messages"`
	Clients                int64  `json:"clients"`
	ConnectedClients       int    `json:"connectedClients"`
	StorageBytes           int64  `json:"storageBytes"`
	StorageFiles           int64  `json:"storageFiles"`
	AttachmentFiles        int64  `json:"attachmentFiles"`
	Plugins                int64  `json:"plugins"`
	Webhooks               int64  `json:"webhooks"`
	MQTTConnections        int64  `json:"mqttConnections"`
	HomeAssistantConnections int64 `json:"homeAssistantConnections"`
	Schedules              int64  `json:"schedules"`
	PendingEscalations     int64  `json:"pendingEscalations"`
	PendingDigests         int64  `json:"pendingDigests"`
	DeferredNotifications  int64  `json:"deferredNotifications"`
	AuditEvents            int64  `json:"auditEvents"`
	DatabaseDialect        string `json:"databaseDialect"`
}
