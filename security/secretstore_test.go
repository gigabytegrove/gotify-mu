package security

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretStoreRoundTripAndTamperDetection(t *testing.T) {
	key := make([]byte, 32)
	for i := range key { key[i] = byte(i + 1) }
	store, err := NewSecretStore(key)
	if err != nil { t.Fatal(err) }

	encrypted, err := store.Encrypt("super-secret")
	if err != nil { t.Fatal(err) }
	if encrypted == "super-secret" || !strings.HasPrefix(encrypted, encryptedPrefix) {
		t.Fatalf("secret was not encrypted: %q", encrypted)
	}
	plain, err := store.Decrypt(encrypted)
	if err != nil { t.Fatal(err) }
	if plain != "super-secret" { t.Fatalf("unexpected plaintext %q", plain) }

	tampered := encrypted[:len(encrypted)-1] + "A"
	if _, err := store.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext should fail authentication")
	}
}

func TestLoadOrCreateSecretStoreUsesPrivateKeyFile(t *testing.T) {
	t.Setenv("GOTIFY_MU_SECRET_KEY", "")
	t.Setenv("GOTIFY_MU_SECRET_KEY_FILE", "")
	path := filepath.Join(t.TempDir(), "secret.key")
	store, err := LoadOrCreateSecretStore(path)
	if err != nil { t.Fatal(err) }
	if store == nil { t.Fatal("expected secret store") }
	info, err := os.Stat(path)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 key permissions, got %o", info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(decoded) != 32 {
		t.Fatalf("invalid generated key: len=%d err=%v", len(decoded), err)
	}
}

func TestHashSecretStable(t *testing.T) {
	if HashSecret("one") != HashSecret("one") { t.Fatal("hash must be stable") }
	if HashSecret("one") == HashSecret("two") { t.Fatal("different secrets must hash differently") }
}
