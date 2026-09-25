package mfa

import (
	"testing"
	"time"
)

func TestTOTPVerification(t *testing.T) {
	// RFC 6238 SHA1 test secret, base32 encoded.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Unix(59, 0)
	if got := totp(secret, now.Unix()/30); got != "287082" {
		t.Fatalf("expected RFC code 287082, got %s", got)
	}
	if !VerifyTOTP(secret, "287082", now) {
		t.Fatal("expected code to verify")
	}
	if VerifyTOTP(secret, "000000", now) {
		t.Fatal("unexpected invalid code verification")
	}
}

func TestRecoveryCodeConsumption(t *testing.T) {
	hashes := []string{HashRecoveryCode("ABCD-EFGH"), HashRecoveryCode("IJKL-MNOP")}
	remaining, ok := ConsumeRecoveryCode(hashes, "abcd-efgh")
	if !ok || len(remaining) != 1 {
		t.Fatalf("expected first recovery code to be consumed: %#v %v", remaining, ok)
	}
	_, ok = ConsumeRecoveryCode(remaining, "ABCD-EFGH")
	if ok {
		t.Fatal("recovery code must be one-time use")
	}
}

func TestProvisioningURI(t *testing.T) {
	uri := ProvisioningURI("Gotify MU", "admin", "ABC123")
	if uri == "" || uri[:15] != "otpauth://totp/" {
		t.Fatalf("unexpected provisioning URI %q", uri)
	}
}
