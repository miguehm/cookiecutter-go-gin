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
	"{{ cookiecutter.go_module }}/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newTestConfig() *configs.Config {
	return testutil.NewConfig()
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := newTestConfig()
	readiness := NewReadiness({% if cookiecutter.use_db == "yes" %}nil{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}, nil{% endif %}{% elif cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil{% endif %})
{% if cookiecutter.use_db == "yes" %}
	controller := controllers.NewUserController(nil, cfg)
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness, controller)
{% else %}
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness)
{% endif %}
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}
	return router
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
	rec := do(newTestRouter(t), http.MethodGet, "/healthz", nil)

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

func TestRouterReadyzWithoutDependencies(t *testing.T) {
	rec := do(newTestRouter(t), http.MethodGet, "/readyz", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestRouterUnknownPathReturns404(t *testing.T) {
	rec := do(newTestRouter(t), http.MethodGet, "/definitely-not-here", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ErrNotFound") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestRouterCORSWildcardsOrigin(t *testing.T) {
	cfg := newTestConfig()
	cfg.CORSAllowedOrigins = "*"
	cfg.CORSAllowCredentials = false
{% if cookiecutter.use_db == "yes" %}
	readiness := NewReadiness({% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil, nil{% else %}nil{% endif %})
	controller := controllers.NewUserController(nil, cfg)
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness, controller)
{% else %}
	readiness := NewReadiness({% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil{% endif %})
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness)
{% endif %}
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}

	rec := do(router, http.MethodGet, "/healthz", map[string]string{"Origin": "https://app.example.com"})

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want \"*\"", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Error("credentials must be off with wildcard origin")
	}
}

func TestRouterCORSExplicitOrigins(t *testing.T) {
	cfg := newTestConfig()
	cfg.CORSAllowedOrigins = "https://app.example.com"
	cfg.CORSAllowCredentials = true
{% if cookiecutter.use_db == "yes" %}
	readiness := NewReadiness({% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil, nil{% else %}nil{% endif %})
	controller := controllers.NewUserController(nil, cfg)
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness, controller)
{% else %}
	readiness := NewReadiness({% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil{% endif %})
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness)
{% endif %}
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}

	rec := do(router, http.MethodGet, "/healthz", map[string]string{"Origin": "https://app.example.com"})

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestRouterCORSRejectsUnknownOrigin(t *testing.T) {
	cfg := newTestConfig()
	cfg.CORSAllowedOrigins = "https://app.example.com"
	cfg.CORSAllowCredentials = false
{% if cookiecutter.use_db == "yes" %}
	readiness := NewReadiness({% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil, nil{% else %}nil{% endif %})
	controller := controllers.NewUserController(nil, cfg)
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness, controller)
{% else %}
	readiness := NewReadiness({% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil{% endif %})
	router, err := NewRouter(cfg, testutil.NewLogger(), readiness)
{% endif %}
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}

	rec := do(router, http.MethodGet, "/healthz", map[string]string{"Origin": "https://evil.example.com"})

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for a disallowed origin", rec.Code)
	}
}
