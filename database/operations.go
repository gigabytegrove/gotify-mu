package database

import (
	"github.com/gotify/server/v3/model"
)

func (d *GormDatabase) GetAdminSessions() ([]*model.AdminSession, error) {
	var items []*model.AdminSession
	err := d.DB.Table("clients").
		Select("clients.id, clients.user_id, users.name AS username, users.display_name, clients.name, clients.created_at, clients.last_used, clients.elevated_until, clients.expires_at").
		Joins("JOIN users ON users.id = clients.user_id").
		Where("clients.expires_at IS NULL OR clients.expires_at > ?", d.DB.NowFunc()).
		Order("clients.last_used DESC, clients.created_at DESC").
		Scan(&items).Error
	return items, err
}

func (d *GormDatabase) GetSystemStats() (*model.SystemStats, error) {
	stats := new(model.SystemStats)
	counts := []struct{
		model any
		dest *int64
	}{
		{&model.User{}, &stats.Users},
		{&model.Application{}, &stats.Channels},
		{&model.Message{}, &stats.Messages},
		{&model.Client{}, &stats.Clients},
		{&model.UserGroup{}, &stats.Groups},
		{&model.WebhookRoute{}, &stats.Webhooks},
		{&model.MQTTIntegration{}, &stats.MQTTConnections},
		{&model.HomeAssistantIntegration{}, &stats.HomeAssistantConnections},
		{&model.ScheduledNotification{}, &stats.Schedules},
		{&model.EscalationRule{}, &stats.EscalationRules},
		{&model.AuditEvent{}, &stats.AuditEvents},
		{&model.AutomationRun{}, &stats.AutomationRuns},
	}
	for _, item := range counts {
		if err := d.DB.Model(item.model).Count(item.dest).Error; err != nil { return nil, err }
	}
	return stats, nil
}
