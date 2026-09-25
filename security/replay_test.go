package security

import (
	"testing"
	"time"
)

func TestReplayCacheRejectsLiveDuplicateAndExpires(t *testing.T) {
	now := time.Now()
	cache := NewReplayCache()
	cache.now = func() time.Time { return now }
	if !cache.Remember("request", time.Minute) { t.Fatal("first request should be accepted") }
	if cache.Remember("request", time.Minute) { t.Fatal("duplicate should be rejected") }
	now = now.Add(time.Minute + time.Second)
	if !cache.Remember("request", time.Minute) { t.Fatal("expired replay entry should be accepted again") }
}
