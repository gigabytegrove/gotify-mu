package model

import "time"

// WebhookRoute defines a named inbound webhook that publishes into a Channel.
type WebhookRoute struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"type:text" json:"name"`
	ApplicationID   uint      `gorm:"index" json:"applicationId"`
	Secret          string    `gorm:"type:text" json:"-"`
	SecretHash      string    `gorm:"type:varchar(64);uniqueIndex" json:"-"`
	Enabled         bool      `json:"enabled"`
	RequireSignature bool     `json:"requireSignature"`
	AllowedCIDRs    string    `gorm:"type:text" json:"allowedCidrs"`
	RateLimitPerMinute int    `json:"rateLimitPerMinute"`
	TitleField      string    `gorm:"type:text" json:"titleField"`
	MessageField    string    `gorm:"type:text" json:"messageField"`
	PriorityField   string    `gorm:"type:text" json:"priorityField"`
	DefaultTitle    string    `gorm:"type:text" json:"defaultTitle"`
	DefaultPriority int       `json:"defaultPriority"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// WebhookRouteView includes the generated inbound path without exposing the secret itself.
type WebhookRouteView struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	ApplicationID   uint      `json:"applicationId"`
	Enabled         bool      `json:"enabled"`
	RequireSignature bool     `json:"requireSignature"`
	AllowedCIDRs    string    `json:"allowedCidrs"`
	RateLimitPerMinute int    `json:"rateLimitPerMinute"`
	Path            string    `json:"path"`
	TitleField      string    `json:"titleField"`
	MessageField    string    `json:"messageField"`
	PriorityField   string    `json:"priorityField"`
	DefaultTitle    string    `json:"defaultTitle"`
	DefaultPriority int       `json:"defaultPriority"`
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
	Topic           string     `gorm:"type:text" json:"topic"`
	Enabled         bool       `json:"enabled"`
	Status          string     `gorm:"type:varchar(24)" json:"status"`
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"`
	LastMessageAt   *time.Time `json:"lastMessageAt,omitempty"`
	LastError       string     `gorm:"type:text" json:"lastError,omitempty"`
	LastErrorAt     *time.Time `json:"lastErrorAt,omitempty"`
	ReconnectCount  int        `json:"reconnectCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
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
	Topic               string     `json:"topic"`
	Enabled             bool       `json:"enabled"`
	Status              string     `json:"status"`
	LastConnectedAt     *time.Time `json:"lastConnectedAt,omitempty"`
	LastMessageAt       *time.Time `json:"lastMessageAt,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	LastErrorAt         *time.Time `json:"lastErrorAt,omitempty"`
	ReconnectCount      int        `json:"reconnectCount"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// HomeAssistantIntegration subscribes to Home Assistant events over its WebSocket API.
type HomeAssistantIntegration struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:text" json:"name"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	BaseURL       string    `gorm:"type:text" json:"baseUrl"`
	Token         string    `gorm:"type:text" json:"-"`
	EventType       string     `gorm:"type:text" json:"eventType"`
	Enabled         bool       `json:"enabled"`
	Status          string     `gorm:"type:varchar(24)" json:"status"`
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"`
	LastEventAt     *time.Time `json:"lastEventAt,omitempty"`
	LastError       string     `gorm:"type:text" json:"lastError,omitempty"`
	LastErrorAt     *time.Time `json:"lastErrorAt,omitempty"`
	ReconnectCount  int        `json:"reconnectCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// HomeAssistantIntegrationView masks the stored access token.
type HomeAssistantIntegrationView struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	ApplicationID   uint      `json:"applicationId"`
	BaseURL         string    `json:"baseUrl"`
	TokenConfigured bool      `json:"tokenConfigured"`
	EventType       string     `json:"eventType"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"`
	LastEventAt     *time.Time `json:"lastEventAt,omitempty"`
	LastError       string     `json:"lastError,omitempty"`
	LastErrorAt     *time.Time `json:"lastErrorAt,omitempty"`
	ReconnectCount  int        `json:"reconnectCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// ScheduledNotification is a recurring or one-time Channel notification.
type ScheduledNotification struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"type:text" json:"name"`
	ApplicationID uint       `gorm:"index" json:"applicationId"`
	Title         string     `gorm:"type:text" json:"title"`
	Message       string     `gorm:"type:text" json:"message"`
	Priority      int        `json:"priority"`
	ScheduleType   string     `gorm:"type:varchar(16)" json:"scheduleType"`
	RunAt          *time.Time `json:"runAt,omitempty"`
	Hour           int        `json:"hour"`
	Minute         int        `json:"minute"`
	Weekday        int        `json:"weekday"`
	CronExpression string     `gorm:"type:text" json:"cronExpression,omitempty"`
	ExcludedDates  string     `gorm:"type:text" json:"excludedDates,omitempty"`
	Timezone       string     `gorm:"type:text" json:"timezone"`
	EndAt          *time.Time `json:"endAt,omitempty"`
	MaxRuns        int        `json:"maxRuns"`
	RunCount       int        `json:"runCount"`
	MisfirePolicy  string     `gorm:"type:varchar(16)" json:"misfirePolicy"`
	Enabled        bool       `json:"enabled"`
	LastRunAt      *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt      *time.Time `gorm:"index" json:"nextRunAt,omitempty"`
	LastStatus     string     `gorm:"type:varchar(24)" json:"lastStatus,omitempty"`
	LastError      string     `gorm:"type:text" json:"lastError,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
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
	Mode          string    `gorm:"type:varchar(16)" json:"mode"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
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
	TargetType          string    `gorm:"type:varchar(16)" json:"targetType"`
	TargetID            uint      `gorm:"index" json:"targetId"`
	MinPriority         int       `json:"minPriority"`
	DelayMinutes        int       `json:"delayMinutes"`
	RepeatMinutes       int       `json:"repeatMinutes"`
	MaxRepeats          int       `json:"maxRepeats"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// EscalationState tracks one pending escalation.
type EscalationState struct {
	ID                     uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID                 uint       `gorm:"index;uniqueIndex:uix_escalation_rule_message,priority:1" json:"ruleId"`
	MessageID              uint       `gorm:"index;uniqueIndex:uix_escalation_rule_message,priority:2" json:"messageId"`
	DueAt                  time.Time  `gorm:"index" json:"dueAt"`
	RepeatCount            int        `json:"repeatCount"`
	LastEscalatedMessageID uint       `gorm:"index" json:"lastEscalatedMessageId,omitempty"`
	Completed              bool       `gorm:"index" json:"completed"`
	CreatedAt              time.Time  `json:"createdAt"`
	DoneAt                 *time.Time `json:"doneAt,omitempty"`
}

// MessageAcknowledgement records that a user acknowledged a message.
type MessageAcknowledgement struct {
	UserID         uint      `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	MessageID      uint      `gorm:"primaryKey;autoIncrement:false;index" json:"messageId"`
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
}


