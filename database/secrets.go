package database

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
	"gorm.io/gorm"
)

// ConfigureSecretBox enables encrypted integration credential persistence and
// upgrades existing plaintext integration credentials in place.
func (d *GormDatabase) ConfigureSecretBox(box *security.SecretBox) error {
	if box == nil {
		return errors.New("secret encryption is not configured")
	}
	d.SecretBox = box
	return d.encryptExistingIntegrationSecrets()
}

func (d *GormDatabase) encryptExistingIntegrationSecrets() error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		var webhooks []*model.WebhookRoute
		if err := tx.Find(&webhooks).Error; err != nil {
			return err
		}
		for _, item := range webhooks {
			plain, err := d.SecretBox.DecryptString(item.Secret)
			if err != nil {
				return fmt.Errorf("decrypt webhook %d secret: %w", item.ID, err)
			}
			encrypted := item.Secret
			if !security.IsEncryptedSecret(encrypted) {
				encrypted, err = d.SecretBox.EncryptString(plain)
				if err != nil {
					return err
				}
			}
			if err := tx.Model(&model.WebhookRoute{}).Where("id = ?", item.ID).
				Updates(map[string]any{
					"secret":      encrypted,
					"secret_hash": hashWebhookSecret(plain),
				}).Error; err != nil {
				return err
			}
		}

		var mqtt []*model.MQTTIntegration
		if err := tx.Find(&mqtt).Error; err != nil {
			return err
		}
		for _, item := range mqtt {
			if item.Password == "" {
				continue
			}
			plain, err := d.SecretBox.DecryptString(item.Password)
			if err != nil {
				return fmt.Errorf("decrypt MQTT integration %d password: %w", item.ID, err)
			}
			encrypted := item.Password
			if !security.IsEncryptedSecret(encrypted) {
				encrypted, err = d.SecretBox.EncryptString(plain)
				if err != nil {
					return err
				}
			}
			if err := tx.Model(&model.MQTTIntegration{}).Where("id = ?", item.ID).
				Update("password", encrypted).Error; err != nil {
				return err
			}
		}

		var homeAssistant []*model.HomeAssistantIntegration
		if err := tx.Find(&homeAssistant).Error; err != nil {
			return err
		}
		for _, item := range homeAssistant {
			if item.Token == "" {
				continue
			}
			plain, err := d.SecretBox.DecryptString(item.Token)
			if err != nil {
				return fmt.Errorf("decrypt Home Assistant integration %d token: %w", item.ID, err)
			}
			encrypted := item.Token
			if !security.IsEncryptedSecret(encrypted) {
				encrypted, err = d.SecretBox.EncryptString(plain)
				if err != nil {
					return err
				}
			}
			if err := tx.Model(&model.HomeAssistantIntegration{}).Where("id = ?", item.ID).
				Update("token", encrypted).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func hashWebhookSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func (d *GormDatabase) encryptCredential(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if d.SecretBox == nil {
		// Test/embedded database users may not configure application secret
		// encryption. The production server always configures it during startup.
		return value, nil
	}
	return d.SecretBox.EncryptString(value)
}

func (d *GormDatabase) decryptCredential(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if d.SecretBox == nil {
		if security.IsEncryptedSecret(value) {
			return "", errors.New("encrypted credential cannot be read without the configured secret key")
		}
		return value, nil
	}
	return d.SecretBox.DecryptString(value)
}
