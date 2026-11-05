package responses

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"{{ cookiecutter.go_module }}/internal/core/apierrors"
	"{{ cookiecutter.go_module }}/pkg/pagination"
)

func newTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c, rec
}

func TestOkEnvelope(t *testing.T) {
	c, rec := newTestContext(t)
	Ok(c, gin.H{"hello": "world"})

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"Success"`) || !strings.Contains(rec.Body.String(), `"data":{"hello":"world"}`) {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestCreatedReturns201(t *testing.T) {
	c, rec := newTestContext(t)
	Created(c, gin.H{"id": 1})
	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
}

func TestOkWithPaginationIncludesTotal(t *testing.T) {
	c, rec := newTestContext(t)
	page := &pagination.Page{Items: []string{"a", "b"}, Total: 42}
	OkWithPagination(c, page)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"total":42`) {
		t.Errorf("body = %s, want total 42", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"data":["a","b"]`) {
		t.Errorf("body = %s, want the page items", rec.Body.String())
	}
}

func TestAbortWithAPIErrorCollectsTypedError(t *testing.T) {
	c, _ := newTestContext(t)
	AbortWithAPIError(c, apierrors.ErrEmailTaken)

	if len(c.Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(c.Errors))
	}
	if c.Errors[0].Type != gin.ErrorTypePublic {
		t.Fatalf("error type = %d, want public", c.Errors[0].Type)
	}
	var apiErr *apierrors.APIError
	if !errors.As(c.Errors[0].Err, &apiErr) {
		t.Fatalf("error %T is not an *apierrors.APIError", c.Errors[0].Err)
	}
	if apiErr.ErrorCode != "ErrEmailTaken" {
		t.Errorf("code = %s", apiErr.ErrorCode)
	}
}

func TestAbortWithBindErrorMarksTypeError(t *testing.T) {
	c, _ := newTestContext(t)
	AbortWithBindError(c, errors.New("binding"))
	if len(c.Errors) != 1 || c.Errors[0].Type != gin.ErrorTypeBind {
		t.Fatalf("expected one bind error, got %v", c.Errors)
	}
}

func TestAbortWithErrorMarksTypeError(t *testing.T) {
	c, _ := newTestContext(t)
	AbortWithError(c, errors.New("oops"))
	if len(c.Errors) != 1 || c.Errors[0].Type != gin.ErrorTypePublic {
		t.Fatalf("expected one public error, got %v", c.Errors)
	}
}
