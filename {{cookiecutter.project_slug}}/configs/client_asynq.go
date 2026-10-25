package configs

import (
	"context"
	"crypto/tls"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"
)

// AsynqRedisOpt builds the Redis connection options used by the Asynq client,
// the worker and the asynqmon dashboard.
func AsynqRedisOpt(cfg *Config) asynq.RedisClientOpt {
	var tlsConfig *tls.Config
	if cfg.RedisEnableSsl {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	return asynq.RedisClientOpt{
		Addr:      cfg.RedisHost + ":" + cfg.RedisPort,
		Password:  cfg.RedisPass,
		TLSConfig: tlsConfig,
		DB:        cfg.RedisTaskDB,
	}
}

// NewAsynqClient creates the Asynq client used to enqueue tasks. The caller
// owns it and must close it with CloseAsynqClient.
func NewAsynqClient(cfg *Config) *asynq.Client {
	return asynq.NewClient(AsynqRedisOpt(cfg))
}

// CloseAsynqClient closes the Asynq client.
func CloseAsynqClient(client *asynq.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}

// ProvideAsynqClient is the fx aware variant of NewAsynqClient: the client is
// closed on application shutdown.
func ProvideAsynqClient(lc fx.Lifecycle, cfg *Config) *asynq.Client {
	client := NewAsynqClient(cfg)
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return CloseAsynqClient(client)
		},
	})
	return client
}
