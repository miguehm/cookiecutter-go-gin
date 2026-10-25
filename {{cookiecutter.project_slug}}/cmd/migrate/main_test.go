package main

import (
	"testing"

	"{{ cookiecutter.go_module }}/configs"
)

func dbConfig() configs.Config {
	return configs.Config{
		DBUser:    "app",
		DBPass:    "p@ss",
		DBHost:    "db.example.com",
		DBPort:    "5432",
		DBName:    "mydb",
		DBSslMode: "require",
	}
}

func TestBuildDatabaseURL(t *testing.T) {
	cfg := dbConfig()

	got := buildDatabaseURL(&cfg)

	want := "postgresql://app:p%40ss@db.example.com:5432/mydb?sslmode=require"
	if got != want {
		t.Fatalf("buildDatabaseURL = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLWithoutSSLMode(t *testing.T) {
	cfg := dbConfig()
	cfg.DBPass = "secret"
	cfg.DBHost = "127.0.0.1"
	cfg.DBName = "app"
	cfg.DBSslMode = ""

	got := buildDatabaseURL(&cfg)

	want := "postgresql://app:secret@127.0.0.1:5432/app"
	if got != want {
		t.Fatalf("buildDatabaseURL = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLEscapesIPv6(t *testing.T) {
	cfg := dbConfig()
	cfg.DBPass = "secret"
	cfg.DBHost = "::1"
	cfg.DBName = "app"
	cfg.DBSslMode = "disable"

	got := buildDatabaseURL(&cfg)

	want := "postgresql://app:secret@[::1]:5432/app?sslmode=disable"
	if got != want {
		t.Fatalf("buildDatabaseURL = %q, want %q", got, want)
	}
}
