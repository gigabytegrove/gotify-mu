package securestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	store, err := LoadOrCreate(filepath.Join(t.TempDir(), "secrets.key"))
	if err != nil { t.Fatal(err) }
	encrypted, err := store.Encrypt("super-secret")
	if err != nil { t.Fatal(err) }
	if encrypted == "super-secret" || !IsEncrypted(encrypted) {
		t.Fatalf("secret was not encrypted: %q", encrypted)
	}
	plain, err := store.Decrypt(encrypted)
	if err != nil { t.Fatal(err) }
	if plain != "super-secret" {
		t.Fatalf("expected original secret, got %q", plain)
	}
}

func TestKeyPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.key")
	if _, err := LoadOrCreate(path); err != nil { t.Fatal(err) }
	info, err := os.Stat(path)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestPlaintextMigrationRead(t *testing.T) {
	store, err := LoadOrCreate(filepath.Join(t.TempDir(), "secrets.key"))
	if err != nil { t.Fatal(err) }
	value, err := store.Decrypt("legacy-plaintext")
	if err != nil { t.Fatal(err) }
	if value != "legacy-plaintext" { t.Fatalf("unexpected value %q", value) }
}
