package database

import (
	"github.com/gotify/server/v3/fracdex"
	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func membershipConflict() clause.OnConflict {
	return clause.OnConflict{
		Columns:   []clause.Column{{Name: "application_id"}, {Name: "user_id"}},
		DoNothing: true,
	}
}

func assignApplicationToAllUsers(tx *gorm.DB, applicationID, ownerID uint) error {
	var userIDs []uint
	if err := tx.Model(&model.User{}).Pluck("id", &userIDs).Error; err != nil {
		return err
	}

	memberships := make([]model.ApplicationMembership, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID == ownerID {
			continue
		}
		memberships = append(memberships, model.ApplicationMembership{
			ApplicationID:        applicationID,
			UserID:               userID,
			ReceiveNotifications: true,
			AutoAssigned:         true,
		})
	}
	if len(memberships) == 0 {
		return nil
	}
	return tx.Clauses(membershipConflict()).Create(&memberships).Error
}

func assignUserToAutoApplications(tx *gorm.DB, userID uint) error {
	var apps []model.Application
	if err := tx.Where("auto_assign = ?", true).Find(&apps).Error; err != nil {
		return err
	}
	for _, app := range apps {
		if app.UserID == userID {
			continue
		}
		membership := model.ApplicationMembership{
			ApplicationID:        app.ID,
			UserID:               userID,
			ReceiveNotifications: true,
			AutoAssigned:         true,
		}
		if err := tx.Clauses(membershipConflict()).Create(&membership).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillApplicationMemberships(tx *gorm.DB) error {
	var apps []model.Application
	if err := tx.Find(&apps).Error; err != nil {
		return err
	}
	for _, app := range apps {
		if app.UserID != 0 {
			owner := model.ApplicationMembership{
				ApplicationID:        app.ID,
				UserID:               app.UserID,
				ReceiveNotifications: true,
			}
			if err := tx.Clauses(membershipConflict()).Create(&owner).Error; err != nil {
				return err
			}
		}
		if app.AutoAssign {
			if err := assignApplicationToAllUsers(tx, app.ID, app.UserID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *GormDatabase) GetApplicationMembership(applicationID, userID uint) (*model.ApplicationMembership, error) {
	membership := new(model.ApplicationMembership)
	err := d.DB.Where("application_id = ? AND user_id = ?", applicationID, userID).Find(membership).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if membership.ApplicationID == applicationID && membership.UserID == userID {
		return membership, err
	}
	return nil, err
}

func (d *GormDatabase) GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error) {
	var memberships []*model.ApplicationMembership
	err := d.DB.Where("application_id = ?", applicationID).Order("user_id ASC").Find(&memberships).Error
	return memberships, err
}

func (d *GormDatabase) UpsertApplicationMembership(membership *model.ApplicationMembership) error {
	membership.AutoAssigned = false
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "application_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"receive_notifications", "auto_assigned", "updated_at"}),
	}).Create(membership).Error
}

func (d *GormDatabase) DeleteApplicationMembership(applicationID, userID uint) error {
	return d.DB.Where("application_id = ? AND user_id = ?", applicationID, userID).Delete(&model.ApplicationMembership{}).Error
}

func (d *GormDatabase) CountApplicationMemberships(applicationID uint) (int64, error) {
	var count int64
	err := d.DB.Model(&model.ApplicationMembership{}).Where("application_id = ?", applicationID).Count(&count).Error
	return count, err
}

func (d *GormDatabase) GetApplicationRecipientUserIDs(applicationID uint) ([]uint, error) {
	var userIDs []uint
	err := d.DB.Model(&model.ApplicationMembership{}).
		Where("application_id = ? AND receive_notifications = ?", applicationID, true).
		Order("user_id ASC").Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// SetApplicationMembershipNotifications changes realtime delivery for one channel member
// without changing their access to channel history.
func (d *GormDatabase) SetApplicationMembershipNotifications(
	applicationID, userID uint,
	enabled bool,
) error {
	result := d.DB.Model(&model.ApplicationMembership{}).
		Where("application_id = ? AND user_id = ?", applicationID, userID).
		Update("receive_notifications", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := d.DB.Model(&model.ApplicationMembership{}).
			Where("application_id = ? AND user_id = ?", applicationID, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

// TransferApplicationOwnership changes the canonical Gotify application owner.
// The new owner is guaranteed to have a manual membership so disabling
// auto-assignment later cannot remove the owner from the channel.
func (d *GormDatabase) TransferApplicationOwnership(applicationID, newOwnerID uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var app model.Application
		if err := tx.First(&app, applicationID).Error; err != nil {
			return err
		}

		var user model.User
		if err := tx.First(&user, newOwnerID).Error; err != nil {
			return err
		}

		var membership model.ApplicationMembership
		err := tx.Where(
			"application_id = ? AND user_id = ?",
			applicationID,
			newOwnerID,
		).First(&membership).Error
		switch {
		case err == nil:
			if err := tx.Model(&model.ApplicationMembership{}).
				Where("application_id = ? AND user_id = ?", applicationID, newOwnerID).
				Update("auto_assigned", false).Error; err != nil {
				return err
			}
		case err == gorm.ErrRecordNotFound:
			membership = model.ApplicationMembership{
				ApplicationID:        applicationID,
				UserID:               newOwnerID,
				ReceiveNotifications: true,
				AutoAssigned:         false,
			}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
		default:
			return err
		}

		lastSortKey := ""
		err = tx.Model(&model.Application{}).
			Select("sort_key").
			Where("user_id = ? AND id <> ?", newOwnerID, applicationID).
			Order("sort_key DESC").
			Limit(1).
			Find(&lastSortKey).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		newSortKey, err := fracdex.KeyBetween(lastSortKey, "")
		if err != nil {
			return err
		}

		return tx.Model(&model.Application{}).
			Where("id = ?", applicationID).
			Updates(map[string]any{
				"user_id":  newOwnerID,
				"sort_key": newSortKey,
			}).Error
	})
}

// SetApplicationMemberPosting controls whether channel members may publish
// using their normal user/client authentication.
func (d *GormDatabase) SetApplicationMemberPosting(applicationID uint, enabled bool) error {
	return d.DB.Model(&model.Application{}).
		Where("id = ?", applicationID).
		Update("allow_member_post", enabled).Error
}

// SetApplicationMembersCanPost controls whether any channel member may
// publish using client-token authentication.
func (d *GormDatabase) SetApplicationMembersCanPost(applicationID uint, enabled bool) error {
	return d.DB.Model(&model.Application{}).
		Where("id = ?", applicationID).
		Update("members_can_post", enabled).Error
}

func (d *GormDatabase) SetApplicationAutoAssign(applicationID uint, enabled bool) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var app model.Application
		if err := tx.First(&app, applicationID).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Application{}).Where("id = ?", applicationID).Update("auto_assign", enabled).Error; err != nil {
			return err
		}
		if enabled {
			return assignApplicationToAllUsers(tx, applicationID, app.UserID)
		}
		return tx.Where("application_id = ? AND auto_assigned = ?", applicationID, true).
			Delete(&model.ApplicationMembership{}).Error
	})
}
