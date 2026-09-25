package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) CreateServiceCredential(item *model.ServiceCredential) error {
	return d.DB.Create(item).Error
}

func (d *GormDatabase) GetServiceCredentialByHash(hash string) (*model.ServiceCredential, error) {
	item := new(model.ServiceCredential)
	err := d.DB.Where("token_hash = ? AND (expires_at IS NULL OR expires_at > ?)", hash, d.DB.NowFunc()).First(item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, err
}

func (d *GormDatabase) GetServiceCredentialsByUser(userID uint) ([]*model.ServiceCredential, error) {
	var items []*model.ServiceCredential
	err := d.DB.Where("user_id = ?", userID).Order("name ASC, id ASC").Find(&items).Error
	return items, err
}

func (d *GormDatabase) DeleteServiceCredential(id, userID uint) error {
	return d.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.ServiceCredential{}).Error
}

func (d *GormDatabase) TouchServiceCredential(id uint, now time.Time) error {
	return d.DB.Model(&model.ServiceCredential{}).Where("id = ?", id).Update("last_used", now).Error
}

func (d *GormDatabase) CleanupExpiredServiceCredentials(now time.Time) error {
	return d.DB.Where("expires_at IS NOT NULL AND expires_at <= ?", now).Delete(&model.ServiceCredential{}).Error
}
