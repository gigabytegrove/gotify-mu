package database

import (
	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) GetDirectoryConfig() (*model.DirectoryConfig, error) {
	item := new(model.DirectoryConfig)
	if err := d.DB.Order("id asc").First(item).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) SaveDirectoryConfig(item *model.DirectoryConfig) error {
	existing, err := d.GetDirectoryConfig()
	if err != nil { return err }
	if existing != nil {
		item.ID = existing.ID
		item.CreatedAt = existing.CreatedAt
	}
	return d.DB.Save(item).Error
}
