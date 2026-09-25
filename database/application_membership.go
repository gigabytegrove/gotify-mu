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
			Role:                 model.ApplicationRoleMember,
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
			Role:                 model.ApplicationRoleMember,
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
				Role:                 model.ApplicationRoleManager,
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
	var direct *model.ApplicationMembership
	membership := new(model.ApplicationMembership)
	err := d.DB.Where("application_id = ? AND user_id = ?", applicationID, userID).First(membership).Error
	switch {
	case err == nil:
		direct = membership
	case err == gorm.ErrRecordNotFound:
		direct = nil
	default:
		return nil, err
	}

	grants, err := d.GetApplicationGroupGrantsForUser(applicationID, userID)
	if err != nil {
		return nil, err
	}
	if direct == nil && len(grants) == 0 {
		return nil, nil
	}

	result := &model.ApplicationMembership{
		ApplicationID:        applicationID,
		UserID:               userID,
		ReceiveNotifications: false,
		Role:                 model.ApplicationRoleReadOnly,
	}
	if direct != nil {
		*result = *direct
		if result.Role == "" {
			result.Role = model.ApplicationRoleMember
		}
	}
	for _, grant := range grants {
		if roleRank(grant.Role) > roleRank(result.Role) {
			result.Role = grant.Role
		}
		if direct == nil && grant.ReceiveNotifications {
			result.ReceiveNotifications = true
		}
	}

	var preference model.ApplicationNotificationPreference
	prefErr := d.DB.Where("application_id = ? AND user_id = ?", applicationID, userID).First(&preference).Error
	switch {
	case prefErr == nil:
		result.ReceiveNotifications = preference.Enabled
	case prefErr == gorm.ErrRecordNotFound:
	default:
		return nil, prefErr
	}
	return result, nil
}

func (d *GormDatabase) GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error) {
	var userIDs []uint
	if err := d.DB.Raw(`
		SELECT user_id FROM application_memberships WHERE application_id = ?
		UNION
		SELECT ugm.user_id
		FROM application_group_grants agg
		JOIN user_group_memberships ugm ON ugm.group_id = agg.group_id
		WHERE agg.application_id = ?
		ORDER BY user_id
	`, applicationID, applicationID).Scan(&userIDs).Error; err != nil {
		return nil, err
	}
	memberships := make([]*model.ApplicationMembership, 0, len(userIDs))
	for _, userID := range userIDs {
		membership, err := d.GetApplicationMembership(applicationID, userID)
		if err != nil {
			return nil, err
		}
		if membership != nil {
			memberships = append(memberships, membership)
		}
	}
	return memberships, nil
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
	err := d.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT user_id FROM application_memberships WHERE application_id = ?
			UNION
			SELECT ugm.user_id
			FROM application_group_grants agg
			JOIN user_group_memberships ugm ON ugm.group_id = agg.group_id
			WHERE agg.application_id = ?
		) AS effective_members
	`, applicationID, applicationID).Scan(&count).Error
	return count, err
}

func (d *GormDatabase) GetApplicationRecipientUserIDs(applicationID uint) ([]uint, error) {
	memberships, err := d.GetApplicationMemberships(applicationID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint, 0, len(memberships))
	for _, membership := range memberships {
		if membership.ReceiveNotifications {
			userIDs = append(userIDs, membership.UserID)
		}
	}
	return userIDs, nil
}

// SetApplicationMembershipNotifications changes realtime delivery for one channel member
// without changing their access to channel history.
func (d *GormDatabase) SetApplicationMembershipNotifications(
	applicationID, userID uint,
	enabled bool,
) error {
	membership, err := d.GetApplicationMembership(applicationID, userID)
	if err != nil {
		return err
	}
	if membership == nil {
		return gorm.ErrRecordNotFound
	}
	preference := &model.ApplicationNotificationPreference{
		ApplicationID: applicationID,
		UserID:        userID,
		Enabled:       enabled,
	}
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"application_id"},{Name:"user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled","updated_at"}),
	}).Create(preference).Error
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


func roleRank(role string) int {
	switch role {
	case model.ApplicationRoleManager:
		return 4
	case model.ApplicationRolePublisher:
		return 3
	case model.ApplicationRoleMember:
		return 2
	case model.ApplicationRoleReadOnly:
		return 1
	default:
		return 0
	}
}
