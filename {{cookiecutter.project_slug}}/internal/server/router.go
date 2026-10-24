package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"{{ cookiecutter.go_module }}/configs"
{% if cookiecutter.use_db == "yes" %}
	"{{ cookiecutter.go_module }}/internal/controllers"
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
	"{{ cookiecutter.go_module }}/internal/core/asynctasks"
{% endif %}
	"{{ cookiecutter.go_module }}/internal/middlewares"

{% if cookiecutter.use_pprof == "yes" %}
	"github.com/gin-contrib/pprof"
{% endif %}
{% if cookiecutter.use_sentry == "yes" %}
	sentrygin "github.com/getsentry/sentry-go/gin"
{% endif %}
	"github.com/gin-contrib/cors"
	gin "github.com/gin-gonic/gin"
{% if cookiecutter.use_swagger == "yes" %}
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
{% endif %}
)

func NewRouter(cfg *configs.Config, logger *slog.Logger, readiness *Readiness{% if cookiecutter.use_db == "yes" %}, userController *controllers.UserController{% endif %}) (*gin.Engine, error) {
	gin.SetMode(cfg.GinMode)
	router := gin.New()
	_ = router.SetTrustedProxies(trustedProxiesList(cfg))
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())
	router.Use(middlewares.SecurityHeaders(cfg))
	router.Use(middlewares.BodyLimit(cfg.MaxBodyBytes))
{% if cookiecutter.use_sentry == "yes" %}
	router.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
{% endif %}
	router.Use(cors.New(corsConfig(cfg)))
	router.Use(middlewares.ErrorHandler(logger))
	router.Use(middlewares.RequestLogger(logger, cfg))

	rateLimiter := middlewares.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, cfg.RateLimitEnabled)
	router.Use(rateLimiter.Middleware("/healthz", "/readyz"))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.GET("/readyz", readiness.Handler())

{% if cookiecutter.use_db == "yes" %}
	router.POST("/signup", userController.CreateUser)
	router.POST("/login", userController.Login)
	router.GET("/me", middlewares.JwtAuth(cfg), userController.Me)
{% endif %}

{% if cookiecutter.use_swagger == "yes" %}
	if cfg.EnableSwagger {
		router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
{% endif %}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "ErrNotFound", "error": "ErrNotFound"})
	})

{% if cookiecutter.use_asynq == "yes" %}
	asynctasks.AddAsynqmonRoute(router, cfg, logger)
{% endif %}
{% if cookiecutter.use_pprof == "yes" %}
	if cfg.EnablePPROF {
		pprof.Register(router, "pprof")
	}
{% endif %}

	return router, nil
}

func corsConfig(cfg *configs.Config) cors.Config {
	origins := allowedOriginsList(cfg)
	if len(origins) == 0 {
		return cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
			ExposeHeaders:    []string{"X-Request-ID"},
			AllowCredentials: false,
			MaxAge:           24 * time.Hour,
		}
	}
	creds := cfg.CORSAllowCredentials
	if creds {
		for _, o := range origins {
			if o == "*" {
				creds = false
				break
			}
		}
	}
	return cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: creds,
		MaxAge:           24 * time.Hour,
	}
}

func allowedOriginsList(cfg *configs.Config) []string {
	if cfg.CORSAllowedOrigins == "" {
		return nil
	}
	parts := strings.Split(cfg.CORSAllowedOrigins, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func trustedProxiesList(cfg *configs.Config) []string {
	if cfg.TrustedProxies == "" {
		return nil
	}
	parts := strings.Split(cfg.TrustedProxies, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
