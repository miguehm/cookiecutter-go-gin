package middlewares

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rate     rate.Limit
	burst    int
	enabled  bool
	now      func() time.Time
}

func NewRateLimiter(rps float64, burst int, enabled bool) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		enabled:  enabled,
		now:      time.Now,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.limiters {
			if rl.now().Sub(v.lastSeen) > 3*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: rl.now()}
		return limiter
	}
	v.lastSeen = rl.now()
	return v.limiter
}

func (rl *RateLimiter) Middleware(skipPaths ...string) gin.HandlerFunc {
	skip := map[string]struct{}{}
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}
	return func(c *gin.Context) {
		if !rl.enabled {
			c.Next()
			return
		}
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			c.Header("Retry-After", strconv.Itoa(int(time.Second.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": "ErrRateLimited", "message": "too many requests"})
			return
		}
		c.Next()
	}
}
