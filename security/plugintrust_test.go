package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestVerifyPluginDigest(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil { t.Fatal(err) }
	digest := sha256.Sum256([]byte("plugin binary"))
	signature := ed25519.Sign(privateKey, digest[:])
	t.Setenv("GOTIFY_MU_PLUGIN_TRUST_KEYS", base64.StdEncoding.EncodeToString(publicKey))
	if err := VerifyPluginDigest(digest[:], base64.StdEncoding.EncodeToString(signature)); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyPluginDigestRejectsUnknownSigner(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil { t.Fatal(err) }
	_, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil { t.Fatal(err) }
	digest := sha256.Sum256([]byte("plugin binary"))
	signature := ed25519.Sign(otherPrivate, digest[:])
	t.Setenv("GOTIFY_MU_PLUGIN_TRUST_KEYS", base64.StdEncoding.EncodeToString(publicKey))
	if err := VerifyPluginDigest(digest[:], base64.StdEncoding.EncodeToString(signature)); err == nil {
		t.Fatal("expected signature from unknown signer to fail")
	}
}
