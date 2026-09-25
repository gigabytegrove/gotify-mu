package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) CreatePasskeyChallenge(item *model.PasskeyChallenge) error {
	if err := d.DB.Where("expires_at <= ?", d.DB.NowFunc()).Delete(&model.PasskeyChallenge{}).Error; err != nil {
		return err
	}
	return d.DB.Create(item).Error
}

func (d *GormDatabase) ConsumePasskeyChallenge(id, kind string, now time.Time) (*model.PasskeyChallenge, error) {
	var item model.PasskeyChallenge
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND kind = ? AND expires_at > ?", id, kind, now).First(&item).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
	if err == gorm.ErrRecordNotFound { return nil, nil }
	return &item, err
}

func (d *GormDatabase) CreatePasskeyCredential(item *model.PasskeyCredential) error {
	return d.DB.Create(item).Error
}

func (d *GormDatabase) GetPasskeyCredentialByCredentialID(id string) (*model.PasskeyCredential, error) {
	item := new(model.PasskeyCredential)
	err := d.DB.Where("credential_id = ?", id).First(item).Error
	if err == gorm.ErrRecordNotFound { return nil, nil }
	return item, err
}

func (d *GormDatabase) GetPasskeyCredentialsByUser(userID uint) ([]*model.PasskeyCredential, error) {
	var items []*model.PasskeyCredential
	err := d.DB.Where("user_id = ?", userID).Order("name ASC, id ASC").Find(&items).Error
	return items, err
}

func (d *GormDatabase) UpdatePasskeyCredentialUse(id uint, signCount uint32, now time.Time) error {
	return d.DB.Model(&model.PasskeyCredential{}).Where("id = ?", id).Updates(map[string]any{
		"sign_count": signCount,
		"last_used": now,
	}).Error
}

func (d *GormDatabase) DeletePasskeyCredential(id, userID uint) error {
	return d.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.PasskeyCredential{}).Error
}
