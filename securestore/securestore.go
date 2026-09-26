package securestore

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

const prefix = "enc:v1:"

// Store encrypts small application secrets using AES-256-GCM.
type Store struct {
	aead cipher.AEAD
}

// LoadOrCreate loads a 32-byte key from path or creates it with owner-only permissions.
func LoadOrCreate(path string) (*Store, error) {
	path = filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create secret-key directory: %w", err)
	}

	key, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("generate secret key: %w", err)
		}
		if err := os.WriteFile(path, key, 0o600); err != nil {
			return nil, fmt.Errorf("write secret key: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("read secret key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("secret key must be exactly 32 bytes")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("secure secret key permissions: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Store{aead: aead}, nil
}

func (s *Store) Encrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, prefix) {
		return value, nil
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := s.aead.Seal(nil, nonce, []byte(value), nil)
	payload := append(nonce, sealed...)
	return prefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (s *Store) Decrypt(value string) (string, error) {
	if value == "" || !strings.HasPrefix(value, prefix) {
		// Plaintext fallback exists only so existing installations can be migrated in-place.
		return value, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", fmt.Errorf("decode encrypted secret: %w", err)
	}
	nonceSize := s.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", errors.New("encrypted secret payload is invalid")
	}
	plain, err := s.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", errors.New("encrypted secret could not be decrypted")
	}
	return string(plain), nil
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefix)
}
