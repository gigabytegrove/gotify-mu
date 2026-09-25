package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetSecurityPolicy() (*model.SecurityPolicy, error) {
	item := new(model.SecurityPolicy)
	if err := d.DB.First(item, 1).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		defaults := model.DefaultSecurityPolicy()
		if createErr := d.DB.Create(defaults).Error; createErr != nil {
			if createErr != gorm.ErrDuplicatedKey {
				return nil, createErr
			}
			if reloadErr := d.DB.First(item, 1).Error; reloadErr != nil {
				return nil, reloadErr
			}
			return item, nil
		}
		return defaults, nil
	}
	return item, nil
}

func (d *GormDatabase) SaveSecurityPolicy(item *model.SecurityPolicy) error {
	item.ID = 1
	item.UpdatedAt = time.Now()
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"min_password_length","session_lifetime_hours","elevation_minutes",
			"audit_retention_days","automation_retention_days",
			"allow_native_plugin_uploads","require_plugin_checksum","require_plugin_signature",
			"require_mfa_admins","require_mfa_all","updated_at",
		}),
	}).Create(item).Error
}
