package database

import (
	"errors"
	"sort"

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
			Role:                 model.ChannelRoleMember,
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
				Role:                 model.ChannelRoleOwner,
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
	err := d.DB.Where("application_id = ? AND user_id = ?", applicationID, userID).First(membership).Error
	if err == nil {
		if membership.Role == "" { membership.Role = model.ChannelRoleMember }
		return membership, nil
	}
	if err != gorm.ErrRecordNotFound { return nil, err }

	var assignments []model.ApplicationGroupAssignment
	err = d.DB.Table("application_group_assignments AS aga").
		Joins("JOIN user_group_memberships ugm ON ugm.group_id = aga.group_id AND ugm.user_id = ?", userID).
		Where("aga.application_id = ?", applicationID).
		Find(&assignments).Error
	if err != nil { return nil, err }
	if len(assignments) == 0 { return nil, nil }

	effective := &model.ApplicationMembership{
		ApplicationID: applicationID,
		UserID: userID,
		Role: model.ChannelRoleReadOnly,
	}
	best := -1
	for _, assignment := range assignments {
		if assignment.ReceiveNotifications { effective.ReceiveNotifications = true }
		if rank := channelRoleRank(assignment.Role); rank > best {
			best = rank
			effective.Role = assignment.Role
		}
	}
	return effective, nil
}

func (d *GormDatabase) GetApplicationMemberships(applicationID uint) ([]*model.ApplicationMembership, error) {
	var direct []*model.ApplicationMembership
	if err := d.DB.Where("application_id = ?", applicationID).Order("user_id ASC").Find(&direct).Error; err != nil {
		return nil, err
	}
	merged := make(map[uint]*model.ApplicationMembership, len(direct))
	for _, item := range direct {
		if item.Role == "" { item.Role = model.ChannelRoleMember }
		merged[item.UserID] = item
	}

	type groupRow struct {
		UserID uint
		Role string
		ReceiveNotifications bool
	}
	var rows []groupRow
	if err := d.DB.Table("application_group_assignments AS aga").
		Select("ugm.user_id, aga.role, aga.receive_notifications").
		Joins("JOIN user_group_memberships ugm ON ugm.group_id = aga.group_id").
		Where("aga.application_id = ?", applicationID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if existing, ok := merged[row.UserID]; ok {
			if row.ReceiveNotifications { existing.ReceiveNotifications = true }
			if channelRoleRank(row.Role) > channelRoleRank(existing.Role) { existing.Role = row.Role }
			continue
		}
		merged[row.UserID] = &model.ApplicationMembership{
			ApplicationID:applicationID, UserID:row.UserID,
			ReceiveNotifications:row.ReceiveNotifications, Role:row.Role,
		}
	}
	result := make([]*model.ApplicationMembership, 0, len(merged))
	for _, item := range merged { result = append(result, item) }
	sort.Slice(result, func(i,j int) bool { return result[i].UserID < result[j].UserID })
	return result, nil
}

func (d *GormDatabase) UpsertApplicationMembership(membership *model.ApplicationMembership) error {
	membership.AutoAssigned = false
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "application_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"receive_notifications", "auto_assigned", "role", "updated_at"}),
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
	var direct []uint
	if err := d.DB.Model(&model.ApplicationMembership{}).
		Where("application_id = ? AND receive_notifications = ?", applicationID, true).
		Pluck("user_id", &direct).Error; err != nil { return nil, err }

	var grouped []uint
	if err := d.DB.Table("application_group_assignments AS aga").
		Distinct("ugm.user_id").
		Joins("JOIN user_group_memberships ugm ON ugm.group_id = aga.group_id").
		Where("aga.application_id = ? AND aga.receive_notifications = ?", applicationID, true).
		Pluck("ugm.user_id", &grouped).Error; err != nil { return nil, err }

	set := make(map[uint]struct{}, len(direct)+len(grouped))
	for _, id := range direct { set[id]=struct{}{} }
	for _, id := range grouped { set[id]=struct{}{} }
	out := make([]uint,0,len(set))
	for id := range set { out=append(out,id) }
	sort.Slice(out,func(i,j int)bool{return out[i]<out[j]})
	return out,nil
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


func channelRoleRank(role string) int {
	switch role {
	case model.ChannelRoleOwner: return 5
	case model.ChannelRoleManager: return 4
	case model.ChannelRolePublisher: return 3
	case model.ChannelRoleMember: return 2
	case model.ChannelRoleReadOnly: return 1
	default: return 0
	}
}

func (d *GormDatabase) GetApplicationGroupAssignments(applicationID uint) ([]*model.ApplicationGroupAssignment, error) {
	var items []*model.ApplicationGroupAssignment
	return items, d.DB.Where("application_id = ?", applicationID).Order("group_id asc").Find(&items).Error
}

func (d *GormDatabase) UpsertApplicationGroupAssignment(item *model.ApplicationGroupAssignment) error {
	if !model.ValidChannelRole(item.Role) || item.Role == model.ChannelRoleOwner {
		return errors.New("invalid Group Channel role")
	}
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"application_id"},{Name:"group_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role","receive_notifications","updated_at"}),
	}).Create(item).Error
}

func (d *GormDatabase) DeleteApplicationGroupAssignment(applicationID, groupID uint) error {
	return d.DB.Where("application_id = ? AND group_id = ?", applicationID, groupID).
		Delete(&model.ApplicationGroupAssignment{}).Error
}
