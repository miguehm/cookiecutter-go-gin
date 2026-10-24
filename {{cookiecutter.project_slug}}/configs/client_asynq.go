package configs

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/hibiken/asynq"
)

var asynqClient *asynq.Client
var asynqInitErr error
var onceInitAsynqClient sync.Once

func GetAsynqRedisOpt() asynq.RedisClientOpt {
	var tlsConfig *tls.Config
	if Env.RedisEnableSsl {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	return asynq.RedisClientOpt{
		Addr:      fmt.Sprintf("%s:%s", Env.RedisHost, Env.RedisPort),
		Password:  Env.RedisPass,
		TLSConfig: tlsConfig,
		DB:        Env.RedisTaskDB,
	}
}

func InitAsynqClient() error {
	onceInitAsynqClient.Do(func() {
		asynqClient = asynq.NewClient(GetAsynqRedisOpt())
		if asynqClient == nil {
			asynqInitErr = fmt.Errorf("asynq client not initialized")
		}
	})
	return asynqInitErr
}

func GetAsynqClient() (*asynq.Client, error) {
	if asynqInitErr != nil {
		return nil, asynqInitErr
	}
	if asynqClient == nil {
		return nil, fmt.Errorf("asynq client not initialized")
	}
	return asynqClient, nil
}
