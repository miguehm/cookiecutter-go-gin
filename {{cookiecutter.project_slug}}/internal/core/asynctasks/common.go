package asynctasks

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"{{ cookiecutter.go_module }}/configs"
)

func AddAsynqmonRoute(r *gin.Engine) {
	if !configs.Env.EnableAsynqmon {
		return
	}
	if configs.Env.BasicAuthUser == "" || configs.Env.BasicAuthPass == "" {
		panic("asynqmon enabled but BASIC_AUTH_USER/BASIC_AUTH_PASS are empty")
	}
	if configs.Env.BasicAuthUser == "dev" && configs.Env.BasicAuthPass == "dev" {
		panic("asynqmon enabled but BASIC_AUTH_USER/PASS must not use dev/dev defaults")
	}
	basicAuthMidd := gin.BasicAuth(gin.Accounts{
		configs.Env.BasicAuthUser: configs.Env.BasicAuthPass,
	})

	h := asynqmon.New(asynqmon.Options{
		RootPath:     "/asynqmon",
		RedisConnOpt: configs.GetAsynqRedisOpt(),
	})

	r.Any(h.RootPath()+"/*a", basicAuthMidd, gin.WrapH(h))
}

func NewTask[T any](taskType string, data T) (*asynq.Task, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(taskType, payload), nil
}

func PublishTask[T any](taskType string, payload T, opts ...asynq.Option) error {
	task, err := NewTask(taskType, payload)
	if err != nil {
		return fmt.Errorf("could not create task: %w", err)
	}
	client, err := configs.GetAsynqClient()
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task, opts...)
	if err != nil {
		return fmt.Errorf("could not enqueue task: %w", err)
	}
	return nil
}
