package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const encryptedPrefix = "enc:v1:"
const webhookHashPrefix = "sha256:"

var (
	keyOnce sync.Once
	keyData []byte
	keyErr error
)

func keyPath() string {
	if value := strings.TrimSpace(os.Getenv("GOTIFY_MU_SECRET_KEY_FILE")); value != "" {
		return value
	}
	return "data/secret.key"
}

func loadKey() ([]byte, error) {
	keyOnce.Do(func() {
		path := keyPath()
		if raw, err := os.ReadFile(path); err == nil {
			decoded, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
			if decodeErr != nil || len(decoded) != 32 {
				keyErr = errors.New("integration secret key is invalid")
				return
			}
			keyData = decoded
			return
		} else if !os.IsNotExist(err) {
			keyErr = fmt.Errorf("read integration secret key: %w", err)
			return
		}

		raw := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, raw); err != nil {
			keyErr = fmt.Errorf("generate integration secret key: %w", err)
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			keyErr = fmt.Errorf("create integration secret key directory: %w", err)
			return
		}
		encoded := []byte(base64.RawStdEncoding.EncodeToString(raw) + "\n")
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			keyErr = fmt.Errorf("write integration secret key: %w", err)
			return
		}
		keyData = raw
	})
	if keyErr != nil {
		return nil, keyErr
	}
	return append([]byte(nil), keyData...), nil
}

// Protect encrypts a reusable secret for storage. Existing encrypted values are returned unchanged.
func Protect(plaintext string) (string, error) {
	if plaintext == "" || strings.HasPrefix(plaintext, encryptedPrefix) {
		return plaintext, nil
	}
	key, err := loadKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, sealed...)
	return encryptedPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

// Reveal decrypts an encrypted secret. Plaintext values are accepted for migration compatibility.
func Reveal(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, encryptedPrefix) {
		return stored, nil
	}
	key, err := loadKey()
	if err != nil {
		return "", err
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(stored, encryptedPrefix))
	if err != nil {
		return "", errors.New("stored integration secret is invalid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("stored integration secret is invalid")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("stored integration secret could not be decrypted")
	}
	return string(plaintext), nil
}

// WebhookVerifier returns a non-reversible verifier suitable for storing webhook credentials.
func WebhookVerifier(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return webhookHashPrefix + hex.EncodeToString(sum[:])
}

func IsWebhookVerifier(value string) bool {
	return strings.HasPrefix(value, webhookHashPrefix)
}

func StableHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}
