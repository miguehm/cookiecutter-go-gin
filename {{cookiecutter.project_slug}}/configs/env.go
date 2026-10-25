// Package configs loads, validates and exposes the application configuration
// together with the client constructors that consume it.
package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the application configuration, populated from environment
// variables (and the stage env file) through viper.
type Config struct {
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

	JwtSecret     string        `mapstructure:"JWT_SECRET"`
	JwtIssuer     string        `mapstructure:"JWT_ISSUER"`
	JwtAudience   string        `mapstructure:"JWT_AUDIENCE"`
	JwtExpiration time.Duration `mapstructure:"JWT_EXPIRATION"`
	JwtLeeway     time.Duration `mapstructure:"JWT_LEEWAY"`

	BasicAuthUser string `mapstructure:"BASIC_AUTH_USER"`
	BasicAuthPass string `mapstructure:"BASIC_AUTH_PASS"`

	RedisHost      string `mapstructure:"REDIS_HOST"`
	RedisPort      string `mapstructure:"REDIS_PORT"`
	RedisPass      string `mapstructure:"REDIS_PASS"`
	RedisDB        int    `mapstructure:"REDIS_DB"`
	RedisEnableSsl bool   `mapstructure:"REDIS_ENABLE_SSL"`
	RedisTaskDB    int    `mapstructure:"REDIS_TASK_DB"`

	EnableSwagger        bool   `mapstructure:"ENABLE_SWAGGER"`
	EnableAsynqmon       bool   `mapstructure:"ENABLE_ASYNQMON"`
	EnableHSTS           bool   `mapstructure:"ENABLE_HSTS"`
	TrustedProxies       string `mapstructure:"TRUSTED_PROXIES"`
	CORSAllowedOrigins   string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	CORSAllowCredentials bool   `mapstructure:"CORS_ALLOW_CREDENTIALS"`

	MaxBodyBytes     int64   `mapstructure:"MAX_BODY_BYTES"`
	RateLimitEnabled bool    `mapstructure:"RATE_LIMIT_ENABLED"`
	RateLimitRPS     float64 `mapstructure:"RATE_LIMIT_RPS"`
	RateLimitBurst   int     `mapstructure:"RATE_LIMIT_BURST"`

	WorkerConcurrency int `mapstructure:"WORKER_CONCURRENCY"`
}

var validStages = []string{"dev", "testing", "staging", "prod"}

// LoadOptions controls where the configuration is read from. Every field is
// optional: STAGE and ENV_FILE/ENV_DIR environment variables are used as a
// fallback, and the loader falls back to searching env/<stage>.env upwards
// from the working directory and from the executable directory.
type LoadOptions struct {
	Stage   string
	EnvFile string
	EnvDir  string
}

// NewConfig loads, applies defaults and validates the configuration.
func NewConfig() (*Config, error) {
	cfg, err := Load(LoadOptions{})
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Load reads env/<stage>.env (or ENV_FILE/ENV_DIR) into a fresh Config.
// Environment variables take precedence over the file. No package level state
// is mutated.
func Load(opts LoadOptions) (*Config, error) {
	stage, explicit := opts.Stage, opts.Stage != ""
	if !explicit {
		if env, ok := os.LookupEnv("STAGE"); ok {
			stage, explicit = env, true
		}
	}
	if stage == "" {
		stage = "dev"
	}
	if !slices.Contains(validStages, stage) {
		return nil, fmt.Errorf("invalid STAGE %q, must be one of %s", stage, strings.Join(validStages, ", "))
	}

	envFile, _, err := resolveEnvFile(opts, stage)
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(envFile)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("can't read env file %s: %w", envFile, err)
	}
	bindEnvKeys(v)

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("environment can't be loaded: %w", err)
	}

	if explicit {
		cfg.Stage = stage
	}
	if cfg.Stage == "" {
		cfg.Stage = "dev"
	}
	if !slices.Contains(validStages, cfg.Stage) {
		return nil, fmt.Errorf("invalid STAGE %q, must be one of %s", cfg.Stage, strings.Join(validStages, ", "))
	}

	cfg.ApplyDefaults()
	return cfg, nil
}

