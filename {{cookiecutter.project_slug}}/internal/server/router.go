package server

import (
	"net/http"
	"time"

{% if cookiecutter.use_pprof == "yes" %}
	"{{ cookiecutter.go_module }}/configs"
{% endif %}
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
	router.Use(gin.Recovery())
{% if cookiecutter.use_sentry == "yes" %}
	router.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
{% endif %}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))
	router.Use(middlewares.ErrorHandler())
	router.Use(middlewares.RequestLogger())

{% if cookiecutter.use_db == "yes" %}
	router.GET("/login", userController.SignUp)
{% endif %}

	// api := r.Group("/api/wallet/v1")
	// api.POST("/login", controllers.PlayerLoginHandler)

	// 	api.POST("/users/:id", middlewares.AuthMiddleware(), s.DeleteUser)

{% if cookiecutter.use_swagger == "yes" %}
	// Docs
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
{% endif %}

	// Not Found Route
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "ErrNotFound", "error": "ErrNotFound"})
	})

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

{% if cookiecutter.use_asynq == "yes" %}
	// AddAsynqmonRoute
	asynctasks.AddAsynqmonRoute(router)
{% endif %}
{% if cookiecutter.use_pprof == "yes" %}
	if configs.Env.EnablePPROF {
		pprof.Register(router, "pprof")
	}
{% endif %}

	return router
}
