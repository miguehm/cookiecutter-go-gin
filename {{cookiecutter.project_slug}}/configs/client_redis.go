package configs

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// RedisOptions builds the connection options shared by the cache client and
// the Asynq workers.
func RedisOptions(cfg *Config) *redis.Options {
	var tlsConfig *tls.Config
	if cfg.RedisEnableSsl {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	return &redis.Options{
		Addr:      fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:  cfg.RedisPass,
		DB:        cfg.RedisDB,
		TLSConfig: tlsConfig,
	}
}

// NewRedis connects to Redis and verifies the connection with a ping. The
// caller owns the returned client and must close it with CloseRedis.
func NewRedis(cfg *Config) (*redis.Client, error) {
	client := redis.NewClient(RedisOptions(cfg))
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}
	return client, nil
}

// CloseRedis closes the connection pool.
func CloseRedis(client *redis.Client) error {
	if client == nil {
		return nil
	}
	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}
	return nil
}

// ProvideRedis is the fx aware variant of NewRedis: the connection is closed
// on application shutdown.
func ProvideRedis(lc fx.Lifecycle, cfg *Config) (*redis.Client, error) {
	client, err := NewRedis(cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return CloseRedis(client)
		},
	})
	return client, nil
}
