package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validEnv() Config {
	return Config{
		Stage:             "dev",
		Port:              "8080",
		GinMode:           "debug",
		JwtSecret:         "dev-secret-change-me-0123456789abcdef",
		JwtExpiration:     24 * time.Hour,
		JwtLeeway:         30 * time.Second,
		MaxBodyBytes:      1 << 20,
		RateLimitRPS:      5,
		RateLimitBurst:    10,
		WorkerConcurrency: 10,
		CORSAllowedOrigins: "*",
	}
}

func TestConfigValidate(t *testing.T) {
	valid := validEnv()
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

func TestConfigValidateMissing(t *testing.T) {
	env := Config{
		Stage:             "dev",
		DBHost:            "127.0.0.1",
		DBPort:            "5432",
		RedisHost:         "127.0.0.1",
		MaxBodyBytes:      1 << 20,
		RateLimitRPS:      5,
		RateLimitBurst:    10,
		WorkerConcurrency: 10,
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

func TestConfigValidateJWTSecretLength(t *testing.T) {
	env := validEnv()
	env.JwtSecret = "short"

	if err := env.Validate(); err == nil {
		t.Fatal("expected error for short JWT secret")
	}
}

func TestConfigValidateCORSWildcardWithCredentials(t *testing.T) {
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

func TestConfigValidateCORSExplicitOriginWithCredentials(t *testing.T) {
	env := validEnv()
	env.CORSAllowedOrigins = "https://app.example.com"
	env.CORSAllowCredentials = true

	if err := env.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigValidateProdRequiresOrigins(t *testing.T) {
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

func TestConfigValidateAsynqmonDevDefaults(t *testing.T) {
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

func TestConfigValidateRateLimit(t *testing.T) {
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

func TestConfigValidateIgnoresRateLimitWhenDisabled(t *testing.T) {
	env := validEnv()
	env.RateLimitEnabled = false
	env.RateLimitRPS = 0
	env.RateLimitBurst = 0

	if err := env.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigValidateInvalidStage(t *testing.T) {
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

func TestConfigValidateWorkerConcurrency(t *testing.T) {
	env := validEnv()
	env.WorkerConcurrency = 0

	err := env.Validate()
	if err == nil {
		t.Fatal("expected error for zero worker concurrency")
	}
	if !strings.Contains(err.Error(), "WORKER_CONCURRENCY") {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadReadsFileFromParentDirectory(t *testing.T) {
	t.Chdir(filepath.Join("..", "internal"))

	cfg, err := Load(LoadOptions{Stage: "testing"})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Stage != "testing" {
		t.Errorf("Stage = %q, want testing", cfg.Stage)
	}
	if cfg.Port == "" {
		t.Error("Port must be loaded from env/testing.env")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("loaded configuration is not valid: %v", err)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	t.Setenv("PORT", "9099")

	cfg, err := Load(LoadOptions{Stage: "testing"})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != "9099" {
		t.Errorf("Port = %q, want 9099 (environment must win over the file)", cfg.Port)
	}
}

func TestLoadUsesExplicitEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.env")
	content := "STAGE=testing\nPORT=7070\nGIN_MODE=test\nJWT_SECRET=custom-secret-0123456789abcdef0123456789\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Chdir(t.TempDir())

	cfg, err := Load(LoadOptions{EnvFile: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != "7070" {
		t.Errorf("Port = %q, want 7070", cfg.Port)
	}
	if cfg.JwtSecret == "" {
		t.Error("JWT_SECRET must be read from the explicit env file")
	}
}

func TestLoadFailsOutsideTheProject(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := Load(LoadOptions{Stage: "testing", EnvFile: ""})
	if err == nil {
		t.Fatal("expected an error when no env file can be found")
	}
	if !strings.Contains(err.Error(), "can't find env file") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRejectsInvalidStage(t *testing.T) {
	if _, err := Load(LoadOptions{Stage: "production"}); err == nil {
		t.Fatal("expected an error for an unknown stage")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "minimal.env")
	payload := "STAGE=testing\nPORT=8080\nJWT_SECRET=" + strings.Repeat("x", 32) + "\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, err := Load(LoadOptions{EnvFile: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	checks := map[string]struct{ got, want any }{
		"GinMode":           {cfg.GinMode, "debug"},
		"JwtExpiration":     {cfg.JwtExpiration, 24 * time.Hour},
		"JwtLeeway":         {cfg.JwtLeeway, 30 * time.Second},
		"MaxBodyBytes":      {cfg.MaxBodyBytes, int64(1 << 20)},
		"WorkerConcurrency": {cfg.WorkerConcurrency, 10},
	}
	for name, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", name, check.got, check.want)
		}
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("configuration with defaults must be valid: %v", err)
	}
}
