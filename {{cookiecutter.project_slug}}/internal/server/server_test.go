package server

import (
	"net/http"
	"testing"
	"time"
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
