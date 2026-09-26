package model

import "time"

// RSSIntegration polls an RSS/Atom feed and publishes newly discovered entries.
type RSSIntegration struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text" json:"name"`
	ApplicationID uint `gorm:"index" json:"applicationId"`
	URL string `gorm:"type:text" json:"url"`
	PollMinutes int `json:"pollMinutes"`
	TitlePrefix string `gorm:"type:text" json:"titlePrefix"`
	Enabled bool `json:"enabled"`
	ETag string `gorm:"type:text" json:"-"`
	LastModified string `gorm:"type:text" json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CalendarIntegration polls an iCalendar feed and emits reminders.
type CalendarIntegration struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text" json:"name"`
	ApplicationID uint `gorm:"index" json:"applicationId"`
	URL string `gorm:"type:text" json:"url"`
	PollMinutes int `json:"pollMinutes"`
	AdvanceMinutes int `json:"advanceMinutes"`
	Enabled bool `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// EmailGateway sends qualifying Channel messages through an SMTP server.
type EmailGateway struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text" json:"name"`
	ApplicationID uint `gorm:"index" json:"applicationId"`
	Host string `gorm:"type:text" json:"host"`
	Port int `json:"port"`
	UseTLS bool `json:"useTls"`
	StartTLS bool `json:"startTls"`
	Username string `gorm:"type:text" json:"username"`
	Password string `gorm:"type:text" json:"-"`
	FromAddress string `gorm:"type:text" json:"fromAddress"`
	ToAddresses string `gorm:"type:text" json:"toAddresses"`
	MinPriority int `json:"minPriority"`
	Enabled bool `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EmailGatewayView struct {
	ID uint `json:"id"`
	Name string `json:"name"`
	ApplicationID uint `json:"applicationId"`
	Host string `json:"host"`
	Port int `json:"port"`
	UseTLS bool `json:"useTls"`
	StartTLS bool `json:"startTls"`
	Username string `json:"username"`
	PasswordConfigured bool `json:"passwordConfigured"`
	FromAddress string `json:"fromAddress"`
	ToAddresses string `json:"toAddresses"`
	MinPriority int `json:"minPriority"`
	Enabled bool `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SMTPReceiver configures the built-in authenticated SMTP ingress listener.
type SMTPReceiver struct {
	ID uint `gorm:"primaryKey;autoIncrement:false" json:"id"`
	ListenAddress string `gorm:"type:text" json:"listenAddress"`
	Username string `gorm:"type:text" json:"username"`
	Password string `gorm:"type:text" json:"-"`
	AllowedCIDRs string `gorm:"type:text" json:"allowedCidrs"`
	MaxMessageBytes int64 `json:"maxMessageBytes"`
	Enabled bool `json:"enabled"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SMTPReceiverView struct {
	ID uint `json:"id"`
	ListenAddress string `json:"listenAddress"`
	Username string `json:"username"`
	PasswordConfigured bool `json:"passwordConfigured"`
	AllowedCIDRs string `json:"allowedCidrs"`
	MaxMessageBytes int64 `json:"maxMessageBytes"`
	Enabled bool `json:"enabled"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SMTPRoute struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	Recipient string `gorm:"type:varchar(320);uniqueIndex" json:"recipient"`
	ApplicationID uint `gorm:"index" json:"applicationId"`
	Enabled bool `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SyslogReceiver listens for RFC3164/RFC5424-style syslog messages.
type SyslogReceiver struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text" json:"name"`
	ApplicationID uint `gorm:"index" json:"applicationId"`
	ListenAddress string `gorm:"type:text" json:"listenAddress"`
	Protocol string `gorm:"type:varchar(8)" json:"protocol"`
	AllowedCIDRs string `gorm:"type:text" json:"allowedCidrs"`
	MinSeverity int `json:"minSeverity"`
	Enabled bool `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
