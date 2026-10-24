package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"{{ cookiecutter.go_module }}/configs"
)

var sensitiveKeys = map[string]bool{
	"password":        true,
	"passwordconfirm": true,
	"token":           true,
	"accesstoken":     true,
	"refreshtoken":    true,
	"jwt":             true,
	"secret":          true,
	"clientsecret":    true,
	"privatekey":      true,
	"authorization":   true,
	"cookie":          true,
	"set-cookie":      true,
	"apikey":          true,
	"x-api-key":       true,
}

func redactMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if sensitiveKeys[strings.ToLower(k)] {
			out[k] = "[REDACTED]"
			continue
		}
		if mv, ok := v.(map[string]any); ok {
			out[k] = redactMap(mv)
			continue
		}
		out[k] = v
	}
	return out
}

func redactBody(contentType string, body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if len(body) > 4096 {
		body = body[:4096]
	}
	if strings.Contains(contentType, "application/json") {
		var m map[string]any
		if json.Unmarshal(body, &m) == nil {
			rm := redactMap(m)
			if b, err := json.Marshal(rm); err == nil {
				return string(b)
			}
		}
		return "[REDACTED_OR_TRUNCATED]"
	}
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return "[REDACTED_OR_TRUNCATED]"
		}
		for k := range values {
			if sensitiveKeys[strings.ToLower(k)] {
				values.Set(k, "[REDACTED]")
			}
		}
		return values.Encode()
	}
	if strings.Contains(contentType, "multipart/form-data") {
		return "[REDACTED_OR_TRUNCATED]"
	}
	return "[REDACTED_OR_TRUNCATED]"
}

func fmtString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func RequestLogger(logger *slog.Logger, cfg *configs.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		path := c.Request.URL.Path
		if path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		detail := cfg.EnableLogRequestDetail
		var bodyBytes []byte
		if detail && c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		latency := time.Since(startTime)
		reqID, _ := c.Get("request_id")
		if reqID == nil {
			reqID = c.Writer.Header().Get("X-Request-ID")
		}

		attrs := []any{
			slog.String("request_id", fmtString(reqID)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		}

		if cfg.EnableLogRequestDetail {
			q := c.Request.URL.Query()
			for k := range q {
				if sensitiveKeys[strings.ToLower(k)] {
					q.Set(k, "[REDACTED]")
				}
			}
			attrs = append(attrs,
				slog.String("query", q.Encode()),
				slog.String("body", redactBody(c.Request.Header.Get("Content-Type"), bodyBytes)),
			)
		}

		logger.Info("request", attrs...)
	}
}
