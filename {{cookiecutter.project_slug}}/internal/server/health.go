// Package server wires the Gin router, the HTTP server and the health check
// endpoints.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
{% if cookiecutter.use_db == "yes" %}
	"gorm.io/gorm"
{% endif %}
{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
	"github.com/redis/go-redis/v9"
{% endif %}
)

const readinessTimeout = 2 * time.Second

// CheckFunc reports whether a dependency is able to serve requests.
type CheckFunc func(context.Context) error

// Readiness aggregates the dependency checks exposed at /readyz. Unlike
// /healthz (liveness) this endpoint fails when a backing service is down.
type Readiness struct {
	checks map[string]CheckFunc
}

// NewReadiness returns a Readiness that pings the enabled dependencies
// (database, Redis) before serving /readyz.
func NewReadiness({% if cookiecutter.use_db == "yes" %}db *gorm.DB{% endif %}{% if cookiecutter.use_db == "yes" and (cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes") %}, {% endif %}{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}rdb *redis.Client{% endif %}) *Readiness {
	r := &Readiness{checks: map[string]CheckFunc{}}
{% if cookiecutter.use_db == "yes" %}
	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			r.Add("db", func(ctx context.Context) error {
				return sqlDB.PingContext(ctx)
			})
		}
	}
{% endif %}
{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
	if rdb != nil {
		r.Add("redis", func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		})
	}
{% endif %}
	return r
}

// Add registers an additional dependency check.
func (r *Readiness) Add(name string, check CheckFunc) {
	r.checks[name] = check
}

// Handler runs every check with a short deadline and answers 200 when all of
// them succeed, 503 naming the failing ones otherwise.
func (r *Readiness) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		details := gin.H{}

		for name, check := range r.checks {
			ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
			if err := check(ctx); err != nil {
				details[name] = err.Error()
			}
			cancel()
		}

		if len(details) == 0 {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "checks": details})
	}
}
