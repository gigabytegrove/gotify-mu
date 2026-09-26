package model

import (
	"time"
)

// Message holds information about a message.
type Message struct {
	ID            uint `gorm:"autoIncrement;primaryKey;index"`
	ApplicationID uint
	Message       string `gorm:"type:text"`
	Title         string `gorm:"type:text"`
	Priority      int
	Extras        []byte
	Date          time.Time
	SenderUserID  uint `gorm:"index"`
	SenderName    string `gorm:"type:text"`
	Acknowledged        bool                      `gorm:"-" json:"-"`
	AcknowledgedAny     bool                      `gorm:"-" json:"-"`
	AcknowledgementCount int                      `gorm:"-" json:"-"`
	AcknowledgedBy      []MessageAcknowledgementView `gorm:"-" json:"-"`
	ParentMessageID     uint                      `gorm:"index" json:"-"`
	EscalationRuleID    uint                      `gorm:"index" json:"-"`
}

// MessageExternal Model
//
// The MessageExternal holds information about a message which was sent by an Application.
//
// swagger:model Message
type MessageExternal struct {
	// The message id.
	//
	// read only: true
	// required: true
	// example: 25
	ID uint `json:"id"`
	// The application id that send this message.
	//
	// read only: true
	// required: true
	// example: 5
	ApplicationID uint `form:"appid" query:"appid" json:"appid"`
	// The message. Markdown (excluding html) is allowed.
	//
	// required: true
	// example: **Backup** was successfully finished.
	Message string `form:"message" query:"message" json:"message" binding:"required"`
	// The title of the message.
	//
	// example: Backup
	Title string `form:"title" query:"title" json:"title"`
	// The priority of the message. If unset, then the default priority of the
	// application will be used.
	//
	// example: 2
	Priority *int `form:"priority" query:"priority" json:"priority"`
	// The extra data sent along the message.
	//
	// The extra fields are stored in a key-value scheme. Only accepted in CreateMessage requests with application/json content-type.
	//
	// The keys should be in the following format: &lt;top-namespace&gt;::[&lt;sub-namespace&gt;::]&lt;action&gt;
	//
	// These namespaces are reserved and might be used in the official clients: gotify android ios web server client. Do not use them for other purposes.
	//
	// example: {"home::appliances::thermostat::change_temperature":{"temperature":23},"home::appliances::lighting::on":{"brightness":15}}
	Extras map[string]any `form:"-" query:"-" json:"extras,omitempty"`
	// The date the message was created.
	//
	// read only: true
	// required: true
	// example: 2018-02-27T19:36:10.5045044+01:00
	Date time.Time `json:"date"`
	// The Gotify MU user id that posted this message, when the message was sent by a user.
	//
	// read only: true
	SenderUserID uint `json:"senderUserId,omitempty"`
	// The Gotify MU username that posted this message, when available.
	//
	// read only: true
	SenderName string `json:"senderName,omitempty"`
	// Whether the current requesting user has acknowledged this message.
	Acknowledged bool `json:"acknowledged,omitempty"`
	// Whether anyone has acknowledged this message.
	AcknowledgedAny bool `json:"acknowledgedAny,omitempty"`
	// Number of users who acknowledged this message.
	AcknowledgementCount int `json:"acknowledgementCount,omitempty"`
	// Users who acknowledged this message.
	AcknowledgedBy []MessageAcknowledgementView `json:"acknowledgedBy,omitempty"`
	// Original message id when this message was created by an escalation.
	ParentMessageID uint `json:"parentMessageId,omitempty"`
	// Escalation rule that created this message.
	EscalationRuleID uint `json:"escalationRuleId,omitempty"`
}

// CreateMessage Model
//
// The CreateMessage holds information about a message that will be sent.
//
// swagger:model CreateMessage
type CreateMessage struct {
	// The application id that send this message. Always set when returned via the API.
	//
	// example: 5
	ApplicationID uint `form:"appid" query:"appid" json:"appid"`
	// The message. Markdown (excluding html) is allowed.
	//
	// required: true
	// example: **Backup** was successfully finished.
	Message string `form:"message" query:"message" json:"message" binding:"required"`
	// The title of the message.
	//
	// example: Backup
	Title string `form:"title" query:"title" json:"title"`
	// The priority of the message. If unset, then the default priority of the
	// application will be used.
	//
	// example: 2
	Priority *int `form:"priority" query:"priority" json:"priority"`
	// The extra data sent along the message.
	//
	// The extra fields are stored in a key-value scheme. Only accepted in CreateMessage requests with application/json content-type.
	//
	// The keys should be in the following format: &lt;top-namespace&gt;::[&lt;sub-namespace&gt;::]&lt;action&gt;
	//
	// These namespaces are reserved and might be used in the official clients: gotify android ios web server client. Do not use them for other purposes.
	//
	// example: {"home::appliances::thermostat::change_temperature":{"temperature":23},"home::appliances::lighting::on":{"brightness":15}}
	Extras map[string]any `form:"-" query:"-" json:"extras,omitempty"`
}


// MessageAcknowledgementView is user-facing acknowledgement history.
type MessageAcknowledgementView struct {
	UserID         uint      `json:"userId"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"displayName,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
}
