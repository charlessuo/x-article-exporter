package api

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limits  map[string]int // per-key override; 0 means use defaultLimit
	defaultLimit int
}

type bucket struct {
	tokens   float64
	lastTime time.Time
}

func newRateLimiter(defaultLimit int) *rateLimiter {
	return &rateLimiter{
		buckets:      make(map[string]*bucket),
		limits:       make(map[string]int),
		defaultLimit: defaultLimit,
	}
}

// setKeyLimit sets a per-key rate limit override.
func (rl *rateLimiter) setKeyLimit(key string, limit int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limits[key] = limit
}

// allow checks whether the given key can make a request.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limit := rl.defaultLimit
	if l, ok := rl.limits[key]; ok && l > 0 {
		limit = l
	}

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(limit), lastTime: now}
		rl.buckets[key] = b
	}

	elapsed := now.Sub(b.lastTime).Seconds()
	b.lastTime = now

	// Refill tokens based on elapsed time.
	refillRate := float64(limit) / 3600.0
	b.tokens += elapsed * refillRate
	if b.tokens > float64(limit) {
		b.tokens = float64(limit)
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
