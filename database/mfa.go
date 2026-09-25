package database

import (
	"encoding/json"
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) GetUserMFA(userID uint) (*model.UserMFA, error) {
	item := new(model.UserMFA)
	if err := d.DB.First(item, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	secret, err := d.Secrets.Decrypt(item.Secret)
	if err != nil { return nil, err }
	item.Secret = secret
	return item, nil
}

func (d *GormDatabase) SaveUserMFA(item *model.UserMFA) error {
	secret, err := d.Secrets.Decrypt(item.Secret)
	if err != nil { return err }
	encrypted, err := d.Secrets.Encrypt(secret)
	if err != nil { return err }
	item.Secret = encrypted
	err = d.DB.Save(item).Error
	item.Secret = secret
	return err
}

func (d *GormDatabase) DeleteUserMFA(userID uint) error {
	return d.DB.Where("user_id = ?", userID).Delete(&model.UserMFA{}).Error
}

func (d *GormDatabase) ConsumeRecoveryCode(userID uint, codeHash string) (bool, error) {
	item, err := d.GetUserMFA(userID)
	if err != nil || item == nil || !item.Enabled { return false, err }
	var hashes []string
	if err := json.Unmarshal([]byte(item.RecoveryHashes), &hashes); err != nil { return false, err }
	index := -1
	for i, hash := range hashes {
		if hash == codeHash { index = i; break }
	}
	if index < 0 { return false, nil }
	hashes = append(hashes[:index], hashes[index+1:]...)
	raw, err := json.Marshal(hashes)
	if err != nil { return false, err }
	item.RecoveryHashes = string(raw)
	item.UpdatedAt = time.Now()
	return true, d.SaveUserMFA(item)
}
