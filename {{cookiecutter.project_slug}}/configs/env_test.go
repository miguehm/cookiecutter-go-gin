package configs

import (
	"strings"
	"testing"
)

func TestEnvConfigValidate(t *testing.T) {
	t.Parallel()

	valid := EnvConfig{
		Port:      "8080",
		GinMode:   "debug",
		JwtSecret: "dev-secret",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	withDB := valid
	withDB.DBHost = "127.0.0.1"
	withDB.DBPort = "5432"
	withDB.DBUser = "dev"
	withDB.DBName = "app"
	withDB.DBSslMode = "disable"
	if err := withDB.Validate(); err != nil {
		t.Fatalf("unexpected error with DB: %v", err)
	}

	withRedis := valid
	withRedis.RedisHost = "127.0.0.1"
	withRedis.RedisPort = "6379"
	if err := withRedis.Validate(); err != nil {
		t.Fatalf("unexpected error with Redis: %v", err)
	}
}

func TestEnvConfigValidateMissing(t *testing.T) {
	t.Parallel()

	env := EnvConfig{
		DBHost:    "127.0.0.1",
		DBPort:    "5432",
		RedisHost: "127.0.0.1",
	}

	err := env.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	for _, expected := range []string{"PORT", "GIN_MODE", "JWT_SECRET", "DB_USER", "DB_NAME", "DB_SSL_MODE", "REDIS_PORT"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("error %q does not mention %q", err, expected)
		}
	}
}
