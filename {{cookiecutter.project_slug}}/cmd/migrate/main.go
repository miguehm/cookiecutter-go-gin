// Command migrate applies the Atlas migrations in migrations/ to the database
// described by the configuration.
package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"

	"{{ cookiecutter.go_module }}/configs"
)

// buildDatabaseURL ensambla la URL de conexión a partir de la configuración.
// net/url y net.JoinHostPort no fallan, por eso no devuelve error.
func buildDatabaseURL(cfg *configs.Config) string {
	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPass),
		Host:   net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path:   "/" + cfg.DBName,
	}
	q := u.Query()
	if cfg.DBSslMode != "" {
		q.Set("sslmode", cfg.DBSslMode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func main() {
	cfg, err := configs.NewConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dsn := buildDatabaseURL(cfg)
	args := []string{"migrate", "apply", "--url", dsn, "--dir", "file://migrations"}
	// #nosec G204 -- argv fijo construido internamente, sin shell de por medio
	cmd := exec.Command("atlas", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
