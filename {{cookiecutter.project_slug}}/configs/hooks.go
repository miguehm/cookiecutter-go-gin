package configs

{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
import "fmt"
{% endif %}

func Bootstrap() {
	InitLogger()
{% if cookiecutter.use_db == "yes" %}
	InitDBClient()
{% endif %}
{% if cookiecutter.use_redis == "yes" %}
	InitRedisClient()
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
	InitAsynqClient()
{% endif %}
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
