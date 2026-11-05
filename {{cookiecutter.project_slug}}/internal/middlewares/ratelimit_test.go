package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterStopReleasesCleanupGoroutine(t *testing.T) {
	rl := NewRateLimiter(1, 1, true)

	finished := make(chan struct{})
	go func() {
		rl.Stop()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not release the cleanup goroutine in time")
	}

	rl.Stop() // must be idempotent
}

func TestRateLimiterStopWithoutCleanupGoroutine(t *testing.T) {
	rl := NewRateLimiter(1, 1, false)
	rl.Stop() // must not block when the limiter was disabled
}

func TestRateLimiterEnforcesBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := gin.New()
	app.Use(NewRateLimiter(1, 2, true).Middleware("/healthz"))
	app.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	statuses := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		statuses = append(statuses, rec.Code)
	}

	if statuses[0] != http.StatusOK || statuses[1] != http.StatusOK {
		t.Fatalf("burst should allow 2 requests: %v", statuses)
	}
	if statuses[2] != http.StatusTooManyRequests {
		t.Fatalf("3rd request should be rate limited: %v", statuses)
	}
	if statuses[3] != http.StatusTooManyRequests {
		t.Fatalf("4th request should be rate limited: %v", statuses)
	}
}

func TestRateLimiterSkipsConfiguredPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := gin.New()
	app.Use(NewRateLimiter(1, 1, true).Middleware("/healthz"))
	app.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d = %d, healthz must never be limited", i+1, rec.Code)
		}
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := gin.New()
	app.Use(NewRateLimiter(1, 1, false).Middleware())
	app.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter blocked request %d", i+1)
		}
	}
}

func TestRateLimiterRetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := gin.New()
	app.Use(NewRateLimiter(1, 1, true).Middleware())
	app.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if i == 1 && rec.Code == http.StatusTooManyRequests {
			if got := rec.Header().Get("Retry-After"); got != "1" {
				t.Errorf("Retry-After = %q, want \"1\"", got)
			}
		}
	}
}
