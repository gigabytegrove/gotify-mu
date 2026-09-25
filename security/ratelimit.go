package security

import (
	"sync"
	"time"
)

type limiterEntry struct {
	tokens float64
	last   time.Time
}

// Limiter is a small in-process token bucket used to protect authentication and public ingress.
// It is intentionally dependency-free. Multi-instance edge throttling should still be enforced
// at the reverse proxy, but every Gotify MU instance protects itself independently.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]limiterEntry
	rate    float64
	burst   float64
	now     func() time.Time
}

func NewLimiter(perMinute int, burst int) *Limiter {
	if perMinute < 1 { perMinute = 1 }
	if burst < 1 { burst = 1 }
	return &Limiter{
		entries: make(map[string]limiterEntry),
		rate: float64(perMinute) / 60.0,
		burst: float64(burst),
		now: time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	if key == "" { key = "unknown" }
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		entry = limiterEntry{tokens: l.burst, last: now}
	}
	elapsed := now.Sub(entry.last).Seconds()
	if elapsed > 0 {
		entry.tokens += elapsed * l.rate
		if entry.tokens > l.burst { entry.tokens = l.burst }
	}
	entry.last = now
	if entry.tokens < 1 {
		l.entries[key] = entry
		return false
	}
	entry.tokens--
	l.entries[key] = entry
	return true
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

func (l *Limiter) Cleanup(maxIdle time.Duration) {
	cutoff := l.now().Add(-maxIdle)
	l.mu.Lock()
	for key, entry := range l.entries {
		if entry.last.Before(cutoff) {
			delete(l.entries, key)
		}
	}
	l.mu.Unlock()
}
