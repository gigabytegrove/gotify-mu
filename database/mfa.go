package database

import (
	"strings"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) GetUserMFA(userID uint) (*model.UserMFA, error) {
	item := new(model.UserMFA)
	err := d.DB.Where("user_id = ?", userID).First(item).Error
	if err == gorm.ErrRecordNotFound { return nil, nil }
	if err != nil { return nil, err }
	plain, err := d.secrets.Decrypt(item.Secret)
	if err != nil { return nil, err }
	item.Secret = plain
	return item, nil
}

func (d *GormDatabase) SaveUserMFA(item *model.UserMFA) error {
	plain := item.Secret
	encrypted, err := d.secrets.Encrypt(plain)
	if err != nil { return err }
	item.Secret = encrypted
	err = d.DB.Save(item).Error
	item.Secret = plain
	return err
}

func (d *GormDatabase) DisableUserMFA(userID uint) error {
	return d.DB.Where("user_id = ?", userID).Delete(&model.UserMFA{}).Error
}

func (d *GormDatabase) ConsumeUserMFARecoveryHash(userID uint, target string) (bool, error) {
	item, err := d.GetUserMFA(userID)
	if err != nil || item == nil { return false, err }
	hashes := strings.Split(item.RecoveryHashes, ",")
	remaining := make([]string, 0, len(hashes))
	found := false
	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" { continue }
		if !found && hash == target {
			found = true
			continue
		}
		remaining = append(remaining, hash)
	}
	if !found { return false, nil }
	item.RecoveryHashes = strings.Join(remaining, ",")
	if err := d.SaveUserMFA(item); err != nil { return false, err }
	return true, nil
}
