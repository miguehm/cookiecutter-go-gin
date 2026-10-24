package asynctasks

import (
	"encoding/json"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"{{ cookiecutter.go_module }}/configs"
)

// AddAsynqmonRoute mounts the asynq dashboard behind HTTP basic auth. The
// credentials are validated at boot, so a missing or default credential set
// only means the route is not exposed.
func AddAsynqmonRoute(r *gin.Engine, cfg *configs.Config, logger *slog.Logger) {
	if !cfg.EnableAsynqmon {
		return
	}
	if cfg.BasicAuthUser == "" || cfg.BasicAuthPass == "" ||
		(cfg.BasicAuthUser == "dev" && cfg.BasicAuthPass == "dev") {
		logger.Warn("asynqmon disabled: BASIC_AUTH_USER/PASS are missing or use the dev/dev defaults")
		return
	}
	basicAuthMidd := gin.BasicAuth(gin.Accounts{
		cfg.BasicAuthUser: cfg.BasicAuthPass,
	})

	h := asynqmon.New(asynqmon.Options{
		RootPath:     "/asynqmon",
		RedisConnOpt: configs.AsynqRedisOpt(cfg),
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
