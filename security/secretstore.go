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
)

const encryptedPrefix = "enc:v1:"

type SecretStore struct {
	aead cipher.AEAD
}

func NewSecretStore(key []byte) (*SecretStore, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secret key must be exactly 32 bytes")
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

func LoadOrCreateSecretStore(path string) (*SecretStore, error) {
	if raw := strings.TrimSpace(os.Getenv("GOTIFY_MU_SECRET_KEY")); raw != "" {
		key, err := decodeSecretKey(raw)
		if err != nil {
			return nil, fmt.Errorf("GOTIFY_MU_SECRET_KEY: %w", err)
		}
		return NewSecretStore(key)
	}
	if override := strings.TrimSpace(os.Getenv("GOTIFY_MU_SECRET_KEY_FILE")); override != "" {
		path = override
	}
	if path == "" {
		path = filepath.Join("data", ".gotify-mu-secrets.key")
	}
	if content, err := os.ReadFile(path); err == nil {
		key, err := decodeSecretKey(strings.TrimSpace(string(content)))
		if err != nil {
			return nil, fmt.Errorf("read secret key %s: %w", path, err)
		}
		return NewSecretStore(key)
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
	encoded := base64.RawStdEncoding.EncodeToString(key)
	if err := os.WriteFile(path, []byte(encoded+"\n"), 0o600); err != nil {
		return nil, err
	}
	return NewSecretStore(key)
}

func NewTestSecretStore() *SecretStore {
	sum := sha256.Sum256([]byte("gotify-mu-test-secret-key"))
	store, _ := NewSecretStore(sum[:])
	return store
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
		return "", err
	}
	nonceSize := s.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", errors.New("encrypted secret payload is truncated")
	}
	plain, err := s.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func HashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func decodeSecretKey(raw string) ([]byte, error) {
	if decoded, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	return nil, errors.New("must be 32 bytes encoded as base64 or 64 hex characters")
}
