package middlewares

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	seen := ""
	app := gin.New()
	app.Use(RequestID())
	app.GET("/", func(c *gin.Context) {
		v, ok := c.Get("request_id")
		if !ok {
			t.Error("request_id not set in context")
		}
		id, _ := v.(string)
		seen = id
		c.String(200, "ok")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID response header")
	}
	if seen == "" {
		t.Error("context value empty")
	}
}

func TestRequestIDPropagatesIncomingID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	seen := ""
	app := gin.New()
	app.Use(RequestID())
	app.GET("/", func(c *gin.Context) {
		id, _ := c.Get("request_id")
		seen, _ = id.(string)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "my-trace-id")
	app.ServeHTTP(rec, req)

	if seen != "my-trace-id" {
		t.Errorf("request_id = %q, want %q", seen, "my-trace-id")
	}
	if got := rec.Header().Get("X-Request-ID"); got != "my-trace-id" {
		t.Errorf("X-Request-ID = %q, want %q", got, "my-trace-id")
	}
}
