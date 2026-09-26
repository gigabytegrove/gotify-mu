package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) GetPasskeysByUser(userID uint) ([]*model.PasskeyCredential, error) {
	var items []*model.PasskeyCredential
	return items, d.DB.Where("user_id = ?", userID).Order("created_at asc, id asc").Find(&items).Error
}

func (d *GormDatabase) GetPasskeyByID(id uint) (*model.PasskeyCredential, error) {
	item := new(model.PasskeyCredential)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) GetPasskeyByCredentialID(credentialID string) (*model.PasskeyCredential, error) {
	item := new(model.PasskeyCredential)
	if err := d.DB.First(item, "credential_id = ?", credentialID).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) SavePasskey(item *model.PasskeyCredential) error { return d.DB.Save(item).Error }

func (d *GormDatabase) DeletePasskey(userID, id uint) error {
	return d.DB.Where("user_id = ? AND id = ?", userID, id).Delete(&model.PasskeyCredential{}).Error
}

func (d *GormDatabase) SaveWebAuthnChallenge(item *model.WebAuthnChallenge) error {
	return d.DB.Create(item).Error
}

func (d *GormDatabase) GetWebAuthnChallenge(challenge string, now time.Time) (*model.WebAuthnChallenge, error) {
	item := new(model.WebAuthnChallenge)
	if err := d.DB.First(item, "challenge = ? AND expires_at > ?", challenge, now).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}

func (d *GormDatabase) DeleteWebAuthnChallenge(challenge string) error {
	return d.DB.Where("challenge = ?", challenge).Delete(&model.WebAuthnChallenge{}).Error
}

func (d *GormDatabase) CleanupWebAuthnChallenges(now time.Time) error {
	return d.DB.Where("expires_at <= ?", now).Delete(&model.WebAuthnChallenge{}).Error
}
