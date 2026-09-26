package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil { return "", err }
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

func TOTPCode(secret string, now time.Time) (string, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	padding := len(secret) % 8
	if padding != 0 { secret += strings.Repeat("=", 8-padding) }
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil { return "", err }
	counter := uint64(now.Unix() / 30)
	var msg [8]byte
	for i:=7;i>=0;i-- { msg[i]=byte(counter); counter >>= 8 }
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000), nil
}

func VerifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 { return false }
	if _, err := strconv.Atoi(code); err != nil { return false }
	for _, step := range []int{-1,0,1} {
		expected, err := TOTPCode(secret, now.Add(time.Duration(step)*30*time.Second))
		if err == nil && hmac.Equal([]byte(expected), []byte(code)) { return true }
	}
	return false
}

func GenerateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 { count = 10 }
	result := make([]string, 0, count)
	for i:=0;i<count;i++ {
		raw:=make([]byte,6)
		if _,err:=rand.Read(raw);err!=nil{return nil,err}
		encoded:=strings.ToUpper(hex.EncodeToString(raw))
		result=append(result,encoded[:6]+"-"+encoded[6:])
	}
	return result,nil
}

func HashRecoveryCode(code string) string {
	sum:=sha256.Sum256([]byte(strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code),"-",""))))
	return hex.EncodeToString(sum[:])
}
