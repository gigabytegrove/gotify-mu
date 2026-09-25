package security

import (
	"testing"
	"time"
)

func TestTOTPAgainstRFC6238SHA1Vector(t *testing.T) {
	// RFC 6238 SHA-1 seed, encoded as base32. The implementation uses six digits.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := TOTPCode(secret, time.Unix(59, 0))
	if err != nil { t.Fatal(err) }
	if code != "287082" {
		t.Fatalf("expected 287082, got %s", code)
	}
	if !VerifyTOTP(secret, code, time.Unix(59, 0)) {
		t.Fatal("generated code should verify")
	}
	if VerifyTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("incorrect code should not verify")
	}
}

func TestRecoveryCodeNormalization(t *testing.T) {
	a := HashRecoveryCode("ABCDEF-123456")
	b := HashRecoveryCode("abcdef123456")
	if a != b { t.Fatal("recovery-code hash should ignore case and dash formatting") }
}
