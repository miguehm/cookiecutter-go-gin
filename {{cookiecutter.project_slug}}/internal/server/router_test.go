package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{ cookiecutter.go_module }}/configs"
{% if cookiecutter.use_db == "yes" %}
	"{{ cookiecutter.go_module }}/internal/controllers"
{% endif %}

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
{% if cookiecutter.use_db == "yes" %}
	return NewRouter(controllers.NewUserController(nil))
{% else %}
	return NewRouter()
{% endif %}
}

func do(router *gin.Engine, method, target string, header map[string]string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func TestRouterHealthz(t *testing.T) {
	rec := do(newTestRouter(), http.MethodGet, "/healthz", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID header")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
}

func TestRouterUnknownPathReturns404(t *testing.T) {
	rec := do(newTestRouter(), http.MethodGet, "/definitely-not-here", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ErrNotFound") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestRouterCORSWildcardsOrigin(t *testing.T) {
	configs.Env.CORSAllowedOrigins = "*"
	configs.Env.CORSAllowCredentials = false
	defer func() {
		configs.Env.CORSAllowedOrigins = "*"
		configs.Env.CORSAllowCredentials = false
	}()

	rec := do(newTestRouter(), http.MethodGet, "/healthz", map[string]string{"Origin": "https://app.example.com"})

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want \"*\"", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Error("credentials must be off with wildcard origin")
	}
}

func TestRouterCORSExplicitOrigins(t *testing.T) {
	configs.Env.CORSAllowedOrigins = "https://app.example.com"
	configs.Env.CORSAllowCredentials = true
	defer func() {
		configs.Env.CORSAllowedOrigins = "*"
		configs.Env.CORSAllowCredentials = false
	}()

	rec := do(newTestRouter(), http.MethodGet, "/healthz", map[string]string{"Origin": "https://app.example.com"})

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestRouterCORSRejectsUnknownOrigin(t *testing.T) {
	configs.Env.CORSAllowedOrigins = "https://app.example.com"
	configs.Env.CORSAllowCredentials = false
	defer func() {
		configs.Env.CORSAllowedOrigins = "*"
		configs.Env.CORSAllowCredentials = false
	}()

	rec := do(newTestRouter(), http.MethodGet, "/healthz", map[string]string{"Origin": "https://evil.example.com"})

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for a disallowed origin", rec.Code)
	}
}
