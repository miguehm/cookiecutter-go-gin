package configs

import (
	"strings"
	"testing"
)

func TestEnvConfigValidate(t *testing.T) {
	valid := EnvConfig{
		Stage:     "dev",
		Port:      "8080",
		GinMode:   "debug",
		JwtSecret: "dev-secret-change-me-0123456789abcdef",
		MaxBodyBytes: 1048576,
		RateLimitRPS: 5,
		RateLimitBurst: 10,
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
	env := EnvConfig{
		Stage:        "dev",
		DBHost:       "127.0.0.1",
		DBPort:       "5432",
		RedisHost:    "127.0.0.1",
		MaxBodyBytes: 1048576,
		RateLimitRPS: 5,
		RateLimitBurst: 10,
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

func TestEnvConfigValidateJWTSecretLength(t *testing.T) {
	env := EnvConfig{
		Stage:         "dev",
		Port:          "8080",
		GinMode:       "debug",
		JwtSecret:     "short",
		MaxBodyBytes:  1048576,
		RateLimitRPS:  5,
		RateLimitBurst: 10,
	}
	if err := env.Validate(); err == nil {
		t.Fatal("expected error for short JWT secret")
	}
}

func validEnv() EnvConfig {
	return EnvConfig{
		Stage:           "dev",
		Port:            "8080",
		GinMode:         "debug",
		JwtSecret:       "dev-secret-change-me-0123456789abcdef",
		MaxBodyBytes:    1048576,
		RateLimitRPS:    5,
		RateLimitBurst:  10,
		CORSAllowedOrigins: "*",
	}
}

func TestEnvConfigValidateCORSWildcardWithCredentials(t *testing.T) {
	env := validEnv()
	env.CORSAllowCredentials = true

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error for wildcard origin with credentials")
	}
	if !strings.Contains(err.Error(), "CORS_ALLOW_CREDENTIALS") {
		t.Fatalf("error = %q", err)
	}
}

func TestEnvConfigValidateCORSExplicitOriginWithCredentials(t *testing.T) {
	env := validEnv()
	env.CORSAllowedOrigins = "https://app.example.com"
	env.CORSAllowCredentials = true

	if err := env.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvConfigValidateProdRequiresOrigins(t *testing.T) {
	env := validEnv()
	env.Stage = "prod"
	env.CORSAllowedOrigins = ""

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error for prod without origins")
	}
	if !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
		t.Fatalf("error = %q", err)
	}
}

func TestEnvConfigValidateAsynqmonDevDefaults(t *testing.T) {
	env := validEnv()
	env.EnableAsynqmon = true
	env.BasicAuthUser = "dev"
	env.BasicAuthPass = "dev"

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error for dev/dev asynqmon credentials")
	}
	if !strings.Contains(err.Error(), "BASIC_AUTH") {
		t.Fatalf("error = %q", err)
	}
}

func TestEnvConfigValidateRateLimit(t *testing.T) {
	env := validEnv()
	env.RateLimitEnabled = true
	env.RateLimitRPS = 0

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error for zero rate limit rps")
	}
	if !strings.Contains(err.Error(), "RATE_LIMIT_RPS") {
		t.Fatalf("error = %q", err)
	}
}

func TestEnvConfigValidateIgnoresRateLimitWhenDisabled(t *testing.T) {
	env := validEnv()
	env.RateLimitEnabled = false
	env.RateLimitRPS = 0
	env.RateLimitBurst = 0

	if err := env.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvConfigValidateInvalidStage(t *testing.T) {
	env := validEnv()
	env.Stage = "production"

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error for unknown stage")
	}
	if !strings.Contains(err.Error(), "STAGE") {
		t.Fatalf("error = %q", err)
	}
}
