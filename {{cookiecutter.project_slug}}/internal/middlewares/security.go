package middlewares

import (
	"net/http"
	"strings"

	"{{ cookiecutter.go_module }}/configs"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps the request body at maxBytes, answering 413 when exceeded.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// SecurityHeaders sets the browser security headers; the stricter set is
// relaxed on the documentation and monitoring pages so they can still render.
func SecurityHeaders(cfg *configs.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/docs") || strings.HasPrefix(path, "/asynqmon") {
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("X-Frame-Options", "SAMEORIGIN")
			c.Header("Referrer-Policy", "no-referrer")
			c.Next()
			return
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		if cfg.EnableHSTS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
