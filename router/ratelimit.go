package router

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type limiterBucket struct {
	Count   int
	ResetAt time.Time
}

type requestLimiter struct {
	mutex  sync.Mutex
	limit  int
	window time.Duration
	now    func() time.Time
	items  map[string]limiterBucket
}

func newRequestLimiter(limit int, window time.Duration) *requestLimiter {
	return &requestLimiter{
		limit:  limit,
		window: window,
		now:    time.Now,
		items:  make(map[string]limiterBucket),
	}
}

func (l *requestLimiter) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.ClientIP()
		now := l.now()

		l.mutex.Lock()
		bucket := l.items[key]
		if bucket.ResetAt.IsZero() || !now.Before(bucket.ResetAt) {
			bucket = limiterBucket{ResetAt: now.Add(l.window)}
		}
		if bucket.Count >= l.limit {
			retryAfter := int(time.Until(bucket.ResetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			l.items[key] = bucket
			l.mutex.Unlock()

			ctx.Header("Retry-After", http.StatusText(retryAfter))
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Try again later.",
			})
			return
		}
		bucket.Count++
		l.items[key] = bucket

		if len(l.items) > 10000 {
			for existingKey, existing := range l.items {
				if !now.Before(existing.ResetAt) {
					delete(l.items, existingKey)
				}
			}
		}
		l.mutex.Unlock()

		ctx.Next()
	}
}
