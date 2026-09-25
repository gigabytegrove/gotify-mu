package security

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type limiterEntry struct {
	WindowStart time.Time
	Count       int
	LastSeen    time.Time
}

type FixedWindowLimiter struct {
	mu      sync.Mutex
	entries map[string]limiterEntry
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		entries: make(map[string]limiterEntry),
		limit: limit,
		window: window,
		now: time.Now,
	}
}

func (l *FixedWindowLimiter) Allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[key]
	if entry.WindowStart.IsZero() || now.Sub(entry.WindowStart) >= l.window {
		entry.WindowStart = now
		entry.Count = 0
	}
	entry.Count++
	entry.LastSeen = now
	l.entries[key] = entry

	if len(l.entries) > 4096 {
		cutoff := now.Add(-2 * l.window)
		for k, candidate := range l.entries {
			if candidate.LastSeen.Before(cutoff) {
				delete(l.entries, k)
			}
		}
	}

	if entry.Count <= l.limit {
		return true, 0
	}
	retry := l.window - now.Sub(entry.WindowStart)
	if retry < time.Second {
		retry = time.Second
	}
	return false, retry
}

func RateLimitMiddleware(limiter *FixedWindowLimiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		allowed, retry := limiter.Allow(key(ctx))
		if allowed {
			ctx.Next()
			return
		}
		seconds := int(retry.Round(time.Second).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		ctx.Header("Retry-After", strconv.Itoa(seconds))
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": fmt.Sprintf("too many requests; retry in %d seconds", seconds),
		})
	}
}


type DynamicLimiter struct {
	mu      sync.Mutex
	entries map[string]limiterEntry
	now     func() time.Time
}

func NewDynamicLimiter() *DynamicLimiter {
	return &DynamicLimiter{entries: make(map[string]limiterEntry), now: time.Now}
}

func (l *DynamicLimiter) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if limit <= 0 {
		return true, 0
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entries[key]
	if entry.WindowStart.IsZero() || now.Sub(entry.WindowStart) >= window {
		entry.WindowStart = now
		entry.Count = 0
	}
	entry.Count++
	entry.LastSeen = now
	l.entries[key] = entry
	if len(l.entries) > 4096 {
		cutoff := now.Add(-2 * window)
		for k, candidate := range l.entries {
			if candidate.LastSeen.Before(cutoff) { delete(l.entries, k) }
		}
	}
	if entry.Count <= limit { return true, 0 }
	retry := window - now.Sub(entry.WindowStart)
	if retry < time.Second { retry = time.Second }
	return false, retry
}