// DeferredNotification queues one realtime notification until Quiet Hours end.
type DeferredNotification struct {
	UserID    uint      `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	MessageID uint      `gorm:"primaryKey;autoIncrement:false;index" json:"messageId"`
	CreatedAt time.Time `json:"createdAt"`
}

// AutomationLease elects one active worker for schedulers and persistent integrations.
type AutomationLease struct {
	Name      string    `gorm:"primaryKey;type:varchar(220)" json:"name"`
	Holder    string    `gorm:"type:varchar(96);index" json:"holder"`
	ExpiresAt time.Time `gorm:"index" json:"expiresAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MessageAcknowledgementView is the user-facing acknowledgement history for a message.
type MessageAcknowledgementView struct {
	UserID         uint      `json:"userId"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"displayName,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
}


// ScheduledNotificationRun records one scheduler execution attempt.
type ScheduledNotificationRun struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ScheduleID   uint       `gorm:"index" json:"scheduleId"`
	ScheduledFor time.Time  `gorm:"index" json:"scheduledFor"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	Status       string     `gorm:"type:varchar(24);index" json:"status"`
	MessageID    uint       `gorm:"index" json:"messageId,omitempty"`
	Error        string     `gorm:"type:text" json:"error,omitempty"`
}


// EscalationTargetApplication maps non-Channel escalation targets to internal history Channels.
type EscalationTargetApplication struct {
	TargetType    string    `gorm:"primaryKey;type:varchar(16)" json:"targetType"`
	TargetID      uint      `gorm:"primaryKey;autoIncrement:false" json:"targetId"`
	ApplicationID uint      `gorm:"uniqueIndex" json:"applicationId"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
