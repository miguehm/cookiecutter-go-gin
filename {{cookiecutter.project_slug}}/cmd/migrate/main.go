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

// buildDatabaseURL assembles the connection URL from the configuration.
// net/url and net.JoinHostPort do not fail, hence no error is returned.
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
	// #nosec G204 -- fixed argv built internally, no shell involved
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
