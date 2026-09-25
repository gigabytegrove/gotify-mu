package security

import (
	"testing"
	"time"
)

func TestLimiterBlocksAndRefills(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(60, 2)
	limiter.now = func() time.Time { return now }
	if !limiter.Allow("a") || !limiter.Allow("a") { t.Fatal("initial burst should pass") }
	if limiter.Allow("a") { t.Fatal("third request should be blocked") }
	now = now.Add(time.Second)
	if !limiter.Allow("a") { t.Fatal("one token should refill after one second") }
}

func TestLimiterReset(t *testing.T) {
	limiter := NewLimiter(1, 1)
	if !limiter.Allow("a") { t.Fatal("first request should pass") }
	if limiter.Allow("a") { t.Fatal("second request should block") }
	limiter.Reset("a")
	if !limiter.Allow("a") { t.Fatal("reset should restore burst") }
}
