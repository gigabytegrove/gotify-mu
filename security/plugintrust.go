package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// VerifyPluginDigest verifies an Ed25519 signature over the raw SHA-256 digest
// of a plugin binary against administrator-configured trusted public keys.
//
// GOTIFY_MU_PLUGIN_TRUST_KEYS accepts comma/semicolon/newline separated
// Ed25519 public keys encoded as base64 or hex.
func VerifyPluginDigest(digest []byte, signatureText string) error {
	signature, err := decodePluginTrustValue(signatureText, ed25519.SignatureSize)
	if err != nil {
		return fmt.Errorf("invalid plugin signature: %w", err)
	}

	rawKeys := strings.TrimSpace(os.Getenv("GOTIFY_MU_PLUGIN_TRUST_KEYS"))
	if rawKeys == "" {
		return errors.New("plugin signature verification is required but no trusted signing keys are configured")
	}

	for _, raw := range strings.FieldsFunc(rawKeys, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	}) {
		key, decodeErr := decodePluginTrustValue(raw, ed25519.PublicKeySize)
		if decodeErr != nil {
			continue
		}
		if ed25519.Verify(ed25519.PublicKey(key), digest, signature) {
			return nil
		}
	}
	return errors.New("plugin signature was not created by a trusted signing key")
}

func decodePluginTrustValue(raw string, expected int) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("value is empty")
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == expected {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil && len(decoded) == expected {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == expected {
		return decoded, nil
	}
	return nil, fmt.Errorf("expected %d-byte base64 or hex value", expected)
}
