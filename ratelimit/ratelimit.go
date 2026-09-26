package ratelimit

import (
	"sync"
	"time"
)

type entry struct {
	count   int
	resetAt time.Time
}

// Limiter is a small in-memory fixed-window limiter for abuse-sensitive HTTP endpoints.
// It is intentionally process-local; distributed deployments should use an external
// edge limiter in addition to this protection.
type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	items  map[string]entry
	now    func() time.Time
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, items: map[string]entry{}, now: time.Now}
}

// Allow returns whether the key is allowed and the time the current window resets.
func (l *Limiter) Allow(key string) (bool, time.Time) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	item, ok := l.items[key]
	if !ok || !now.Before(item.resetAt) {
		item = entry{resetAt: now.Add(l.window)}
	}
	item.count++
	l.items[key] = item

	// Opportunistic cleanup keeps the map bounded without a background goroutine.
	if len(l.items) > 4096 {
		for candidate, value := range l.items {
			if !now.Before(value.resetAt) {
				delete(l.items, candidate)
			}
		}
	}
	return item.count <= l.limit, item.resetAt
}
