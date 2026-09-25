package database

import (
	"time"

	"github.com/gotify/server/v3/model"
)

// CreateAuditEvent persists an administrative/security audit event.
func (d *GormDatabase) CreateAuditEvent(event *model.AuditEvent) error {
	return d.DB.Create(event).Error
}

// GetAuditEvents returns newest audit events first, optionally filtered by action or target.
func (d *GormDatabase) GetAuditEvents(limit int, action, target string) ([]*model.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	query := d.DB.Model(&model.AuditEvent{})
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if target != "" {
		query = query.Where("target = ?", target)
	}

	var events []*model.AuditEvent
	err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&events).Error
	return events, err
}

// DeleteAuditEventsBefore removes old audit events for retention-policy support.
func (d *GormDatabase) DeleteAuditEventsBefore(before time.Time) error {
	return d.DB.Where("created_at < ?", before).Delete(&model.AuditEvent{}).Error
}
