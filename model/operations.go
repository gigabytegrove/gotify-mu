package model

import "time"

type AdminSession struct {
	ID uint `json:"id"`
	UserID uint `json:"userId"`
	Username string `json:"username"`
	DisplayName string `json:"displayName,omitempty"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsed *time.Time `json:"lastUsed,omitempty"`
	ElevatedUntil *time.Time `json:"elevatedUntil,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type SystemStats struct {
	Users int64 `json:"users"`
	Channels int64 `json:"channels"`
	Messages int64 `json:"messages"`
	Clients int64 `json:"clients"`
	Groups int64 `json:"groups"`
	Webhooks int64 `json:"webhooks"`
	MQTTConnections int64 `json:"mqttConnections"`
	HomeAssistantConnections int64 `json:"homeAssistantConnections"`
	Schedules int64 `json:"schedules"`
	EscalationRules int64 `json:"escalationRules"`
	AuditEvents int64 `json:"auditEvents"`
	AutomationRuns int64 `json:"automationRuns"`
	DatabaseBytes int64 `json:"databaseBytes,omitempty"`
	DataBytes int64 `json:"dataBytes,omitempty"`
}
