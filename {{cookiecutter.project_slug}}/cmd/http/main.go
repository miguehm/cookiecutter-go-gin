package main

import (
	"context"
{% if cookiecutter.use_sentry == "yes" %}
	"time"

	"github.com/getsentry/sentry-go"
{% endif %}
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"{{ cookiecutter.go_module }}/configs"
{% if cookiecutter.use_db == "yes" %}
	"{{ cookiecutter.go_module }}/internal/controllers"
	"{{ cookiecutter.go_module }}/internal/core/repositories"
	"{{ cookiecutter.go_module }}/internal/core/services"
{% endif %}
	"{{ cookiecutter.go_module }}/internal/server"
)

func main() {
	configs.MustBootstrap()
	defer configs.Shutdown()
{% if cookiecutter.use_sentry == "yes" %}
	defer sentry.Flush(2 * time.Second)
{% endif %}

	Run()
}

func Run() {
	app := fx.New(
		fx.Provide(
{% if cookiecutter.use_db == "yes" %}
			configs.GetDB,
			repositories.NewUserRepository,
			services.NewUserService,
			controllers.NewUserController,
{% endif %}
			server.NewRouter,
			server.NewServer,
		),
		fx.Invoke(startServer),
	)

	app.Run()
}

func startServer(lc fx.Lifecycle, s *server.Server) {
	logger := configs.GetLogger()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				gin.SetMode(configs.Env.GinMode)
				s.Run()
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("wait for server stop")
			if err := s.HttpServer.Shutdown(ctx); err != nil {
				logger.Error("server forced to shutdown:", "err", err)
			}
			return nil
		},
	})
}
