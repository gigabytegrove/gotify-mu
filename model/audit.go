package model

import "time"

// AuditEvent records security-sensitive and administrative actions.
// Details must never contain plaintext secrets, tokens, passwords, recovery codes, or plugin binaries.
type AuditEvent struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	Username  string    `gorm:"type:text" json:"username"`
	Action    string    `gorm:"type:varchar(120);index" json:"action"`
	Target    string    `gorm:"type:varchar(120);index" json:"target"`
	TargetID  string    `gorm:"type:varchar(180)" json:"targetId,omitempty"`
	Details   string    `gorm:"type:text" json:"details,omitempty"`
	IPAddress string    `gorm:"type:varchar(180)" json:"ipAddress,omitempty"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}
