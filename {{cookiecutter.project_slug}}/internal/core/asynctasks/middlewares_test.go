package asynctasks

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
)

func captureLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	return logger, buf
}

func TestLoggingMiddlewareLogsSuccess(t *testing.T) {
	logger, buf := captureLogger()
	inner := asynq.HandlerFunc(func(context.Context, *asynq.Task) error { return nil })

	err := LoggingMiddleware(logger)(inner).ProcessTask(context.Background(), asynq.NewTask("demo:type", nil))
	if err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "task processed") || !strings.Contains(buf.String(), "demo:type") {
		t.Errorf("log = %s", buf.String())
	}
}

func TestLoggingMiddlewareLogsAndReturnsError(t *testing.T) {
	logger, buf := captureLogger()
	want := errors.New("task failed")
	inner := asynq.HandlerFunc(func(context.Context, *asynq.Task) error { return want })

	got := LoggingMiddleware(logger)(inner).ProcessTask(context.Background(), asynq.NewTask("demo:type", nil))
	if !errors.Is(got, want) {
		t.Fatalf("middleware changed the error: %v", got)
	}
	if !strings.Contains(buf.String(), "failed to process task") {
		t.Errorf("log = %s", buf.String())
	}
}
