package database

import (
	"encoding/json"
	"errors"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) GetUserMFA(userID uint) (*model.UserMFA, error) {
	item := new(model.UserMFA)
	if err := d.DB.First(item, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) SaveUserMFA(item *model.UserMFA) error {
	return d.DB.Save(item).Error
}

func (d *GormDatabase) DeleteUserMFA(userID uint) error {
	return d.DB.Where("user_id = ?", userID).Delete(&model.UserMFA{}).Error
}

func (d *GormDatabase) ConsumeMFARecoveryCode(userID uint, hash string) (bool, error) {
	var used bool
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		item := new(model.UserMFA)
		if err := tx.First(item, "user_id = ?", userID).Error; err != nil { return err }
		var hashes []string
		if item.RecoveryCodes != "" {
			if err := json.Unmarshal([]byte(item.RecoveryCodes), &hashes); err != nil { return err }
		}
		for i, candidate := range hashes {
			if candidate == hash {
				hashes = append(hashes[:i], hashes[i+1:]...)
				encoded, _ := json.Marshal(hashes)
				item.RecoveryCodes = string(encoded)
				used = true
				return tx.Save(item).Error
			}
		}
		return nil
	})
	return used, err
}

func (d *GormDatabase) CountUsersWithoutMFA(adminOnly bool) (int64, error) {
	query := d.DB.Table("users").
		Joins("LEFT JOIN user_mfas ON user_mfas.user_id = users.id AND user_mfas.enabled = ?", true).
		Where("user_mfas.user_id IS NULL")
	if adminOnly { query = query.Where("users.admin = ?", true) }
	var count int64
	return count, query.Count(&count).Error
}
