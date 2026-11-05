// Command http runs the HTTP API server.
package main

import (
	"context"
	"log/slog"

	"{{ cookiecutter.go_module }}/configs"
{% if cookiecutter.use_db == "yes" %}
	"{{ cookiecutter.go_module }}/internal/controllers"
	"{{ cookiecutter.go_module }}/internal/core/repositories"
	"{{ cookiecutter.go_module }}/internal/core/services"
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
	"{{ cookiecutter.go_module }}/internal/core/asynctasks"
{% endif %}
	"{{ cookiecutter.go_module }}/internal/middlewares"
	"{{ cookiecutter.go_module }}/internal/server"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			configs.NewConfig,
			configs.ProvideLogger,
{% if cookiecutter.use_db == "yes" %}
			configs.ProvideDB,
			repositories.NewUserRepository,
			services.NewUserService,
			controllers.NewUserController,
{% endif %}
{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
			configs.ProvideRedis,
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
			configs.ProvideAsynqClient,
			asynctasks.NewPublisher,
{% endif %}
			server.NewReadiness,
			middlewares.NewRateLimiterFromConfig,
			server.NewRouter,
			server.NewServer,
		),
		fx.Invoke(registerServer),
	)
	app.Run()
}

// registerServer binds the HTTP listener and the rate limiter to the fx
// lifecycle: the port is reserved during startup (so a bind failure aborts
// the boot) and any error returned by Serve shuts the application down with
// a non-zero exit code. The rate limiter's cleanup goroutine is released
// after the server drains the in-flight requests.
func registerServer(lc fx.Lifecycle, shutdowner fx.Shutdowner, s *server.Server, rateLimiter *middlewares.RateLimiter, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if err := s.Start(); err != nil {
				return err
			}
			go func() {
				if err := s.Serve(); err != nil {
					logger.Error("server failed", "err", err)
					if err := shutdowner.Shutdown(fx.ExitCode(1)); err != nil {
						logger.Error("could not stop the application", "err", err)
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping http server")
			serveErr := s.Shutdown(ctx)
			rateLimiter.Stop()
			return serveErr
		},
	})
}
