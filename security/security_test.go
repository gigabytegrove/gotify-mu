package security

import (
	"strings"
	"testing"
	"time"
)

func TestSecretBoxRoundTripAndLegacyRead(t *testing.T) {
	box, err := NewSecretBox(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := box.EncryptString("super-secret")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "super-secret" || !IsEncryptedSecret(encrypted) {
		t.Fatalf("expected encrypted value, got %q", encrypted)
	}
	decrypted, err := box.DecryptString(encrypted)
	if err != nil || decrypted != "super-secret" {
		t.Fatalf("unexpected decrypt result %q %v", decrypted, err)
	}
	legacy, err := box.DecryptString("legacy-plaintext")
	if err != nil || legacy != "legacy-plaintext" {
		t.Fatalf("legacy plaintext read failed: %q %v", legacy, err)
	}
}

func TestSecretBoxRejectsWrongKey(t *testing.T) {
	first, _ := NewSecretBox(strings.Repeat("11", 32))
	second, _ := NewSecretBox(strings.Repeat("22", 32))
	encrypted, _ := first.EncryptString("secret")
	if _, err := second.DecryptString(encrypted); err == nil {
		t.Fatal("expected wrong-key decryption failure")
	}
}

func TestFailureLimiterBlocksAndResets(t *testing.T) {
	limiter := NewFailureLimiter(2, time.Minute, time.Minute)
	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	if ok, _ := limiter.Allow("user"); !ok {
		t.Fatal("initial attempt should be allowed")
	}
	limiter.Failure("user")
	limiter.Failure("user")
	if ok, _ := limiter.Allow("user"); ok {
		t.Fatal("user should be blocked")
	}
	limiter.Success("user")
	if ok, _ := limiter.Allow("user"); !ok {
		t.Fatal("success should clear limiter")
	}
}

func TestWindowLimiter(t *testing.T) {
	limiter := NewWindowLimiter(2, time.Minute)
	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	if ok, _ := limiter.Allow("hook"); !ok { t.Fatal("first request denied") }
	if ok, _ := limiter.Allow("hook"); !ok { t.Fatal("second request denied") }
	if ok, _ := limiter.Allow("hook"); ok { t.Fatal("third request should be denied") }
	now = now.Add(time.Minute)
	if ok, _ := limiter.Allow("hook"); !ok { t.Fatal("new window should allow request") }
}
