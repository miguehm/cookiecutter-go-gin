package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"{{ cookiecutter.go_module }}/internal/testutil"
)

func TestNewHTTPServerTimeouts(t *testing.T) {
	handler := http.NewServeMux()
	srv := newHTTPServer(":8080", handler)

	if srv.Addr != ":8080" {
		t.Errorf("Addr = %q", srv.Addr)
	}
	if srv.Handler != handler {
		t.Error("Handler no coincide")
	}

	checks := map[string]struct {
		got, want time.Duration
	}{
		"ReadHeaderTimeout": {srv.ReadHeaderTimeout, 5 * time.Second},
		"ReadTimeout":       {srv.ReadTimeout, 30 * time.Second},
		"WriteTimeout":      {srv.WriteTimeout, 30 * time.Second},
		"IdleTimeout":       {srv.IdleTimeout, 120 * time.Second},
	}
	for name, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, se espera %v", name, check.got, check.want)
		}
		if check.got <= 0 {
			t.Errorf("%s no debe ser ilimitado", name)
		}
	}

	if srv.MaxHeaderBytes != 1<<20 {
		t.Errorf("MaxHeaderBytes = %d, se espera %d", srv.MaxHeaderBytes, 1<<20)
	}
}

func TestServerStartFailsWhenPortIsBusy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo reservar un puerto: %v", err)
	}
	defer func() { _ = listener.Close() }()

	cfg := testutil.NewConfig()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	cfg.Port = port

	server := NewServer(cfg, testutil.NewLogger(), newTestRouter(t))
	if err := server.Start(); err == nil {
		t.Fatal("Start debía fallar con el puerto ocupado")
	}
}

func TestServerShutdownClosesConnections(t *testing.T) {
	cfg := testutil.NewConfig()
	cfg.Port = "0"

	server := NewServer(cfg, testutil.NewLogger(), newTestRouter(t))
	if err := server.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if server.Addr() == "" {
		t.Fatal("Addr debe devolver la dirección real tras Start")
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve() }()

	_, port, err := net.SplitHostPort(server.Addr())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve returned %v after a clean shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve no terminó tras el apagado")
	}
}
