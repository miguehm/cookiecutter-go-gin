package middlewares

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"{{ cookiecutter.go_module }}/configs"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter applies a per-IP token bucket rate limit and evicts idle
// entries periodically.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rate     rate.Limit
	burst    int
	enabled  bool
	now      func() time.Time
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// NewRateLimiter starts a limiter allowing rps requests per second with the
// given burst; when enabled is false the middleware passes requests through.
// Call Stop once the limiter is no longer needed to release its background
// cleanup goroutine.
func NewRateLimiter(rps float64, burst int, enabled bool) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		enabled:  enabled,
		now:      time.Now,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	if enabled {
		go rl.cleanup()
	} else {
		close(rl.done)
	}
	return rl
}

func (rl *RateLimiter) cleanup() {
	defer close(rl.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.limiters {
				if rl.now().Sub(v.lastSeen) > 3*time.Minute {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Stop terminates the background cleanup goroutine and waits until it has
// finished. It is safe to call Stop multiple times and from several
// goroutines.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stop) })
	<-rl.done
}

// NewRateLimiterFromConfig builds the limiter from the application
// configuration, which makes it injectable as a single fx dependency that can
// also be stopped during shutdown.
func NewRateLimiterFromConfig(cfg *configs.Config) *RateLimiter {
	return NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, cfg.RateLimitEnabled)
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

// Middleware returns a handler that answers 429 for client IPs over their
// limit, skipping the paths given in skipPaths.
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
		reserve := limiter.Reserve()
		delay := reserve.Delay()
		if delay > 0 {
			// The reservation took a token without being granted in time;
			// return it so the client can try again.
			reserve.Cancel()
			retry := int(math.Ceil(delay.Seconds()))
			if retry < 1 {
				retry = 1
			}
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": "ErrRateLimited", "message": "too many requests"})
			return
		}
		c.Next()
	}
}
