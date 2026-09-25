package model

import "time"

type EmailGateway struct {
	ID                  uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string     `gorm:"type:text" json:"name"`
	SourceApplicationID uint       `gorm:"index" json:"sourceApplicationId"`
	SMTPHost            string     `gorm:"type:text" json:"smtpHost"`
	SMTPPort            int        `json:"smtpPort"`
	TLSMode             string     `gorm:"type:varchar(16)" json:"tlsMode"`
	Username            string     `gorm:"type:text" json:"username"`
	Password            string     `gorm:"type:text" json:"-"`
	FromAddress         string     `gorm:"type:text" json:"fromAddress"`
	ToAddresses         string     `gorm:"type:text" json:"toAddresses"`
	MinPriority         int        `json:"minPriority"`
	Enabled             bool       `json:"enabled"`
	Status              string     `gorm:"type:varchar(24)" json:"status"`
	LastSentAt          *time.Time `json:"lastSentAt,omitempty"`
	LastError           string     `gorm:"type:text" json:"lastError,omitempty"`
	LastErrorAt         *time.Time `json:"lastErrorAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type SMTPRoute struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:text" json:"name"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	Recipient     string    `gorm:"type:text;index" json:"recipient"`
	AllowedCIDRs  string    `gorm:"type:text" json:"allowedCidrs"`
	SenderContains string   `gorm:"type:text" json:"senderContains"`
	SubjectContains string  `gorm:"type:text" json:"subjectContains"`
	MaxMessageBytes int     `json:"maxMessageBytes"`
	Username      string    `gorm:"type:text" json:"username"`
	Password      string    `gorm:"type:text" json:"-"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type RSSMonitor struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"type:text" json:"name"`
	ApplicationID uint       `gorm:"index" json:"applicationId"`
	URL           string     `gorm:"type:text" json:"url"`
	IntervalMinutes int      `json:"intervalMinutes"`
	TitleContains  string     `gorm:"type:text" json:"titleContains"`
	CategoryContains string   `gorm:"type:text" json:"categoryContains"`
	Priority       int        `json:"priority"`
	Enabled       bool       `json:"enabled"`
	Status        string     `gorm:"type:varchar(24)" json:"status"`
	ETag          string     `gorm:"type:text" json:"-"`
	LastModified  string     `gorm:"type:text" json:"-"`
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	LastItemAt    *time.Time `json:"lastItemAt,omitempty"`
	LastError     string     `gorm:"type:text" json:"lastError,omitempty"`
	LastErrorAt   *time.Time `json:"lastErrorAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type SyslogRoute struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"type:text" json:"name"`
	ApplicationID uint      `gorm:"index" json:"applicationId"`
	Facility      int       `json:"facility"`
	MaxSeverity   int       `json:"maxSeverity"`
	AllowedCIDRs  string    `gorm:"type:text" json:"allowedCidrs"`
	DeduplicateSeconds int   `json:"deduplicateSeconds"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CalendarMonitor struct {
	ID               uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string     `gorm:"type:text" json:"name"`
	ApplicationID    uint       `gorm:"index" json:"applicationId"`
	URL              string     `gorm:"type:text" json:"url"`
	IntervalMinutes  int        `json:"intervalMinutes"`
	NotifyBeforeMinutes int     `json:"notifyBeforeMinutes"`
	TitleContains      string    `gorm:"type:text" json:"titleContains"`
	LocationContains   string    `gorm:"type:text" json:"locationContains"`
	Priority           int       `json:"priority"`
	Enabled          bool       `json:"enabled"`
	Status           string     `gorm:"type:varchar(24)" json:"status"`
	LastCheckedAt    *time.Time `json:"lastCheckedAt,omitempty"`
	LastEventAt      *time.Time `json:"lastEventAt,omitempty"`
	LastError        string     `gorm:"type:text" json:"lastError,omitempty"`
	LastErrorAt      *time.Time `json:"lastErrorAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// ConnectorSeenItem prevents a polled RSS/iCal item from being emitted repeatedly.
type ConnectorSeenItem struct {
	ConnectorType string    `gorm:"primaryKey;type:varchar(24)"`
	ConnectorID   uint      `gorm:"primaryKey;autoIncrement:false"`
	ItemKey       string    `gorm:"primaryKey;type:varchar(220)"`
	SeenAt        time.Time `gorm:"index"`
}

type EmailGatewayView struct {
	EmailGateway
	PasswordConfigured bool `json:"passwordConfigured"`
}

type SMTPRouteView struct {
	SMTPRoute
	PasswordConfigured bool `json:"passwordConfigured"`
}
