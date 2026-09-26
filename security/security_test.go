package security

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSecretStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	store, err := NewSecretStore(path)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := store.Encrypt("super-secret")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "super-secret" || !store.IsEncrypted(encrypted) {
		t.Fatalf("secret was not encrypted: %q", encrypted)
	}
	decrypted, err := store.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "super-secret" {
		t.Fatalf("unexpected plaintext %q", decrypted)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected key file mode 0600, got %o", info.Mode().Perm())
	}
}

func TestSecretStoreReadsLegacyPlaintext(t *testing.T) {
	store, err := NewSecretStore(filepath.Join(t.TempDir(), "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := store.Decrypt("legacy-plaintext")
	if err != nil || value != "legacy-plaintext" {
		t.Fatalf("legacy plaintext failed: %q %v", value, err)
	}
}

func TestRateLimiterBlocksAndResets(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute, 30*time.Second)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	if ok, _ := limiter.Allow("key"); !ok { t.Fatal("first request should pass") }
	if ok, _ := limiter.Allow("key"); !ok { t.Fatal("second request should pass") }
	if ok, _ := limiter.Allow("key"); ok { t.Fatal("third request should be blocked") }

	now = now.Add(31 * time.Second)
	if ok, _ := limiter.Allow("key"); ok {
		t.Fatal("window count should still block until window resets")
	}
	now = now.Add(30 * time.Second)
	if ok, _ := limiter.Allow("key"); !ok {
		t.Fatal("new window should permit requests")
	}
}
