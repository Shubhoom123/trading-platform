package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is a per-client token-bucket limiter. Buckets are keyed by client
// IP and evicted after an idle period so the map can't grow without bound.
type RateLimiter struct {
	rps   rate.Limit
	burst int

	mu       sync.Mutex
	visitors map[string]*visitor
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rps:      rate.Limit(rps),
		burst:    burst,
		visitors: make(map[string]*visitor),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) limiterFor(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		lim := rate.NewLimiter(rl.rps, rl.burst)
		rl.visitors[key] = &visitor{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for key, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware rejects requests that exceed the per-client rate with 429.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.limiterFor(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
