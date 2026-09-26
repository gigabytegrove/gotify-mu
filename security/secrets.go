package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const encryptedPrefix = "enc:v1:"

type SecretStore struct {
	aead cipher.AEAD
}

func NewSecretStore(path string) (*SecretStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("secret key file path is required")
	}
	key, err := loadOrCreateSecretKey(path)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretStore{aead: aead}, nil
}

func loadOrCreateSecretKey(path string) ([]byte, error) {
	if raw, err := os.ReadFile(path); err == nil {
		decoded, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if decodeErr != nil || len(decoded) != 32 {
			return nil, errors.New("secret key file is invalid")
		}
		return decoded, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".secret-key-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return nil, err
	}
	if _, err := tmp.WriteString(base64.RawStdEncoding.EncodeToString(key) + "\n"); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *SecretStore) Encrypt(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, encryptedPrefix) {
		return value, nil
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := s.aead.Seal(nil, nonce, []byte(value), nil)
	payload := append(nonce, sealed...)
	return encryptedPrefix + base64.RawStdEncoding.EncodeToString(payload), nil
}

func (s *SecretStore) Decrypt(value string) (string, error) {
	if value == "" || !strings.HasPrefix(value, encryptedPrefix) {
		return value, nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, encryptedPrefix))
	if err != nil {
		return "", fmt.Errorf("decode encrypted secret: %w", err)
	}
	nonceSize := s.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", errors.New("encrypted secret is truncated")
	}
	plain, err := s.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", errors.New("encrypted secret could not be decrypted")
	}
	return string(plain), nil
}

func (s *SecretStore) IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}
