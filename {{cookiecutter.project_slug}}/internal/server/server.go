package server

import (
	"errors"
	"net/http"
	"time"

	"{{ cookiecutter.go_module }}/configs"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Router     *gin.Engine
	HttpServer *http.Server
}

func NewServer(router *gin.Engine) *Server {
	return &Server{
		Router: router,
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

func (server *Server) Run() {
	logger := configs.GetLogger()
	logger.Info("Listening to address: 0.0.0.0:" + configs.Env.Port)
	server.HttpServer = newHTTPServer(":"+configs.Env.Port, server.Router)
	if err := server.HttpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "err", err)
	}
}
