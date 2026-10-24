package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"{{ cookiecutter.go_module }}/internal/testutil"
)

func newTestReadiness() *Readiness {
	return NewReadiness({% if cookiecutter.use_db == "yes" %}nil{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}, nil{% endif %}{% elif cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}nil{% endif %})
}

func TestReadinessOkWithoutChecks(t *testing.T) {
	readiness := newTestReadiness()
	ctx, rec := testutil.NewContext(http.MethodGet, "/readyz", nil)

	readiness.Handler()(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestReadinessReportsFailingDependency(t *testing.T) {
	readiness := newTestReadiness()
	readiness.Add("db", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	ctx, rec := testutil.NewContext(http.MethodGet, "/readyz", nil)
	readiness.Handler()(ctx)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"db"`) {
		t.Errorf("body must name the failing check: %s", body)
	}
	if !strings.Contains(body, "connection refused") {
		t.Errorf("body must report the reason: %s", body)
	}
}

func TestReadinessPassesContextToChecks(t *testing.T) {
	readiness := newTestReadiness()
	readiness.Add("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	// The handler applies its own deadline, so a check that waits for the
	// context to expire must end in 503 instead of blocking forever.
	deadline := time.After(15 * time.Second)
	done := make(chan int, 1)
	go func() {
		ctx, rec := testutil.NewContext(http.MethodGet, "/readyz", nil)
		readiness.Handler()(ctx)
		done <- rec.Code
	}()

	select {
	case code := <-done:
		if code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503 when a check times out", code)
		}
	case <-deadline:
		t.Fatal("el handler no aplicó el timeout de readiness")
	}
}
