package main

import (
	"context"
	"net"
	"testing"
	"time"

	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/core/asynctasks/tasks"
	"{{ cookiecutter.go_module }}/internal/testutil"

	"github.com/hibiken/asynq"
	"go.uber.org/fx"
)

// captureLifecycle records the hooks the worker registers so that the test can
// drive them the way fx would.
type captureLifecycle struct {
	hooks []fx.Hook
}

func (l *captureLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func workerTestConfig() *configs.Config {
	cfg := testutil.NewConfig()
	cfg.RedisHost = "127.0.0.1"
	cfg.RedisPort = "6379"
	return cfg
}

func TestNewAsynqServer(t *testing.T) {
	cfg := workerTestConfig()
	cfg.WorkerConcurrency = 10

	srv := newAsynqServer(cfg)
	if srv == nil {
		t.Fatal("newAsynqServer returned nil")
	}
}

func TestServeMuxRegistersEveryTask(t *testing.T) {
	mux := newServeMux(testutil.NewLogger())

	for _, taskType := range []string{tasks.TypeEmailDelivery, tasks.TypeDebugTask} {
		handler, _ := mux.Handler(asynq.NewTask(taskType, nil))
		if handler == nil {
			t.Errorf("no handler registered for %s", taskType)
		}
	}
}

func reachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestWorkerShutdownIsBoundedByContext(t *testing.T) {
	cfg := workerTestConfig()
	srv := newAsynqServer(cfg)
	mux := newServeMux(testutil.NewLogger())

	lifecycle := &captureLifecycle{}
	runWorker(lifecycle, srv, mux, cfg, testutil.NewLogger())

	if len(lifecycle.hooks) != 1 {
		t.Fatalf("hooks = %d, want 1", len(lifecycle.hooks))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hook := lifecycle.hooks[0]
	if hook.OnStart == nil || hook.OnStop == nil {
		t.Fatal("the worker must register OnStart and OnStop")
	}

	// Stopping a worker that never started must not block.
	if err := hook.OnStop(ctx); err != nil {
		t.Fatalf("OnStop returned error: %v", err)
	}

	if !reachable(cfg.RedisHost + ":" + cfg.RedisPort) {
		t.Log("redis not available: only registration and shutdown are checked")
		return
	}

	if err := hook.OnStart(ctx); err != nil {
		t.Fatalf("OnStart returned error: %v", err)
	}
	if err := hook.OnStop(ctx); err != nil {
		t.Fatalf("OnStop returned error: %v", err)
	}
	// Stopping again must neither block nor fail.
	if err := hook.OnStop(ctx); err != nil {
		t.Fatalf("segundo OnStop returned error: %v", err)
	}
}
