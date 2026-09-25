package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil { return "", err }
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func TOTPCode(secret string, at time.Time) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil { return "", err }
	counter := uint64(at.Unix() / 30)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000), nil
}

func VerifyTOTP(secret, code string, at time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	if _, err := strconv.Atoi(code); err != nil { return false }
	for offset := -1; offset <= 1; offset++ {
		expected, err := TOTPCode(secret, at.Add(time.Duration(offset)*30*time.Second))
		if err == nil && hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

func GenerateRecoveryCodes(count int) ([]string, []string, error) {
	if count < 1 { count = 10 }
	plain := make([]string, 0, count)
	hashes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		raw := make([]byte, 6)
		if _, err := rand.Read(raw); err != nil { return nil, nil, err }
		code := strings.ToUpper(hex.EncodeToString(raw))
		code = code[:4] + "-" + code[4:8] + "-" + code[8:]
		plain = append(plain, code)
		hashes = append(hashes, RecoveryCodeHash(code))
	}
	return plain, hashes, nil
}

func RecoveryCodeHash(code string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
