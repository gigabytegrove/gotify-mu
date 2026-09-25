package database

import (
	"fmt"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

// GetUserByName returns the user by the given name or nil.
func (d *GormDatabase) GetUserByName(name string) (*model.User, error) {
	user := new(model.User)
	err := d.DB.Where("name = ?", name).Find(user).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if user.Name == name {
		return user, err
	}
	return nil, err
}

// GetUserByOIDC returns the user bound to the given oidc id or nil.
func (d *GormDatabase) GetUserByOIDC(oidcID string) (*model.User, error) {
	user := new(model.User)
	err := d.DB.Where("oidc_id = ?", oidcID).Find(user).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if user.OIDCID != nil && *user.OIDCID == oidcID {
		return user, err
	}
	return nil, err
}

// GetUserByID returns the user by the given id or nil.
func (d *GormDatabase) GetUserByID(id uint) (*model.User, error) {
	user := new(model.User)
	err := d.DB.Find(user, id).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if user.ID == id {
		return user, err
	}
	return nil, err
}

// CountUser returns the user count which satisfies the given condition.
func (d *GormDatabase) CountUser(condition ...any) (int64, error) {
	c := int64(-1)
	handle := d.DB.Model(new(model.User))
	if len(condition) == 1 {
		handle = handle.Where(condition[0])
	} else if len(condition) > 1 {
		handle = handle.Where(condition[0], condition[1:]...)
	}
	err := handle.Count(&c).Error
	return c, err
}

// GetUsers returns all users.
func (d *GormDatabase) GetUsers() ([]*model.User, error) {
	var users []*model.User
	err := d.DB.Find(&users).Error
	return users, err
}

// DeleteUserByID deletes a user by its id.
func (d *GormDatabase) DeleteUserByID(id uint) error {
	apps, err := d.GetApplicationsByUser(id)
	if err != nil {
		return err
	}
	for _, app := range apps {
		memberCount, err := d.CountApplicationMemberships(app.ID)
		if err != nil {
			return err
		}
		if memberCount > 1 {
			return fmt.Errorf(
				"cannot delete user %d: shared channel %q must be transferred first",
				id,
				app.Name,
			)
		}
	}
	for _, app := range apps {
		if err := d.DeleteApplicationByID(app.ID); err != nil {
			return err
		}
	}
	clients, _ := d.GetClientsByUser(id)
	for _, client := range clients {
		d.DeleteClientByID(client.ID)
	}
	pluginConfs, _ := d.GetPluginConfByUser(id)
	for _, conf := range pluginConfs {
		d.DeletePluginConfByID(conf.ID)
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.ApplicationMembership{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.MessageDismissal{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.MessageAcknowledgement{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.DigestItem{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.DigestPolicy{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.QuietHoursPolicy{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.UserMFA{}).Error; err != nil {
		return err
	}
	if err := d.DB.Where("user_id = ?", id).Delete(&model.DeferredNotification{}).Error; err != nil {
		return err
	}
	if err := d.DeleteUserGroupMembershipsForUser(id); err != nil {
		return err
	}
	return d.DB.Where("id = ?", id).Delete(&model.User{}).Error
}

// UpdateUser updates a user.
func (d *GormDatabase) UpdateUser(user *model.User) error {
	return d.DB.Save(user).Error
}

// CreateUser creates a user.
func (d *GormDatabase) CreateUser(user *model.User) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return assignUserToAutoApplications(tx, user.ID)
	})
}
