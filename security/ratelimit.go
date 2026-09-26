package security

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type limitEntry struct {
	windowStart time.Time
	count       int
	blockedUntil time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]limitEntry
	limit    int
	window   time.Duration
	blockFor time.Duration
	now      func() time.Time
}

func NewRateLimiter(limit int, window, blockFor time.Duration) *RateLimiter {
	if limit < 1 {
		limit = 1
	}
	return &RateLimiter{
		entries:  make(map[string]limitEntry),
		limit:    limit,
		window:   window,
		blockFor: blockFor,
		now:      time.Now,
	}
}

func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entries[key]
	if entry.blockedUntil.After(now) {
		return false, time.Until(entry.blockedUntil)
	}
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= r.window {
		entry.windowStart = now
		entry.count = 0
		entry.blockedUntil = time.Time{}
	}
	entry.count++
	if entry.count > r.limit {
		entry.blockedUntil = now.Add(r.blockFor)
		r.entries[key] = entry
		return false, r.blockFor
	}
	r.entries[key] = entry

	if len(r.entries) > 10000 {
		for candidate, current := range r.entries {
			if current.blockedUntil.Before(now) && now.Sub(current.windowStart) > 2*r.window {
				delete(r.entries, candidate)
			}
		}
	}
	return true, 0
}

func (r *RateLimiter) Middleware(key func(*gin.Context) string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		allowed, retry := r.Allow(key(ctx))
		if !allowed {
			seconds := int(retry.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			ctx.Header("Retry-After", fmtInt(seconds))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests; try again later",
			})
			return
		}
		ctx.Next()
	}
}

func fmtInt(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[index:])
}
