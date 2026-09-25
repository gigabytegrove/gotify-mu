package secretstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreEncryptDecryptAndPersistKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "secret.key")
	store, err := Open(keyPath)
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

	info, err := os.Stat(keyPath)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected key mode 0600, got %o", info.Mode().Perm())
	}

	reopened, err := Open(keyPath)
	if err != nil { t.Fatal(err) }
	plain, err = reopened.Decrypt(encrypted)
	if err != nil { t.Fatal(err) }
	if plain != "super-secret" {
		t.Fatalf("reopened store could not decrypt value")
	}
}

func TestHashIsStableAndNonPlaintext(t *testing.T) {
	a := Hash("value")
	b := Hash("value")
	if a != b || a == "value" || len(a) != 64 {
		t.Fatalf("unexpected hash values: %q %q", a, b)
	}
}
