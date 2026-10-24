package configs

{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
import (
	"fmt"
	"log/slog"
	"os"
)
{% else %}
import (
	"log/slog"
	"os"
)
{% endif %}

func Bootstrap() error {
	if err := LoadEnv(); err != nil {
		return err
	}
	if err := Env.Validate(); err != nil {
		return err
	}
	InitLogger()
{% if cookiecutter.use_db == "yes" %}
	if err := InitDBClient(); err != nil {
		return err
	}
{% endif %}
{% if cookiecutter.use_redis == "yes" %}
	if err := InitRedisClient(); err != nil {
		return err
	}
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
	if err := InitAsynqClient(); err != nil {
		return err
	}
{% endif %}
	return nil
}

func MustBootstrap() {
	if err := Bootstrap(); err != nil {
		logger := GetLogger()
		logger.Error("bootstrap failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func Shutdown() {
{% if cookiecutter.use_asynq == "yes" %}
	if asynqClient != nil {
		if err := asynqClient.Close(); err != nil {
			fmt.Println("failed to close asynq client:", err)
		}
	}
{% endif %}
{% if cookiecutter.use_redis == "yes" %}
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			fmt.Println("failed to close redis connection:", err)
		}
	}
{% endif %}
{% if cookiecutter.use_db == "yes" %}
	if db != nil {
		dbSQL, err := db.DB()
		if err != nil {
			fmt.Println("failed to get database connection:", err)
			return
		}
		if err := dbSQL.Close(); err != nil {
			fmt.Println("failed to close database connection:", err)
		}
	}
{% endif %}
}
