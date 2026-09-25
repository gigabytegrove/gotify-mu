package database

import (
	"database/sql"
	"time"

	"github.com/gotify/server/v3/fracdex"
	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

// GetApplicationByToken returns the application for the given token or nil.
func (d *GormDatabase) GetApplicationByToken(token string) (*model.Application, error) {
	app := new(model.Application)
	err := d.DB.Where("token = ?", token).Find(app).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if app.Token == token {
		return app, err
	}
	return nil, err
}

// GetApplicationByID returns the application for the given id or nil.
func (d *GormDatabase) GetApplicationByID(id uint) (*model.Application, error) {
	app := new(model.Application)
	err := d.DB.Where("id = ?", id).Find(app).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if app.ID == id {
		return app, err
	}
	return nil, err
}

// CreateApplication creates an application.
func (d *GormDatabase) CreateApplication(application *model.Application) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if application.SortKey == "" {
			sortKey := ""
			err := tx.Model(&model.Application{}).Select("sort_key").Where("user_id = ?", application.UserID).Order("sort_key DESC").Limit(1).Find(&sortKey).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			application.SortKey, err = fracdex.KeyBetween(sortKey, "")
			if err != nil {
				return err
			}
		}

		if err := tx.Create(application).Error; err != nil {
			return err
		}

		if application.UserID != 0 {
			membership := &model.ApplicationMembership{
				ApplicationID:        application.ID,
				UserID:               application.UserID,
				ReceiveNotifications: true,
				Role:                 model.ChannelRoleOwner,
			}
			if err := tx.Create(membership).Error; err != nil {
				return err
			}
		}

		if application.AutoAssign {
			return assignApplicationToAllUsers(tx, application.ID, application.UserID)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

// DeleteApplicationByID deletes an application by its id.
func (d *GormDatabase) DeleteApplicationByID(id uint) error {
	if err := d.DeleteMessagesByApplication(id); err != nil {
		return err
	}
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var escalationRuleIDs []uint
		if err := tx.Model(&model.EscalationRule{}).
			Where("source_application_id = ? OR target_application_id = ?", id, id).
			Pluck("id", &escalationRuleIDs).Error; err != nil {
			return err
		}
		if len(escalationRuleIDs) > 0 {
			if err := tx.Where("rule_id IN ?", escalationRuleIDs).Delete(&model.EscalationState{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", escalationRuleIDs).Delete(&model.EscalationRule{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("application_id = ?", id).Delete(&model.WebhookRoute{}).Error; err != nil { return err }
		if err := tx.Where("application_id = ?", id).Delete(&model.MQTTIntegration{}).Error; err != nil { return err }
		if err := tx.Where("application_id = ?", id).Delete(&model.HomeAssistantIntegration{}).Error; err != nil { return err }
		if err := tx.Where("application_id = ?", id).Delete(&model.ScheduledNotification{}).Error; err != nil { return err }
		if err := tx.Where("application_id = ?", id).Delete(&model.DigestItem{}).Error; err != nil { return err }
		if err := tx.Where("application_id = ?", id).Delete(&model.ApplicationMembership{}).Error; err != nil { return err }
		if err := tx.Where("application_id = ?", id).Delete(&model.ApplicationGroupAssignment{}).Error; err != nil { return err }
		return tx.Where("id = ?", id).Delete(&model.Application{}).Error
	})
}

// GetApplicationsByUser returns all applications from a user.
func (d *GormDatabase) GetApplicationsByUser(userID uint) ([]*model.Application, error) {
	var apps []*model.Application
	err := d.DB.Where("user_id = ?", userID).Order("sort_key, id ASC").Find(&apps).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return apps, err
}

// UpdateApplication updates an application.
// GetAccessibleApplicationsByUser returns every application that the user owns
// or has been assigned to through an ApplicationMembership.
func (d *GormDatabase) GetAccessibleApplicationsByUser(userID uint) ([]*model.Application, error) {
	var apps []*model.Application
	err := d.DB.Where(
		"EXISTS (SELECT 1 FROM application_memberships am WHERE am.application_id = applications.id AND am.user_id = ?) OR "+
			"EXISTS (SELECT 1 FROM application_group_assignments aga JOIN user_group_memberships ugm ON ugm.group_id = aga.group_id WHERE aga.application_id = applications.id AND ugm.user_id = ?)",
		userID, userID,
	).Order("applications.sort_key, applications.id ASC").Find(&apps).Error
	if err == gorm.ErrRecordNotFound { err = nil }
	return apps, err
}

// UpdateApplication updates an application.
func (d *GormDatabase) UpdateApplication(app *model.Application) error {
	return d.DB.Save(app).Error
}

// UpdateApplicationTokenLastUsed updates the last used time of the application token.
func (d *GormDatabase) UpdateApplicationTokenLastUsed(token string, t *time.Time) error {
	return d.DB.Model(&model.Application{}).Where("token = ?", token).Update("last_used", t).Error
}
