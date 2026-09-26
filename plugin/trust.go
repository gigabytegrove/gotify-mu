package plugin

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type InstallVerification struct {
	ExpectedSHA256 string
	Signature      string
	PublicKey      string
}

type VerifiedPluginFile struct {
	Path   string
	SHA256 string
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	value = strings.TrimSpace(value)
	if value == "" { return nil, errors.New("plugin signing public key is required") }
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		raw, err = hex.DecodeString(value)
	}
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key")
	}
	return ed25519.PublicKey(raw), nil
}

func decodeSignature(value string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil { return nil, errors.New("invalid plugin signature") }
	if len(raw) != ed25519.SignatureSize { return nil, errors.New("invalid plugin signature length") }
	return raw, nil
}

func trustedPluginKeys() map[string]ed25519.PublicKey {
	result := map[string]ed25519.PublicKey{}
	for _, raw := range strings.FieldsFunc(os.Getenv("GOTIFY_MU_PLUGIN_TRUSTED_ED25519_KEYS"), func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == ' '
	}) {
		key, err := decodePublicKey(raw)
		if err != nil { continue }
		result[base64.StdEncoding.EncodeToString(key)] = key
	}
	return result
}

func allowUnsignedPluginInstalls() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GOTIFY_MU_PLUGIN_ALLOW_UNSIGNED_INSTALLS"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func verifyPluginStream(directory, filename string, source io.Reader, verification InstallVerification) (*VerifiedPluginFile, error) {
	if strings.ToLower(filepath.Ext(filename)) != ".so" {
		return nil, errors.New("plugin file must use the .so extension")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil { return nil, err }
	tmp, err := os.CreateTemp(directory, ".gotify-mu-verify-*.so")
	if err != nil { return nil, err }
	path := tmp.Name()
	ok := false
	defer func(){ if !ok { _ = os.Remove(path) } }()

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(source, MaxPluginUploadBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil { return nil, copyErr }
	if closeErr != nil { return nil, closeErr }
	if written > MaxPluginUploadBytes { return nil, fmt.Errorf("plugin exceeds the %d MiB upload limit", MaxPluginUploadBytes>>20) }
	digest := hasher.Sum(nil)
	digestHex := hex.EncodeToString(digest)

	if expected := strings.ToLower(strings.TrimSpace(verification.ExpectedSHA256)); expected != "" && expected != digestHex {
		return nil, fmt.Errorf("plugin checksum mismatch: expected %s, got %s", expected, digestHex)
	}

	signatureSupplied := strings.TrimSpace(verification.Signature) != "" || strings.TrimSpace(verification.PublicKey) != ""
	if !signatureSupplied {
		if !allowUnsignedPluginInstalls() {
			return nil, errors.New("unsigned plugin installation is disabled; provide a trusted Ed25519 signature or explicitly enable unsigned installs")
		}
		ok = true
		return &VerifiedPluginFile{Path:path,SHA256:digestHex}, nil
	}

	publicKey, err := decodePublicKey(verification.PublicKey)
	if err != nil { return nil, err }
	trusted := trustedPluginKeys()
	keyID := base64.StdEncoding.EncodeToString(publicKey)
	if _, exists := trusted[keyID]; !exists {
		return nil, errors.New("plugin signing key is not trusted by this server")
	}
	signature, err := decodeSignature(verification.Signature)
	if err != nil { return nil, err }
	if !ed25519.Verify(publicKey, digest, signature) {
		return nil, errors.New("plugin signature verification failed")
	}

	ok = true
	return &VerifiedPluginFile{Path:path,SHA256:digestHex}, nil
}
