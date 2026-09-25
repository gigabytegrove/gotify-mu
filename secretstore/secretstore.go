package secretstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const prefix = "enc:v1:"

type Store struct {
	aead cipher.AEAD
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("secret key path is required")
	}
	key, err := loadOrCreateKey(path)
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
	return &Store{aead: aead}, nil
}

func (s *Store) Encrypt(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, prefix) {
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
		return value, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", fmt.Errorf("decode encrypted secret: %w", err)
	}
	if len(payload) < s.aead.NonceSize() {
		return "", errors.New("encrypted secret is truncated")
	}
	nonce := payload[:s.aead.NonceSize()]
	ciphertext := payload[s.aead.NonceSize():]
	plain, err := s.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("encrypted secret could not be decrypted")
	}
	return string(plain), nil
}

func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefix)
}

func loadOrCreateKey(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		decoded, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(data)))
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
	encoded := base64.RawStdEncoding.EncodeToString(key)
	if err := os.WriteFile(path, []byte(encoded+"\n"), 0o600); err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}
