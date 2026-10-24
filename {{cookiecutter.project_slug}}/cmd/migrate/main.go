package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"

	"{{ cookiecutter.go_module }}/configs"
)

func buildDatabaseURL(cfg *configs.Config) (string, error) {
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
	return u.String(), nil
}

func main() {
	cfg, err := configs.NewConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dsn, err := buildDatabaseURL(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	args := []string{"migrate", "apply", "--url", dsn, "--dir", "file://migrations"}
	// #nosec G204 -- argv fijo construido internamente, sin shell de por medio
	cmd := exec.Command("atlas", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