// bindEnvKeys registers every mapstructure field so that environment
// variables override the values coming from the file, and so that every key
// takes part in the unmarshal even when the file omits it.
func bindEnvKeys(v *viper.Viper) {
	typ := reflect.TypeOf(Config{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		key := field.Tag.Get("mapstructure")
		if key == "" || key == "-" {
			continue
		}
		v.SetDefault(key, reflect.Zero(field.Type).Interface())
		_ = v.BindEnv(key)
	}
}

// resolveEnvFile returns the env file to read. When it is not given
// explicitly it searches env/<stage>.env from the working directory upwards
// and then from the executable directory upwards, which keeps the loader
// working from any cwd (tests, go run, installed binaries, containers).
func resolveEnvFile(opts LoadOptions, stage string) (string, []string, error) {
	name := stage + ".env"

	if path := opts.EnvFile; path != "" {
		return path, nil, nil
	}
	if path := os.Getenv("ENV_FILE"); path != "" {
		return path, nil, nil
	}

	dir := opts.EnvDir
	if dir == "" {
		dir = os.Getenv("ENV_DIR")
	}
	if dir != "" {
		path := filepath.Join(dir, name)
		if isFile(path) {
			return path, nil, nil
		}
		return "", nil, fmt.Errorf("can't find env file %s (ENV_DIR=%s)", path, dir)
	}

	var tried []string
	if wd, err := os.Getwd(); err == nil {
		for dir := wd; ; {
			path := filepath.Join(dir, "env", name)
			tried = append(tried, path)
			if isFile(path) {
				return path, tried, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		for dir := filepath.Dir(exe); ; {
			path := filepath.Join(dir, "env", name)
			tried = append(tried, path)
			if isFile(path) {
				return path, tried, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", tried, fmt.Errorf("can't find env file %s (searched: %s)", name, strings.Join(tried, ", "))
}

func isFile(path string) bool {
	// #nosec G703 -- solo se comprueba la existencia de un fichero de
	// configuración construido a partir del cwd o del binario, nunca se lee.
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ApplyDefaults fills the zero values with the defaults of the template. It
// runs after the file and the environment have been merged.
func (c *Config) ApplyDefaults() {
	if c.GinMode == "" {
		c.GinMode = "debug"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.JwtIssuer == "" {
		c.JwtIssuer = "go-gin-template"
	}
	if c.JwtAudience == "" {
		c.JwtAudience = "go-gin-template"
	}
	if c.JwtExpiration <= 0 {
		c.JwtExpiration = 24 * time.Hour
	}
	if c.JwtLeeway <= 0 {
		c.JwtLeeway = 30 * time.Second
	}
	if c.CORSAllowedOrigins == "" && c.Stage != "prod" {
		c.CORSAllowedOrigins = "*"
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = 1 << 20
	}
	if c.RateLimitRPS == 0 {
		c.RateLimitRPS = 5
	}
	if c.RateLimitBurst == 0 {
		c.RateLimitBurst = 10
	}
	if c.WorkerConcurrency <= 0 {
		c.WorkerConcurrency = 10
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

// Validate checks that the required settings are present and well formed,
// returning every problem found at once.
func (c *Config) Validate() error {
	var errs []string
	if c.Stage == "" {
		errs = append(errs, "STAGE is required")
	} else if !slices.Contains(validStages, c.Stage) {
		errs = append(errs, fmt.Sprintf("STAGE must be one of %s", strings.Join(validStages, ",")))
	}

	require := func(name, value string) {
		if value == "" {
			errs = append(errs, fmt.Sprintf("%s is required", name))
		}
	}

	require("PORT", c.Port)
	require("GIN_MODE", c.GinMode)
	require("JWT_SECRET", c.JwtSecret)
	if len(c.JwtSecret) < 32 {
		errs = append(errs, "JWT_SECRET must be at least 32 characters")
	}
	if c.JwtExpiration <= 0 {
		errs = append(errs, "JWT_EXPIRATION must be a positive duration")
	}
	if c.JwtLeeway < 0 {
		errs = append(errs, "JWT_LEEWAY must not be negative")
	}

	modes := []string{"debug", "release", "test"}
	if c.GinMode != "" && !slices.Contains(modes, c.GinMode) {
		errs = append(errs, fmt.Sprintf("GIN_MODE must be one of %s", strings.Join(modes, ",")))
	}

	if c.Port != "" {
		if p, err := strconv.Atoi(c.Port); err != nil || p < 1 || p > 65535 {
			errs = append(errs, "PORT must be between 1 and 65535")
		}
	}

	if c.DBHost != "" {
		require("DB_PORT", c.DBPort)
		require("DB_USER", c.DBUser)
		require("DB_NAME", c.DBName)
		require("DB_SSL_MODE", c.DBSslMode)
	}

	if c.RedisHost != "" {
		require("REDIS_PORT", c.RedisPort)
	}

	if c.EnableAsynqmon {
		require("BASIC_AUTH_USER", c.BasicAuthUser)
		require("BASIC_AUTH_PASS", c.BasicAuthPass)
		if c.BasicAuthUser == "dev" && c.BasicAuthPass == "dev" {
			errs = append(errs, "BASIC_AUTH_USER/PASS must not use dev/dev defaults when asynqmon is enabled")
		}
	}

	origins := splitOrigins(c.CORSAllowedOrigins)
	if c.Stage == "prod" && len(origins) == 0 {
		errs = append(errs, "CORS_ALLOWED_ORIGINS is required in prod")
	}
	if c.CORSAllowCredentials && slices.Contains(origins, "*") {
		errs = append(errs, "CORS_ALLOW_CREDENTIALS cannot be true with wildcard origin")
	}

	if c.MaxBodyBytes <= 0 {
		errs = append(errs, "MAX_BODY_BYTES must be positive")
	}
	if c.RateLimitEnabled {
		if c.RateLimitRPS <= 0 {
			errs = append(errs, "RATE_LIMIT_RPS must be positive")
		}
		if c.RateLimitBurst <= 0 {
			errs = append(errs, "RATE_LIMIT_BURST must be positive")
		}
	}
	if c.WorkerConcurrency <= 0 {
		errs = append(errs, "WORKER_CONCURRENCY must be positive")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid configuration: %s", strings.Join(errs, "; "))
	}

	return nil
}
