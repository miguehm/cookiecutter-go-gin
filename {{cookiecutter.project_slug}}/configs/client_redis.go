package configs

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var redisInitErr error
var onceInitRedisClient sync.Once

func InitRedisClient() error {
	onceInitRedisClient.Do(func() {
		var tlsConfig *tls.Config
		if Env.RedisEnableSsl {
			tlsConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		client := redis.NewClient(&redis.Options{
			Addr:      fmt.Sprintf("%s:%s", Env.RedisHost, Env.RedisPort),
			Password:  Env.RedisPass,
			DB:        Env.RedisDB,
			TLSConfig: tlsConfig,
		})

		ctx := context.Background()
		if err := client.Ping(ctx).Err(); err != nil {
			redisInitErr = fmt.Errorf("failed to connect redis: %w", err)
			return
		}
		rdb = client
	})
	return redisInitErr
}

func GetRedis() (*redis.Client, error) {
	if redisInitErr != nil {
		return nil, redisInitErr
	}
	if rdb == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	return rdb, nil
}
