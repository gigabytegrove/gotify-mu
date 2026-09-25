package model

import "time"

// WebhookRoute defines a named inbound webhook that publishes into a Channel.
type WebhookRoute struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"type:text" json:"name"`
	ApplicationID   uint      `gorm:"index" json:"applicationId"`
	Secret          string    `gorm:"type:varchar(96);uniqueIndex" json:"-"`
	Enabled         bool      `json:"enabled"`
	TitleField      string    `gorm:"type:text" json:"titleField"`
	MessageField    string    `gorm:"type:text" json:"messageField"`
	PriorityField   string    `gorm:"type:text" json:"priorityField"`
	DefaultTitle    string    `gorm:"type:text" json:"defaultTitle"`
	DefaultPriority int       `json:"defaultPriority"`
	AllowedCIDRs     string    `gorm:"type:text" json:"allowedCidrs,omitempty"`
	RequireSignature bool     `json:"requireSignature"`
	SigningSecret    string    `gorm:"type:text" json:"-"`
	MaxAgeSeconds    int       `json:"maxAgeSeconds"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// WebhookRouteView includes the generated inbound path without exposing the secret itself.
type WebhookRouteView struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	ApplicationID   uint      `json:"applicationId"`
	Enabled         bool      `json:"enabled"`
	Path            string    `json:"path"`
	TitleField      string    `json:"titleField"`
	MessageField    string    `json:"messageField"`
	PriorityField   string    `json:"priorityField"`
	DefaultTitle    string    `json:"defaultTitle"`
	DefaultPriority int       `json:"defaultPriority"`
	AllowedCIDRs     string    `json:"allowedCidrs,omitempty"`
	RequireSignature bool     `json:"requireSignature"`
	SigningSecretConfigured bool `json:"signingSecretConfigured"`
	MaxAgeSeconds    int       `json:"maxAgeSeconds"`
	SigningSecret    string    `json:"signingSecret,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// MQTTIntegration subscribes to one broker/topic and publishes received payloads into a Channel.
type MQTTIntegration struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:text" json:"name"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	BrokerURL     string    `gorm:"type:text" json:"brokerUrl"`
	ClientID      string    `gorm:"type:text" json:"clientId"`
	Username      string    `gorm:"type:text" json:"username"`
	Password      string    `gorm:"type:text" json:"-"`
	Topic         string    `gorm:"type:text" json:"topic"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// MQTTIntegrationView masks the stored password.
type MQTTIntegrationView struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	ApplicationID      uint      `json:"applicationId"`
	BrokerURL           string    `json:"brokerUrl"`
	ClientID            string    `json:"clientId"`
	Username            string    `json:"username"`
	PasswordConfigured bool      `json:"passwordConfigured"`
	Topic               string    `json:"topic"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// HomeAssistantIntegration subscribes to Home Assistant events over its WebSocket API.
type HomeAssistantIntegration struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:text" json:"name"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	BaseURL       string    `gorm:"type:text" json:"baseUrl"`
	Token         string    `gorm:"type:text" json:"-"`
	EventType     string    `gorm:"type:text" json:"eventType"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// HomeAssistantIntegrationView masks the stored access token.
type HomeAssistantIntegrationView struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	ApplicationID   uint      `json:"applicationId"`
	BaseURL         string    `json:"baseUrl"`
	TokenConfigured bool      `json:"tokenConfigured"`
	EventType       string    `json:"eventType"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ScheduledNotification is a recurring or one-time Channel notification.
type ScheduledNotification struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"type:text" json:"name"`
	ApplicationID uint       `gorm:"index" json:"applicationId"`
	Title         string     `gorm:"type:text" json:"title"`
	Message       string     `gorm:"type:text" json:"message"`
	Priority      int        `json:"priority"`
	ScheduleType  string     `gorm:"type:varchar(16)" json:"scheduleType"`
	RunAt         *time.Time `json:"runAt,omitempty"`
	Hour          int        `json:"hour"`
	Minute        int        `json:"minute"`
	Weekday       int        `json:"weekday"`
	CronExpression string    `gorm:"type:varchar(160)" json:"cronExpression,omitempty"`
	Timezone      string     `gorm:"type:text" json:"timezone"`
	EndAt         *time.Time `json:"endAt,omitempty"`
	MaxRuns       int        `json:"maxRuns"`
	RunCount      int        `json:"runCount"`
	MisfirePolicy string     `gorm:"type:varchar(16);default:send" json:"misfirePolicy"`
	ExcludeDates  string     `gorm:"type:text" json:"excludeDates,omitempty"`
	Enabled       bool       `json:"enabled"`
	LastRunAt     *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt     *time.Time `gorm:"index" json:"nextRunAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// QuietHoursPolicy controls realtime notification delivery for one user.
type QuietHoursPolicy struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint      `gorm:"uniqueIndex" json:"userId"`
	Enabled       bool      `json:"enabled"`
	StartMinute   int       `json:"startMinute"`
	EndMinute     int       `json:"endMinute"`
	Timezone      string    `gorm:"type:text" json:"timezone"`
	AllowPriority int       `json:"allowPriority"`
	Behavior      string    `gorm:"type:varchar(16);default:suppress" json:"behavior"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// DeferredNotification holds a realtime notification until Quiet Hours ends.
type DeferredNotification struct {
	UserID        uint      `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	MessageID     uint      `gorm:"primaryKey;autoIncrement:false;index" json:"messageId"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	DueAt         time.Time `gorm:"index" json:"dueAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// DigestPolicy controls summary delivery for lower-priority notifications.
type DigestPolicy struct {
	ID               uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID           uint       `gorm:"uniqueIndex" json:"userId"`
	Enabled          bool       `json:"enabled"`
	IntervalMinutes  int        `json:"intervalMinutes"`
	ImmediatePriority int       `json:"immediatePriority"`
	LastSentAt       *time.Time `json:"lastSentAt,omitempty"`
	NextRunAt        *time.Time `gorm:"index" json:"nextRunAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// DigestItem stores one notification awaiting a user's next digest.
type DigestItem struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint      `gorm:"index;uniqueIndex:uix_digest_user_message,priority:1" json:"userId"`
	MessageID     uint      `gorm:"index;uniqueIndex:uix_digest_user_message,priority:2" json:"messageId"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	Title         string    `gorm:"type:text" json:"title"`
	Message       string    `gorm:"type:text" json:"message"`
	Priority      int       `json:"priority"`
	CreatedAt     time.Time `json:"createdAt"`
}

// EscalationRule copies qualifying, unacknowledged messages to another Channel after a delay.
type EscalationRule struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string    `gorm:"type:text" json:"name"`
	SourceApplicationID uint      `gorm:"index" json:"sourceApplicationId"`
	TargetApplicationID uint      `gorm:"index" json:"targetApplicationId"`
	MinPriority         int       `json:"minPriority"`
	DelayMinutes        int       `json:"delayMinutes"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// EscalationState tracks one pending escalation.
type EscalationState struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID    uint       `gorm:"index;uniqueIndex:uix_escalation_rule_message,priority:1" json:"ruleId"`
	MessageID uint       `gorm:"index;uniqueIndex:uix_escalation_rule_message,priority:2" json:"messageId"`
	DueAt     time.Time  `gorm:"index" json:"dueAt"`
	Completed bool       `gorm:"index" json:"completed"`
	CreatedAt time.Time  `json:"createdAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
}

// MessageAcknowledgement records that a user acknowledged a message.
type MessageAcknowledgement struct {
	UserID         uint      `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	MessageID      uint      `gorm:"primaryKey;autoIncrement:false;index" json:"messageId"`
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
}
