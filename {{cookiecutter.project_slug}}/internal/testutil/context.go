package testutil

import (
	"io"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func NewContext(method, target string, body io.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, body)

	return ctx, recorder
}
