package middlewares

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configs.Env.EnableHSTS = false

	ctx, rec := testutil.NewContext(http.MethodGet, "/api/things", nil)
	SecurityHeaders()(ctx)

	header := rec.Header()
	for key, want := range map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Permissions-Policy":      "geolocation=(), microphone=(), camera=()",
		"Cross-Origin-Opener-Policy": "same-origin",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	} {
		if got := header.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if header.Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should be off when ENABLE_HSTS=false")
	}
}

func TestSecurityHeadersDocsRelaxed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, rec := testutil.NewContext(http.MethodGet, "/docs/index.html", nil)
	SecurityHeaders()(ctx)

	if rec.Header().Get("X-Frame-Options") == "DENY" {
		t.Error("docs should not use DENY frame options")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("docs should still set nosniff")
	}
}

func TestSecurityHeadersHSTS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configs.Env.EnableHSTS = true
	defer func() { configs.Env.EnableHSTS = false }()

	ctx, rec := testutil.NewContext(http.MethodGet, "/api/things", nil)
	SecurityHeaders()(ctx)

	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Errorf("HSTS = %q", got)
	}
}

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := strings.NewReader(strings.Repeat("a", 1024))
	ctx, _ := testutil.NewContext(http.MethodPost, "/login", body)
	BodyLimit(16)(ctx)

	if _, err := io.ReadAll(ctx.Request.Body); err == nil {
		t.Fatal("expected an error when body exceeds the limit")
	}
}

func TestBodyLimitAllowsSmallPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := testutil.NewContext(http.MethodPost, "/login", strings.NewReader("ok"))
	BodyLimit(16)(ctx)

	data, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("body = %q", data)
	}
}

func TestRequestIDKeepsIncomingValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, rec := testutil.NewContext(http.MethodGet, "/x", nil)
	ctx.Request.Header.Set("X-Request-ID", "abc-123")
	RequestID()(ctx)

	if got := rec.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("X-Request-ID = %q", got)
	}
}

func TestRequestIDGeneratesValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, rec := testutil.NewContext(http.MethodGet, "/x", nil)
	RequestID()(ctx)

	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected a generated X-Request-ID")
	}
}
