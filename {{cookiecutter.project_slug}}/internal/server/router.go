package server

import (
	"log/slog"
	"net/http"
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

// NewRouter assembles the Gin engine with every middleware, the health
// endpoints and the API routes.
func NewRouter(cfg *configs.Config, logger *slog.Logger, readiness *Readiness, rateLimiter *middlewares.RateLimiter{% if cookiecutter.use_db == "yes" %}, userController *controllers.UserController{% endif %}) (*gin.Engine, error) {
	gin.SetMode(cfg.GinMode)
	router := gin.New()
	if err := router.SetTrustedProxies(configs.SplitList(cfg.TrustedProxies)); err != nil {
		logger.Warn("could not set the trusted proxies; client IPs may be reported incorrectly", "err", err)
	}
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
	origins := configs.SplitList(cfg.CORSAllowedOrigins)
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
