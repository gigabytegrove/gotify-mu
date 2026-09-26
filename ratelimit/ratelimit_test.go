package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	limiter := New(2, time.Minute)
	limiter.now = func() time.Time { return now }

	if ok, _ := limiter.Allow("a"); !ok { t.Fatal("first request should pass") }
	if ok, _ := limiter.Allow("a"); !ok { t.Fatal("second request should pass") }
	if ok, _ := limiter.Allow("a"); ok { t.Fatal("third request should be limited") }
	if ok, _ := limiter.Allow("b"); !ok { t.Fatal("different key should pass") }

	now = now.Add(time.Minute)
	if ok, _ := limiter.Allow("a"); !ok { t.Fatal("new window should pass") }
}
