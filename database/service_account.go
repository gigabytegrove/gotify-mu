package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) CreateServiceAccount(item *model.ServiceAccount) error {
	return d.DB.Create(item).Error
}

func (d *GormDatabase) GetServiceAccounts() ([]*model.ServiceAccount, error) {
	var items []*model.ServiceAccount
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}

func (d *GormDatabase) GetServiceAccountByID(id uint) (*model.ServiceAccount, error) {
	item := new(model.ServiceAccount)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) GetServiceAccountByTokenHash(hash string) (*model.ServiceAccount, error) {
	item := new(model.ServiceAccount)
	if err := d.DB.Where("token_hash = ?", hash).First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) TouchServiceAccount(id uint, now time.Time) error {
	return d.DB.Model(&model.ServiceAccount{}).Where("id = ?", id).Update("last_used", &now).Error
}

func (d *GormDatabase) DeleteServiceAccount(id uint) error {
	return d.DB.Delete(&model.ServiceAccount{}, id).Error
}
