package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/common/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func jwtTestSecret() string {
	return "testing-secret-change-me-0123456789abcdef"
}

func newJwtApp() *gin.Engine {
	gin.SetMode(gin.TestMode)
	configs.Env.JwtSecret = jwtTestSecret()
	configs.Env.JwtIssuer = "go-gin-template"
	configs.Env.JwtAudience = "go-gin-template"

	app := gin.New()
	app.Use(ErrorHandler())
	app.Use(JwtAuth())
	app.GET("/private", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	})
	return app
}

func TestJwtAuthRejectsMissingHeader(t *testing.T) {
	app := newJwtApp()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJwtAuthRejectsWrongScheme(t *testing.T) {
	app := newJwtApp()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJwtAuthRejectsTruncatedBearer(t *testing.T) {
	app := newJwtApp()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer")
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJwtAuthAcceptsValidToken(t *testing.T) {
	app := newJwtApp()

	userID := uuid.New()
	opts := auth.JwtOptions{Issuer: "go-gin-template", Audience: "go-gin-template"}
	token, err := auth.GenerateJwtToken(jwtTestSecret(), userID, opts)
	if err != nil {
		t.Fatalf("GenerateJwtToken returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), userID.String()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
