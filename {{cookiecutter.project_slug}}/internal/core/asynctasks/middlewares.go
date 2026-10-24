package asynctasks

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// LoggingMiddleware returns an Asynq middleware that logs every processed
// task with the injected logger.
func LoggingMiddleware(logger *slog.Logger) func(asynq.Handler) asynq.Handler {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			start := time.Now()
			if err := h.ProcessTask(ctx, t); err != nil {
				logger.Error("failed to process task", "type", t.Type(), "err", err, "elapsed", time.Since(start))
				return err
			}
			logger.Info("task processed", "type", t.Type(), "elapsed", time.Since(start))
			return nil
		})
	}
}
