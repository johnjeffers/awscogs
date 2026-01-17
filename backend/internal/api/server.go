package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/aws"
	"github.com/johnjeffers/infra-utilities/awscogs/backend/internal/config"
)

// Server is the HTTP server for the awscogs API
type Server struct {
	server    *http.Server
	config    *config.Config
	discovery *aws.Discovery
	logger    *slog.Logger
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, discovery *aws.Discovery, logger *slog.Logger) *Server {
	router := NewRouter(cfg, discovery)

	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 5 * time.Minute,
			IdleTimeout:  60 * time.Second,
		},
		config:    cfg,
		discovery: discovery,
		logger:    logger,
	}
}

// Start begins listening for requests
func (s *Server) Start() error {
	s.logger.Info("starting server", "port", s.config.Server.Port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.server.Shutdown(ctx)
}
