// Package responses writes the JSON envelopes shared by the API endpoints.
package responses

import (
	"net/http"

	"{{ cookiecutter.go_module }}/internal/core/apierrors"
	"{{ cookiecutter.go_module }}/pkg/pagination"

	"github.com/gin-gonic/gin"
)

// ResponseOK is the envelope of a successful, non paginated response.
type ResponseOK struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ResponseOKWithPagination is the envelope of a successful paginated
// response.
type ResponseOKWithPagination struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Total   int64       `json:"total"` // TODO: refactor it
	Data interface{} `json:"data"`
}

// Ok answers 200 with data.
func Ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, ResponseOK{
		Code:    "Success",
		Message: "Success",
		Data:    data,
	})
}

// Created answers 201 with the created representation of the resource.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, ResponseOK{
		Code:    "Success",
		Message: "Success",
		Data:    data,
	})
}

// OkWithPagination answers 200 with the items of the given page.
func OkWithPagination(c *gin.Context, page *pagination.Page) {
	c.JSON(http.StatusOK, ResponseOKWithPagination{
		Code:    "Success",
		Message: "Success",
		Data:    page.Items,
		//Total:   page.Total,
	})
}

// AbortWithAPIError aborts the request with a public API error, so the error
// handler renders its status and code.
func AbortWithAPIError(ctx *gin.Context, err *apierrors.APIError) {
	_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
	ctx.Abort()
}

// AbortWithError aborts the request with a public error that is logged but
// rendered as an unexpected error by the error handler.
func AbortWithError(ctx *gin.Context, err error) {
	_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
	ctx.Abort()
}

// AbortWithBindError aborts the request with a binding error, so the error
// handler answers 400 with the field details.
func AbortWithBindError(ctx *gin.Context, err error) {
	_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
	ctx.Abort()
}
