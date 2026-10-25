package configs

import (
{% if cookiecutter.use_sentry == "yes" %}
	"fmt"
{% endif %}
	"log/slog"
	"os"

{% if cookiecutter.use_sentry == "yes" %}
	"context"
	"time"

	"github.com/getsentry/sentry-go"
{% endif %}
	"github.com/phsym/console-slog"
	slogmulti "github.com/samber/slog-multi"
{% if cookiecutter.use_sentry == "yes" %}
	slogsentry "github.com/samber/slog-sentry/v2"
{% endif %}
	"go.uber.org/fx"
)

// NewLogger builds the application logger from the configuration. Nothing is
// stored in package level state: callers keep the returned logger and pass it
// around.
func NewLogger(cfg *Config) (*slog.Logger, error) {
	handlers := []slog.Handler{
		console.NewHandler(os.Stdout, &console.HandlerOptions{Level: slog.LevelInfo, NoColor: true}),
	}

{% if cookiecutter.use_sentry == "yes" %}
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			EnableTracing:    false,
			TracesSampleRate: 1.0,
			Environment:      cfg.Stage,
		}); err != nil {
			return nil, fmt.Errorf("failed to initialize sentry: %w", err)
		}
		handlers = append(handlers, slogsentry.Option{Level: slog.LevelWarn}.NewSentryHandler())
	}
{% else %}
	// Without the Sentry handler nothing reads the configuration here.
	_ = cfg
{% endif %}

	return slog.New(slogmulti.Fanout(handlers...)), nil
}

{% if cookiecutter.use_sentry == "yes" %}
// ProvideLogger is the fx aware variant of NewLogger: it registers the
// lifecycle hook needed to flush pending events before the app stops.
func ProvideLogger(lc fx.Lifecycle, cfg *Config) (*slog.Logger, error) {
	logger, err := NewLogger(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.SentryDSN != "" {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				sentry.Flush(2 * time.Second)
				return nil
			},
		})
	}
	return logger, nil
}
{% else %}
// ProvideLogger is the fx aware variant of NewLogger.
func ProvideLogger(_ fx.Lifecycle, cfg *Config) (*slog.Logger, error) {
	return NewLogger(cfg)
}
{% endif %}
