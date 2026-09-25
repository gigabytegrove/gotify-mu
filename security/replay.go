package security

import (
	"sync"
	"time"
)

type ReplayCache struct {
	mu      sync.Mutex
	entries map[string]time.Time
	now     func() time.Time
}

func NewReplayCache() *ReplayCache {
	return &ReplayCache{entries: make(map[string]time.Time), now: time.Now}
}

// Remember returns false when the key has already been seen and is still live.
func (c *ReplayCache) Remember(key string, ttl time.Duration) bool {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if expires, ok := c.entries[key]; ok && expires.After(now) {
		return false
	}
	c.entries[key] = now.Add(ttl)
	if len(c.entries) > 4096 {
		for k, expires := range c.entries {
			if !expires.After(now) { delete(c.entries, k) }
		}
	}
	return true
}
