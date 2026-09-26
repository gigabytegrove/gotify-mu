package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const encryptedPrefix = "enc:v1:"

// SecretBox encrypts application-held credentials before database persistence.
type SecretBox struct {
	aead cipher.AEAD
}

func NewSecretBox(encodedKey string) (*SecretBox, error) {
	encodedKey = strings.TrimSpace(encodedKey)
	if encodedKey == "" {
		return nil, errors.New("secret encryption key is required")
	}

	key, err := decodeSecretKey(encodedKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize secret cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize secret encryption: %w", err)
	}
	return &SecretBox{aead: aead}, nil
}

func decodeSecretKey(raw string) ([]byte, error) {
	if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	return nil, errors.New("secret encryption key must be 32 bytes encoded as 64 hex characters or base64")
}

func (b *SecretBox) EncryptString(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, encryptedPrefix) {
		return value, nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate secret nonce: %w", err)
	}
	sealed := b.aead.Seal(nil, nonce, []byte(value), nil)
	payload := append(nonce, sealed...)
	return encryptedPrefix + base64.RawStdEncoding.EncodeToString(payload), nil
}

func (b *SecretBox) DecryptString(value string) (string, error) {
	if value == "" || !strings.HasPrefix(value, encryptedPrefix) {
		return value, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, encryptedPrefix))
	if err != nil {
		return "", errors.New("stored credential is not valid encrypted data")
	}
	nonceSize := b.aead.NonceSize()
	if len(raw) <= nonceSize {
		return "", errors.New("stored credential is truncated")
	}
	plain, err := b.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", errors.New("stored credential could not be decrypted with the configured key")
	}
	return string(plain), nil
}

func IsEncryptedSecret(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}
