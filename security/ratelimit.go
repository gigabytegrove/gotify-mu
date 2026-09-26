package security

import (
	"sync"
	"time"
)

type failureEntry struct {
	windowStarted time.Time
	failures      int
	blockedUntil  time.Time
	lastSeen      time.Time
}

// FailureLimiter throttles repeated failed authentication attempts.
type FailureLimiter struct {
	mu          sync.Mutex
	entries     map[string]*failureEntry
	maxFailures int
	window      time.Duration
	block       time.Duration
	now         func() time.Time
}

func NewFailureLimiter(maxFailures int, window, block time.Duration) *FailureLimiter {
	if maxFailures < 1 {
		maxFailures = 1
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	if block <= 0 {
		block = 15 * time.Minute
	}
	return &FailureLimiter{
		entries:     make(map[string]*failureEntry),
		maxFailures: maxFailures,
		window:      window,
		block:       block,
		now:         time.Now,
	}
}

func (l *FailureLimiter) Allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)

	entry := l.entries[key]
	if entry == nil {
		return true, 0
	}
	entry.lastSeen = now
	if entry.blockedUntil.After(now) {
		return false, time.Until(entry.blockedUntil)
	}
	if now.Sub(entry.windowStarted) >= l.window {
		delete(l.entries, key)
	}
	return true, 0
}

func (l *FailureLimiter) Failure(key string) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)

	entry := l.entries[key]
	if entry == nil || now.Sub(entry.windowStarted) >= l.window {
		entry = &failureEntry{windowStarted: now}
		l.entries[key] = entry
	}
	entry.failures++
	entry.lastSeen = now
	if entry.failures >= l.maxFailures {
		entry.blockedUntil = now.Add(l.block)
	}
}

func (l *FailureLimiter) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *FailureLimiter) pruneLocked(now time.Time) {
	for key, entry := range l.entries {
		if entry.blockedUntil.After(now) {
			continue
		}
		if now.Sub(entry.lastSeen) > l.window+l.block {
			delete(l.entries, key)
		}
	}
}

type windowEntry struct {
	start    time.Time
	requests int
	lastSeen time.Time
}

// WindowLimiter provides an in-memory fixed-window request limit.
type WindowLimiter struct {
	mu      sync.Mutex
	entries map[string]*windowEntry
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewWindowLimiter(limit int, window time.Duration) *WindowLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &WindowLimiter{
		entries: make(map[string]*windowEntry),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

func (l *WindowLimiter) Allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	for existingKey, entry := range l.entries {
		if now.Sub(entry.lastSeen) > 2*l.window {
			delete(l.entries, existingKey)
		}
	}

	entry := l.entries[key]
	if entry == nil || now.Sub(entry.start) >= l.window {
		l.entries[key] = &windowEntry{start: now, requests: 1, lastSeen: now}
		return true, 0
	}
	entry.lastSeen = now
	if entry.requests >= l.limit {
		retry := l.window - now.Sub(entry.start)
		if retry < time.Second {
			retry = time.Second
		}
		return false, retry
	}
	entry.requests++
	return true, 0
}
