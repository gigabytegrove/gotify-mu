package security

import (
	"testing"
	"time"
)

func TestFixedWindowLimiterResets(t *testing.T) {
	base := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	limiter := NewFixedWindowLimiter(2, time.Minute)
	limiter.now = func() time.Time { return base }

	if ok, _ := limiter.Allow("ip"); !ok { t.Fatal("first request should pass") }
	if ok, _ := limiter.Allow("ip"); !ok { t.Fatal("second request should pass") }
	if ok, retry := limiter.Allow("ip"); ok || retry <= 0 { t.Fatal("third request should be limited") }

	base = base.Add(time.Minute)
	if ok, _ := limiter.Allow("ip"); !ok { t.Fatal("new window should reset limit") }
}

func TestDynamicLimiterPerKeyAndLimit(t *testing.T) {
	base := time.Now()
	limiter := NewDynamicLimiter()
	limiter.now = func() time.Time { return base }
	if ok, _ := limiter.Allow("a", 1, time.Minute); !ok { t.Fatal("first request should pass") }
	if ok, _ := limiter.Allow("a", 1, time.Minute); ok { t.Fatal("second request should be limited") }
	if ok, _ := limiter.Allow("b", 1, time.Minute); !ok { t.Fatal("different key should pass") }
	if ok, _ := limiter.Allow("unlimited", 0, time.Minute); !ok { t.Fatal("zero limit should be unlimited") }
}
