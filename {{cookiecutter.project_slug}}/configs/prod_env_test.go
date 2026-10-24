package configs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadEnvFile(t *testing.T, name string) map[string]string {
	t.Helper()

	path := filepath.Join("..", "env", name)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", path, err)
	}
	defer func() { _ = file.Close() }()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error leyendo %s: %v", path, err)
	}
	return values
}

func TestProdEnvDisablesExposedFeatures(t *testing.T) {
	prod := loadEnvFile(t, "prod.env")

	for _, key := range []string{"ENABLE_SWAGGER", "ENABLE_ASYNQMON", "ENABLE_PPROF", "ENABLE_LOG_REQUEST_DETAIL"} {
		if got := prod[key]; got != "false" {
			t.Errorf("%s = %q en prod.env, se espera false", key, got)
		}
	}
	if got := prod["ENABLE_HSTS"]; got != "true" {
		t.Errorf("ENABLE_HSTS = %q en prod.env, se espera true", got)
	}
}

func TestProdEnvHasNoEmbeddedSecrets(t *testing.T) {
	prod := loadEnvFile(t, "prod.env")

	for _, key := range []string{"JWT_SECRET", "BASIC_AUTH_PASS", "DB_PASS", "REDIS_PASS", "SENTRY_DSN"} {
		if value := prod[key]; value != "" {
			t.Errorf("%s no debe llevar un valor por defecto en prod.env, se obtuvo %q", key, value)
		}
	}
	if user := prod["BASIC_AUTH_USER"]; user == "dev" {
		t.Error("BASIC_AUTH_USER no puede ser el valor por defecto dev en prod.env")
	}
}

func TestProdEnvIsRejectedWithoutCorsOrigins(t *testing.T) {
	prod := loadEnvFile(t, "prod.env")

	env := validEnv()
	env.Stage = prod["STAGE"]
	env.CORSAllowedOrigins = prod["CORS_ALLOWED_ORIGINS"]
	env.CORSAllowCredentials = prod["CORS_ALLOW_CREDENTIALS"] == "true"

	err := env.Validate()
	if err == nil {
		t.Fatal("una configuración en prod sin CORS_ALLOWED_ORIGINS debe ser rechazada (fallo cerrado)")
	}
	if !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
		t.Fatalf("error = %q", err)
	}
}
