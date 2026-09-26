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
	if err == gorm.ErrRecordNotFound { err = nil }
	if membership.ApplicationID != applicationID || membership.UserID != userID { return nil, err }
	app, appErr := d.GetApplicationByID(applicationID)
	if appErr != nil { return nil, appErr }
	membership.EffectiveRole = model.EffectiveChannelRole(app != nil && app.UserID == userID, membership)
	return membership, err
}

func (d *GormDatabase) GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error) {
	var memberships []*model.ApplicationMembership
	err := d.DB.Where("application_id = ?", applicationID).Order("user_id ASC").Find(&memberships).Error
	if err != nil { return nil, err }
	app, appErr := d.GetApplicationByID(applicationID)
	if appErr != nil { return nil, appErr }
	for _, membership := range memberships {
		membership.EffectiveRole = model.EffectiveChannelRole(app != nil && app.UserID == membership.UserID, membership)
	}
	return memberships, nil
}

func (d *GormDatabase) UpsertApplicationMembership(membership *model.ApplicationMembership) error {
	membership.AutoAssigned = false
	if membership.Role == "" { membership.Role = model.ChannelRoleMember }
	membership.Role = model.NormalizeChannelRole(membership.Role)
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"application_id"},{Name:"user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"receive_notifications","role","auto_assigned","updated_at"}),
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
		Updates(map[string]any{"receive_notifications":enabled, "notification_override":enabled})
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
		switch err {
		case nil:
			if err := tx.Model(&model.ApplicationMembership{}).
				Where("application_id = ? AND user_id = ?", applicationID, newOwnerID).
				Update("auto_assigned", false).Error; err != nil {
				return err
			}
		case gorm.ErrRecordNotFound:
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


type groupAccess struct {
	role    string
	receive bool
}

func (d *GormDatabase) GetApplicationGroupAssignments(applicationID uint) ([]*model.ApplicationGroupAssignment, error) {
	var items []*model.ApplicationGroupAssignment
	err := d.DB.Where("application_id = ?", applicationID).Order("group_id asc").Find(&items).Error
	return items, err
}

func (d *GormDatabase) UpsertApplicationGroupAssignment(item *model.ApplicationGroupAssignment) error {
	item.Role = model.NormalizeChannelRole(item.Role)
	if err := d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"application_id"},{Name:"group_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role","receive_notifications","updated_at"}),
	}).Create(item).Error; err != nil {
		return err
	}
	return d.SyncApplicationGroupAssignments(item.ApplicationID)
}

func (d *GormDatabase) DeleteApplicationGroupAssignment(applicationID, groupID uint) error {
	if err := d.DB.Where("application_id = ? AND group_id = ?", applicationID, groupID).
		Delete(&model.ApplicationGroupAssignment{}).Error; err != nil {
		return err
	}
	return d.SyncApplicationGroupAssignments(applicationID)
}

func (d *GormDatabase) SyncGroupAssignmentsForGroup(groupID uint) error {
	var appIDs []uint
	if err := d.DB.Model(&model.ApplicationGroupAssignment{}).
		Where("group_id = ?", groupID).
		Distinct("application_id").
		Pluck("application_id", &appIDs).Error; err != nil {
		return err
	}
	for _, applicationID := range appIDs {
		if err := d.SyncApplicationGroupAssignments(applicationID); err != nil { return err }
	}
	return nil
}

func (d *GormDatabase) SyncApplicationGroupAssignments(applicationID uint) error {
	app, err := d.GetApplicationByID(applicationID)
	if err != nil { return err }
	if app == nil { return nil }

	assignments, err := d.GetApplicationGroupAssignments(applicationID)
	if err != nil { return err }
	desired := make(map[uint]groupAccess)
	for _, assignment := range assignments {
		members, memberErr := d.GetUserGroupMembers(assignment.GroupID)
		if memberErr != nil { return memberErr }
		for _, user := range members {
			current := desired[user.ID]
			if model.ChannelRoleRank(assignment.Role) > model.ChannelRoleRank(current.role) {
				current.role = model.NormalizeChannelRole(assignment.Role)
			}
			current.receive = current.receive || assignment.ReceiveNotifications
			desired[user.ID] = current
		}
	}

	var memberships []*model.ApplicationMembership
	if err := d.DB.Where("application_id = ?", applicationID).Find(&memberships).Error; err != nil { return err }
	existing := make(map[uint]*model.ApplicationMembership, len(memberships))
	for _, membership := range memberships { existing[membership.UserID] = membership }

	return d.DB.Transaction(func(tx *gorm.DB) error {
		for userID, membership := range existing {
			access, assigned := desired[userID]
			if assigned {
				membership.GroupAssigned = true
				membership.GroupRole = access.role
				membership.GroupReceiveNotifications = access.receive
				if membership.NotificationOverride != nil {
					membership.ReceiveNotifications = *membership.NotificationOverride
				} else if membership.Role == "" && !membership.AutoAssigned && userID != app.UserID {
					membership.ReceiveNotifications = access.receive
				}
				if err := tx.Save(membership).Error; err != nil { return err }
				delete(desired, userID)
				continue
			}
			if !membership.GroupAssigned { continue }
			membership.GroupAssigned = false
			membership.GroupRole = ""
			membership.GroupReceiveNotifications = false
			if membership.Role == "" && !membership.AutoAssigned && userID != app.UserID {
				if err := tx.Delete(&model.ApplicationMembership{}, "application_id = ? AND user_id = ?", applicationID, userID).Error; err != nil { return err }
			} else {
				if membership.NotificationOverride != nil {
					membership.ReceiveNotifications = *membership.NotificationOverride
				}
				if err := tx.Save(membership).Error; err != nil { return err }
			}
		}
		for userID, access := range desired {
			receive := access.receive
			membership := &model.ApplicationMembership{
				ApplicationID:applicationID,
				UserID:userID,
				ReceiveNotifications:receive,
				GroupAssigned:true,
				GroupRole:access.role,
				GroupReceiveNotifications:access.receive,
			}
			if err := tx.Create(membership).Error; err != nil { return err }
		}
		return nil
	})
}
