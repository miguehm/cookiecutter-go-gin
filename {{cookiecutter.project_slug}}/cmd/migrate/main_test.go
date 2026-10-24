package main

import (
	"testing"

	"{{ cookiecutter.go_module }}/configs"
)

func TestBuildDatabaseURL(t *testing.T) {
	configs.Env.DBUser = "app"
	configs.Env.DBPass = "p@ss"
	configs.Env.DBHost = "db.example.com"
	configs.Env.DBPort = "5432"
	configs.Env.DBName = "mydb"
	configs.Env.DBSslMode = "require"

	got, err := buildDatabaseURL()
	if err != nil {
		t.Fatalf("buildDatabaseURL returned error: %v", err)
	}

	want := "postgresql://app:p%40ss@db.example.com:5432/mydb?sslmode=require"
	if got != want {
		t.Fatalf("buildDatabaseURL = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLWithoutSSLMode(t *testing.T) {
	configs.Env.DBUser = "app"
	configs.Env.DBPass = "secret"
	configs.Env.DBHost = "127.0.0.1"
	configs.Env.DBPort = "5432"
	configs.Env.DBName = "app"
	configs.Env.DBSslMode = ""

	got, err := buildDatabaseURL()
	if err != nil {
		t.Fatalf("buildDatabaseURL returned error: %v", err)
	}

	want := "postgresql://app:secret@127.0.0.1:5432/app"
	if got != want {
		t.Fatalf("buildDatabaseURL = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLEscapesIPv6(t *testing.T) {
	configs.Env.DBUser = "app"
	configs.Env.DBPass = "secret"
	configs.Env.DBHost = "::1"
	configs.Env.DBPort = "5432"
	configs.Env.DBName = "app"
	configs.Env.DBSslMode = "disable"

	got, err := buildDatabaseURL()
	if err != nil {
		t.Fatalf("buildDatabaseURL returned error: %v", err)
	}

	want := "postgresql://app:secret@[::1]:5432/app?sslmode=disable"
	if got != want {
		t.Fatalf("buildDatabaseURL = %q, want %q", got, want)
	}
}
