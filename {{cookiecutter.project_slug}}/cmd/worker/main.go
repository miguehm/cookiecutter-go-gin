// Command worker runs the Asynq task consumer.
package main

import (
	"context"
	"log/slog"

	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/core/asynctasks"
	"{{ cookiecutter.go_module }}/internal/core/asynctasks/tasks"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			configs.NewConfig,
			configs.ProvideLogger,
			newAsynqServer,
			newServeMux,
		),
		fx.Invoke(runWorker),
	)
	app.Run()
}

func newAsynqServer(cfg *configs.Config) *asynq.Server {
	return asynq.NewServer(
		configs.AsynqRedisOpt(cfg),
		asynq.Config{
			Concurrency: cfg.WorkerConcurrency,
			Queues: map[string]int{
				"default": 1,
			},
		},
	)
}

func newServeMux(logger *slog.Logger) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.Use(asynctasks.LoggingMiddleware(logger))
	mux.HandleFunc(tasks.TypeEmailDelivery, tasks.HandleEmailDelivery)
	mux.HandleFunc(tasks.TypeDebugTask, tasks.HandleDebugTask)
	return mux
}

// runWorker starts the Asynq server during boot and drains in-flight tasks on
// shutdown, which is what makes SIGTERM a graceful stop.
func runWorker(lc fx.Lifecycle, srv *asynq.Server, mux *asynq.ServeMux, cfg *configs.Config, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			logger.Info("starting asynq worker", "concurrency", cfg.WorkerConcurrency)
			return srv.Start(mux)
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down asynq worker")
			done := make(chan struct{})
			go func() {
				srv.Shutdown()
				close(done)
			}()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}
