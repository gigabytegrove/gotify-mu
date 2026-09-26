package database

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	settingMinimumPasswordLength      = "security.minimum_password_length"
	settingSessionInactivityMinutes   = "security.session_inactivity_minutes"
	settingElevationMinutes           = "security.elevation_minutes"
	settingRequireMFAForAdmins        = "security.require_mfa_admins"
	settingRequireMFAForAllLocalUsers = "security.require_mfa_all_local"
	settingAuditRetentionDays         = "security.audit_retention_days"
)

func defaultSecurityPolicy() model.SecurityPolicy {
	return model.SecurityPolicy{
		MinimumPasswordLength:      12,
		SessionInactivityMinutes:   10080,
		ElevationMinutes:           60,
		RequireMFAForAdmins:        false,
		RequireMFAForAllLocalUsers: false,
		AuditRetentionDays:         90,
	}
}

func (d *GormDatabase) GetSystemSetting(key string) (string, bool, error) {
	var item model.SystemSetting
	result := d.DB.First(&item, "key = ?", key)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, result.Error
	}
	return item.Value, true, nil
}

func (d *GormDatabase) SaveSystemSetting(key, value string) error {
	item := &model.SystemSetting{Key: key, Value: value}
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) GetSecurityPolicy() (model.SecurityPolicy, error) {
	policy := defaultSecurityPolicy()
	settings := map[string]func(string){
		settingMinimumPasswordLength: func(v string) {
			if n, err := strconv.Atoi(v); err == nil { policy.MinimumPasswordLength = n }
		},
		settingSessionInactivityMinutes: func(v string) {
			if n, err := strconv.Atoi(v); err == nil { policy.SessionInactivityMinutes = n }
		},
		settingElevationMinutes: func(v string) {
			if n, err := strconv.Atoi(v); err == nil { policy.ElevationMinutes = n }
		},
		settingRequireMFAForAdmins: func(v string) {
			if b, err := strconv.ParseBool(v); err == nil { policy.RequireMFAForAdmins = b }
		},
		settingRequireMFAForAllLocalUsers: func(v string) {
			if b, err := strconv.ParseBool(v); err == nil { policy.RequireMFAForAllLocalUsers = b }
		},
		settingAuditRetentionDays: func(v string) {
			if n, err := strconv.Atoi(v); err == nil { policy.AuditRetentionDays = n }
		},
	}
	for key, apply := range settings {
		value, ok, err := d.GetSystemSetting(key)
		if err != nil { return policy, err }
		if ok { apply(value) }
	}
	return policy, nil
}

func (d *GormDatabase) SaveSecurityPolicy(policy model.SecurityPolicy) error {
	values := map[string]string{
		settingMinimumPasswordLength: strconv.Itoa(policy.MinimumPasswordLength),
		settingSessionInactivityMinutes: strconv.Itoa(policy.SessionInactivityMinutes),
		settingElevationMinutes: strconv.Itoa(policy.ElevationMinutes),
		settingRequireMFAForAdmins: strconv.FormatBool(policy.RequireMFAForAdmins),
		settingRequireMFAForAllLocalUsers: strconv.FormatBool(policy.RequireMFAForAllLocalUsers),
		settingAuditRetentionDays: strconv.Itoa(policy.AuditRetentionDays),
	}
	return d.DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			item := &model.SystemSetting{Key:key, Value:value}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name:"key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value","updated_at"}),
			}).Create(item).Error; err != nil { return err }
		}
		return nil
	})
}

func (d *GormDatabase) GetOperationsSummary(dialect string) (model.OperationsSummary, error) {
	result := model.OperationsSummary{DatabaseDialect:dialect}
	counts := []struct {
		model any
		target *int64
		where string
		args []any
	}{
		{&model.User{}, &result.Users, "", nil},
		{&model.Application{}, &result.Channels, "internal = ?", []any{false}},
		{&model.Message{}, &result.Messages, "", nil},
		{&model.Client{}, &result.Clients, "", nil},
		{&model.PluginConf{}, &result.Plugins, "", nil},
		{&model.WebhookRoute{}, &result.Webhooks, "", nil},
		{&model.MQTTIntegration{}, &result.MQTTConnections, "", nil},
		{&model.HomeAssistantIntegration{}, &result.HomeAssistantConnections, "", nil},
		{&model.ScheduledNotification{}, &result.Schedules, "", nil},
		{&model.EscalationState{}, &result.PendingEscalations, "completed = ?", []any{false}},
		{&model.DigestItem{}, &result.PendingDigests, "", nil},
		{&model.DeferredNotification{}, &result.DeferredNotifications, "", nil},
		{&model.AuditEvent{}, &result.AuditEvents, "", nil},
	}
	for _, item := range counts {
		query := d.DB.Model(item.model)
		if item.where != "" { query = query.Where(item.where, item.args...) }
		if err := query.Count(item.target).Error; err != nil { return result, err }
	}
	return result, nil
}


func (d *GormDatabase) CreateBackupSnapshot(destination string) error {
	if d.DB.Name() != "sqlite" {
		return errors.New("online backup bundles currently require SQLite")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	_ = os.Remove(destination)
	clean := strings.ReplaceAll(destination, "'", "''")
	return d.DB.Exec("VACUUM INTO '" + clean + "'").Error
}
