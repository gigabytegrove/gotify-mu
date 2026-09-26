package database

import (
	"errors"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetUserGroups() ([]*model.UserGroup, error) {
	var groups []*model.UserGroup
	err := d.DB.Order("name ASC").Find(&groups).Error
	return groups, err
}

func (d *GormDatabase) GetUserGroupByID(id uint) (*model.UserGroup, error) {
	group := new(model.UserGroup)
	err := d.DB.First(group, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return group, err
}

func (d *GormDatabase) CreateUserGroup(group *model.UserGroup) error {
	return d.DB.Create(group).Error
}

func (d *GormDatabase) UpdateUserGroup(group *model.UserGroup) error {
	return d.DB.Save(group).Error
}

func (d *GormDatabase) DeleteUserGroup(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", id).Delete(&model.UserGroupMembership{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", id).Delete(&model.ApplicationGroupAssignment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.UserGroup{}, id).Error
	})
}

func (d *GormDatabase) CountUserGroupMembers(groupID uint) (int64, error) {
	var count int64
	err := d.DB.Model(&model.UserGroupMembership{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func (d *GormDatabase) GetUserGroupMembers(groupID uint) ([]*model.User, error) {
	var users []*model.User
	err := d.DB.
		Joins("JOIN user_group_memberships ugm ON ugm.user_id = users.id").
		Where("ugm.group_id = ?", groupID).
		Order("users.name ASC").
		Find(&users).Error
	return users, err
}

func (d *GormDatabase) AddUserGroupMember(groupID, userID uint) error {
	var group model.UserGroup
	if err := d.DB.First(&group, groupID).Error; err != nil {
		return err
	}
	var user model.User
	if err := d.DB.First(&user, userID).Error; err != nil {
		return err
	}

	membership := model.UserGroupMembership{GroupID: groupID, UserID: userID}
	return d.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&membership).Error
}

func (d *GormDatabase) RemoveUserGroupMember(groupID, userID uint) error {
	return d.DB.Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.UserGroupMembership{}).Error
}

func (d *GormDatabase) DeleteUserGroupMembershipsForUser(userID uint) error {
	return d.DB.Where("user_id = ?", userID).Delete(&model.UserGroupMembership{}).Error
}
