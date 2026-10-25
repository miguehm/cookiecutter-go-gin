// Package testutil provides helpers shared by the unit tests: canned
// configuration, Gin contexts, fixtures and in-memory repositories.
package testutil

import (
	"io"
	"log/slog"

	"{{ cookiecutter.go_module }}/configs"
)

// NewConfig returns a valid configuration for tests. It reads no files and
// writes no package state, so tests can run in parallel.
func NewConfig() *configs.Config {
	cfg := &configs.Config{
		Stage:              "testing",
		GinMode:            "test",
		Port:               "8080",
		JwtSecret:          "testing-secret-change-me-0123456789abcdef",
		CORSAllowedOrigins: "*",
	}
	cfg.ApplyDefaults()
	return cfg
}

// NewLogger returns a logger that discards its output, for tests that only
// need a non-nil logger.
func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
