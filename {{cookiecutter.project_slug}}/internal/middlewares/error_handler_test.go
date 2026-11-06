package middlewares

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"{{ cookiecutter.go_module }}/internal/common/responses"
	"{{ cookiecutter.go_module }}/internal/core/apierrors"
)

func newErrorHandlerApp(t *testing.T, logger *slog.Logger, route gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	app := gin.New()
	app.Use(ErrorHandler(logger))
	app.GET("/boom", route)
	return app
}

func TestErrorHandlerRendersAPIError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := newErrorHandlerApp(t, logger, func(c *gin.Context) {
		responses.AbortWithAPIError(c, apierrors.ErrEmailTaken)
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/boom", nil))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ErrEmailTaken") {
		t.Errorf("body = %s, want ErrEmailTaken", body)
	}
}

func TestErrorHandlerRendersBindErrorsWithDetails(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	type payload struct {
		Email string `form:"email" binding:"required"`
	}
	app := newErrorHandlerApp(t, logger, func(c *gin.Context) {
		var p payload
		if err := c.ShouldBind(&p); err != nil {
			responses.AbortWithBindError(c, err)
		}
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/boom", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ErrBadRequest") || !strings.Contains(body, "details") {
		t.Errorf("body = %s, want ErrBadRequest with details", body)
	}
	if !strings.Contains(body, "Email is required") {
		t.Errorf("body = %s, want the field detail message", body)
	}
}

func TestErrorHandlerUnexpectedErrorIsNotExposed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := newErrorHandlerApp(t, logger, func(c *gin.Context) {
		responses.AbortWithError(c, errors.New("database exploded"))
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ErrUnexpected") {
		t.Errorf("body = %s, want ErrUnexpected", body)
	}
	if strings.Contains(body, "database exploded") {
		t.Errorf("body = %s, internal error must not be exposed", body)
	}
}

func TestErrorHandlerWithoutErrorsWritesNothing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := newErrorHandlerApp(t, logger, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/boom", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestValidationErrorToText(t *testing.T) {
	renderer := validator.New()
	type form struct {
		Email    string `validate:"required,email"`
		Password string `validate:"min=8,max=64"`
	}
	var errs validator.ValidationErrors
	if err := renderer.Struct(form{}); err == nil {
		t.Fatal("expected validation errors")
	} else if !errors.As(err, &errs) {
		t.Fatalf("unexpected error type %T", err)
	} else {
		var messages []string
		for _, fieldErr := range errs {
			messages = append(messages, ValidationErrorToText(fieldErr))
		}
		joined := strings.Join(messages, "|")
		if !strings.Contains(joined, "Email is required") {
			t.Errorf("messages = %q, want required clause", joined)
		}
	}
}
