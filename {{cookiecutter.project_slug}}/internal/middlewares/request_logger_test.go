package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{ cookiecutter.go_module }}/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestRedactBodyJSON(t *testing.T) {
	got := redactBody("application/json", []byte(`{"email":"a@b.c","password":"hunter2","nested":{"token":"abc"}}`))
	for _, leaked := range []string{"hunter2", "abc"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("body leaked %q: %s", leaked, got)
		}
	}
	if !strings.Contains(got, "a@b.c") {
		t.Fatalf("expected non sensitive field to survive: %s", got)
	}
}

func TestRedactBodyForm(t *testing.T) {
	got := redactBody("application/x-www-form-urlencoded", []byte("user=bob&password=hunter2"))
	if strings.Contains(got, "hunter2") {
		t.Fatalf("body leaked password: %s", got)
	}
	if !strings.Contains(got, "user=bob") {
		t.Fatalf("expected non sensitive field to survive: %s", got)
	}
}

func TestRedactBodyUnknownType(t *testing.T) {
	if got := redactBody("application/octet-stream", []byte("binary")); got != "[REDACTED_OR_TRUNCATED]" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactBodyEmpty(t *testing.T) {
	if got := redactBody("application/json", nil); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactMapIsRecursive(t *testing.T) {
	out := redactMap(map[string]any{
		"outer": map[string]any{"password": "x", "name": "ok"},
		"top":   "y",
	})
	inner, ok := out["outer"].(map[string]any)
	if !ok {
		t.Fatalf("outer is %T", out["outer"])
	}
	if inner["password"] != "[REDACTED]" {
		t.Fatalf("password = %v", inner["password"])
	}
	if inner["name"] != "ok" {
		t.Fatalf("name = %v", inner["name"])
	}
}

func TestRequestLoggerSkipsHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := gin.New()
	app.Use(RequestLogger(testutil.NewLogger(), testutil.NewConfig()))
	app.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	app.GET("/readyz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for _, path := range []string{"/healthz", "/readyz"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status for %s = %d", path, rec.Code)
		}
	}
}
