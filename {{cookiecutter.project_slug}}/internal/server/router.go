package server

import (
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

func NewRouter({% if cookiecutter.use_db == "yes" %}userController *controllers.UserController{% endif %}) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(trustedProxiesList())
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())
	router.Use(middlewares.SecurityHeaders())
	router.Use(middlewares.BodyLimit(configs.Env.MaxBodyBytes))
{% if cookiecutter.use_sentry == "yes" %}
	router.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
{% endif %}
	router.Use(cors.New(corsConfig()))
	router.Use(middlewares.ErrorHandler())
	router.Use(middlewares.RequestLogger())

	rateLimiter := middlewares.NewRateLimiter(configs.Env.RateLimitRPS, configs.Env.RateLimitBurst, configs.Env.RateLimitEnabled)
	router.Use(rateLimiter.Middleware("/healthz"))

{% if cookiecutter.use_db == "yes" %}
	router.GET("/login", userController.SignUp)
{% endif %}

{% if cookiecutter.use_swagger == "yes" %}
	if configs.Env.EnableSwagger {
		router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
{% endif %}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "ErrNotFound", "error": "ErrNotFound"})
	})

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

{% if cookiecutter.use_asynq == "yes" %}
	if configs.Env.EnableAsynqmon {
		asynctasks.AddAsynqmonRoute(router)
	}
{% endif %}
{% if cookiecutter.use_pprof == "yes" %}
	if configs.Env.EnablePPROF {
		pprof.Register(router, "pprof")
	}
{% endif %}

	return router
}

func corsConfig() cors.Config {
	origins := allowedOriginsList()
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
	creds := configs.Env.CORSAllowCredentials
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

func allowedOriginsList() []string {
	if configs.Env.CORSAllowedOrigins == "" {
		return nil
	}
	parts := strings.Split(configs.Env.CORSAllowedOrigins, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func trustedProxiesList() []string {
	if configs.Env.TrustedProxies == "" {
		return nil
	}
	parts := strings.Split(configs.Env.TrustedProxies, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
