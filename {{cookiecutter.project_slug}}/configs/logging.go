package configs

import (
{% if cookiecutter.use_sentry == "yes" %}
	"log"
{% endif %}
	"log/slog"
	"os"
	"sync"

{% if cookiecutter.use_sentry == "yes" %}
	"github.com/getsentry/sentry-go"
{% endif %}
	"github.com/phsym/console-slog"
	slogmulti "github.com/samber/slog-multi"
{% if cookiecutter.use_sentry == "yes" %}
	slogsentry "github.com/samber/slog-sentry/v2"
{% endif %}
)

var logger *slog.Logger
var onceInitLogger sync.Once

func InitLogger() {
	GetLogger()
}

func GetLogger() *slog.Logger {
	onceInitLogger.Do(func() {
		handlers := []slog.Handler{
			console.NewHandler(os.Stdout, &console.HandlerOptions{Level: slog.LevelInfo, NoColor: true}),
		}

{% if cookiecutter.use_sentry == "yes" %}
		if Env.SentryDSN != "" {
			err := sentry.Init(sentry.ClientOptions{
				Dsn:              Env.SentryDSN,
				EnableTracing:    false,
				TracesSampleRate: 1.0,
				Environment:      Env.Stage,
			})
			if err != nil {
				log.Printf("failed to initialize sentry: %v", err)
			} else {
				handlers = append(handlers, slogsentry.Option{Level: slog.LevelWarn}.NewSentryHandler())
			}
		}
{% endif %}

		logger = slog.New(slogmulti.Fanout(handlers...))
		slog.SetDefault(logger)
	})
	return logger
}
