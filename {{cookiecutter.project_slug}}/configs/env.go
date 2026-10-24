package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type EnvConfig struct {
	Stage       string `mapstructure:"STAGE"`
	GinMode     string `mapstructure:"GIN_MODE"`
	EnablePPROF bool   `mapstructure:"ENABLE_PPROF"`
	Port        string `mapstructure:"PORT"`

	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPass     string `mapstructure:"DB_PASS"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSslMode  string `mapstructure:"DB_SSL_MODE"`
	DBLogLevel int    `mapstructure:"DB_LOG_LEVEL"`

	SentryDSN              string `mapstructure:"SENTRY_DSN"`
	EnableLogRequestDetail bool   `mapstructure:"ENABLE_LOG_REQUEST_DETAIL"`

	JwtSecret   string `mapstructure:"JWT_SECRET"`
	JwtIssuer   string `mapstructure:"JWT_ISSUER"`
	JwtAudience string `mapstructure:"JWT_AUDIENCE"`
	JwtExpiration string `mapstructure:"JWT_EXPIRATION"`
	JwtLeeway   string `mapstructure:"JWT_LEEWAY"`

	BasicAuthUser string `mapstructure:"BASIC_AUTH_USER"`
	BasicAuthPass string `mapstructure:"BASIC_AUTH_PASS"`

	RedisHost      string `mapstructure:"REDIS_HOST"`
	RedisPort      string `mapstructure:"REDIS_PORT"`
	RedisPass      string `mapstructure:"REDIS_PASS"`
	RedisDB        int    `mapstructure:"REDIS_DB"`
	RedisEnableSsl bool   `mapstructure:"REDIS_ENABLE_SSL"`
	RedisTaskDB    int    `mapstructure:"REDIS_TASK_DB"`

	EnableSwagger   bool   `mapstructure:"ENABLE_SWAGGER"`
	EnableAsynqmon  bool   `mapstructure:"ENABLE_ASYNQMON"`
	EnableHSTS      bool   `mapstructure:"ENABLE_HSTS"`
	TrustedProxies  string `mapstructure:"TRUSTED_PROXIES"`
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	CORSAllowCredentials bool `mapstructure:"CORS_ALLOW_CREDENTIALS"`

	MaxBodyBytes      int64   `mapstructure:"MAX_BODY_BYTES"`
	RateLimitEnabled  bool    `mapstructure:"RATE_LIMIT_ENABLED"`
	RateLimitRPS      float64 `mapstructure:"RATE_LIMIT_RPS"`
	RateLimitBurst    int     `mapstructure:"RATE_LIMIT_BURST"`
}

var Env = EnvConfig{}
var envLoadErr error

func envDefaults() {
	if Env.GinMode == "" {
		Env.GinMode = "debug"
	}
	if Env.Port == "" {
		Env.Port = "8080"
	}
	if Env.JwtIssuer == "" {
		Env.JwtIssuer = "go-gin-template"
	}
	if Env.JwtAudience == "" {
		Env.JwtAudience = "go-gin-template"
	}
	if Env.JwtExpiration == "" {
		Env.JwtExpiration = "24h"
	}
	if Env.JwtLeeway == "" {
		Env.JwtLeeway = "30s"
	}
	if Env.CORSAllowedOrigins == "" {
		if Env.Stage == "prod" {
			Env.CORSAllowedOrigins = ""
		} else {
			Env.CORSAllowedOrigins = "*"
		}
	}
	if Env.MaxBodyBytes == 0 {
		Env.MaxBodyBytes = 1 << 20
	}
	if Env.RateLimitRPS == 0 {
		Env.RateLimitRPS = 5
	}
	if Env.RateLimitBurst == 0 {
		Env.RateLimitBurst = 10
	}
}

func splitOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (e EnvConfig) Validate() error {
	var errs []string
	stages := []string{"dev", "testing", "staging", "prod"}
	if e.Stage == "" {
		errs = append(errs, "STAGE is required")
	} else if !slices.Contains(stages, e.Stage) {
		errs = append(errs, fmt.Sprintf("STAGE must be one of %s", strings.Join(stages, ", ")))
	}

	require := func(name, value string) {
		if value == "" {
			errs = append(errs, fmt.Sprintf("%s is required", name))
		}
	}

	require("PORT", e.Port)
	require("GIN_MODE", e.GinMode)
	require("JWT_SECRET", e.JwtSecret)
	if len(e.JwtSecret) < 32 {
		errs = append(errs, "JWT_SECRET must be at least 32 characters")
	}

	modes := []string{"debug", "release", "test"}
	if e.GinMode != "" && !slices.Contains(modes, e.GinMode) {
		errs = append(errs, fmt.Sprintf("GIN_MODE must be one of %s", strings.Join(modes, ", ")))
	}

	if e.Port != "" {
		if p, err := strconv.Atoi(e.Port); err != nil || p < 1 || p > 65535 {
			errs = append(errs, "PORT must be between 1 and 65535")
		}
	}

	if e.DBHost != "" {
		require("DB_PORT", e.DBPort)
		require("DB_USER", e.DBUser)
		require("DB_NAME", e.DBName)
		require("DB_SSL_MODE", e.DBSslMode)
	}

	if e.RedisHost != "" {
		require("REDIS_PORT", e.RedisPort)
	}

	if e.EnableAsynqmon {
		require("BASIC_AUTH_USER", e.BasicAuthUser)
		require("BASIC_AUTH_PASS", e.BasicAuthPass)
		if e.BasicAuthUser == "dev" && e.BasicAuthPass == "dev" {
			errs = append(errs, "BASIC_AUTH_USER/PASS must not use dev/dev defaults when asynqmon is enabled")
		}
	}

	origins := splitOrigins(e.CORSAllowedOrigins)
	if e.Stage == "prod" {
		if len(origins) == 0 {
			errs = append(errs, "CORS_ALLOWED_ORIGINS is required in prod")
		}
		if e.CORSAllowCredentials && slices.Contains(origins, "*") {
			errs = append(errs, "CORS_ALLOW_CREDENTIALS cannot be true with wildcard origin")
		}
	} else if e.CORSAllowCredentials && slices.Contains(origins, "*") {
		errs = append(errs, "CORS_ALLOW_CREDENTIALS cannot be true with wildcard origin")
	}

	if e.MaxBodyBytes <= 0 {
		errs = append(errs, "MAX_BODY_BYTES must be positive")
	}
	if e.RateLimitEnabled {
		if e.RateLimitRPS <= 0 {
			errs = append(errs, "RATE_LIMIT_RPS must be positive")
		}
		if e.RateLimitBurst <= 0 {
			errs = append(errs, "RATE_LIMIT_BURST must be positive")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid configuration: %s", strings.Join(errs, "; "))
	}

	return nil
}

func LoadEnv() error {
	return envLoadErr
}

func loadEnv() error {
	_, filename, _, _ := runtime.Caller(0)
	rootDir := filepath.Dir(filepath.Dir(filename))
	stage := os.Getenv("STAGE")

	if strings.HasSuffix(os.Args[0], ".test") {
		stage = "testing"
	}

	if stage == "" {
		stage = "dev"
	}

	Env.Stage = stage

	stages := []string{"dev", "testing", "staging", "prod"}
	if !slices.Contains(stages, stage) {
		return fmt.Errorf("invalid STAGE %q", stage)
	}

	envFile := filepath.Join(rootDir, "env", stage+".env")

	viper.SetConfigFile(envFile)

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("can't find env file %s: %w", envFile, err)
	}

	viper.AutomaticEnv()

	err = viper.Unmarshal(&Env)
	if err != nil {
		return fmt.Errorf("environment can't be loaded: %w", err)
	}

	Env.Stage = stage
	envDefaults()
	return nil
}

func init() {
	envLoadErr = loadEnv()
}
