package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func GenerateSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil { return "", err }
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

func VerifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 { return false }
	for offset := int64(-1); offset <= 1; offset++ {
		if totp(secret, now.Unix()/30+offset) == code { return true }
	}
	return false
}

func totp(secret string, counter int64) string {
	padding := (8 - len(secret)%8) % 8
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret) + strings.Repeat("=", padding))
	if err != nil { return "" }
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(counter)
		counter >>= 8
	}
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binary := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", binary%1000000)
}

func ProvisioningURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", issuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", "6")
	values.Set("period", "30")
	return "otpauth://totp/" + label + "?" + values.Encode()
}

func GenerateRecoveryCodes(count int) ([]string, []string, error) {
	codes := make([]string, 0, count)
	hashes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		raw := make([]byte, 8)
		if _, err := rand.Read(raw); err != nil { return nil, nil, err }
		code := strings.ToUpper(hex.EncodeToString(raw[:4]) + "-" + hex.EncodeToString(raw[4:]))
		codes = append(codes, code)
		hashes = append(hashes, HashRecoveryCode(code))
	}
	return codes, hashes, nil
}

func HashRecoveryCode(code string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func ConsumeRecoveryCode(hashes []string, code string) ([]string, bool) {
	target := HashRecoveryCode(code)
	for i, hash := range hashes {
		if hmac.Equal([]byte(hash), []byte(target)) {
			return append(hashes[:i], hashes[i+1:]...), true
		}
	}
	return hashes, false
}

func ParseRecoveryHashes(raw string) []string {
	if strings.TrimSpace(raw) == "" { return nil }
	values := strings.Split(raw, ",")
	out := values[:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" { out = append(out, strings.TrimSpace(value)) }
	}
	return out
}

func SerializeRecoveryHashes(values []string) string { return strings.Join(values, ",") }

func ParseCode(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if _, err := strconv.Atoi(value); err != nil || len(value) != 6 { return "", false }
	return value, true
}
