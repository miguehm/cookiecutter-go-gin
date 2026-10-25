package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"{{ cookiecutter.go_module }}/configs"

	"github.com/gin-gonic/gin"
)

// Server owns the listener and the http.Server behind it.
type Server struct {
	cfg        *configs.Config
	logger     *slog.Logger
	router     *gin.Engine
	httpServer *http.Server
	listener   net.Listener
}

// NewServer returns a Server bound to the given configuration, logger and
// router; call Start before Serve.
func NewServer(cfg *configs.Config, logger *slog.Logger, router *gin.Engine) *Server {
	return &Server{
		cfg:    cfg,
		logger: logger,
		router: router,
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		Handler:           handler,
	}
}

// Start reserves the TCP port so that a failure (port already in use, missing
// permission) is reported before the application reports itself as running.
func (s *Server) Start() error {
	// #nosec G102 -- el bind deliberado es ":<puerto>" para que la imagen
	// funcione dentro del contenedor y del orquestador.
	listener, err := net.Listen("tcp", ":"+s.cfg.Port)
	if err != nil {
		return fmt.Errorf("can't listen on port %s: %w", s.cfg.Port, err)
	}
	s.listener = listener
	s.httpServer = newHTTPServer(listener.Addr().String(), s.router)
	s.logger.Info("listening on " + listener.Addr().String())
	return nil
}

// Addr returns the address the server is listening on, which is only known
// after Start (useful when the port is 0).
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Serve blocks until the server stops. A clean shutdown is not reported as an
// error.
func (s *Server) Serve() error {
	if s.httpServer == nil || s.listener == nil {
		return fmt.Errorf("server has not been started")
	}
	err := s.httpServer.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown drains in-flight requests up to the deadline given by the caller.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
