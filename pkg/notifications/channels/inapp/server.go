// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server represents a WebSocket server
type Server struct {
	provider *Provider
	handler  *Handler
	config   *ServerConfig
	logger   *zap.Logger

	httpServer *http.Server
}

// ServerConfig holds server configuration
type ServerConfig struct {
	// Server address
	Address string
	Port    int

	// WebSocket path
	WebSocketPath string

	// Authentication
	AuthFunc AuthFunc

	// TLS configuration (optional)
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string

	// Timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// CORS (for SSR compatibility)
	AllowedOrigins []string
}

// NewServer creates a new WebSocket server
func NewServer(provider *Provider, config *ServerConfig, logger *zap.Logger) *Server {
	if config.AuthFunc == nil {
		config.AuthFunc = DefaultAuthFunc
	}

	if config.WebSocketPath == "" {
		config.WebSocketPath = "/ws"
	}

	handler := NewHandler(provider.GetHub(), config.AuthFunc, logger)

	return &Server{
		provider: provider,
		handler:  handler,
		config:   config,
		logger:   logger,
	}
}

// Start starts the WebSocket server
func (s *Server) Start(ctx context.Context) error {
	// Set up HTTP mux
	mux := http.NewServeMux()

	// WebSocket endpoint (SSR-compatible)
	mux.HandleFunc(s.config.WebSocketPath, SSRCompatibleHandler(s.handler))

	// Health check endpoint
	mux.HandleFunc("/health", HealthCheckHandler(s.provider.GetHub()))

	// Metrics endpoint
	mux.HandleFunc("/metrics", MetricsHandler(s.provider.GetHub()))

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Address, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// Start provider
	if err := s.provider.Start(ctx); err != nil {
		return fmt.Errorf("failed to start provider: %w", err)
	}

	s.logger.Info("starting websocket server",
		zap.String("address", addr),
		zap.String("ws_path", s.config.WebSocketPath),
		zap.Bool("tls_enabled", s.config.TLSEnabled),
	)

	// Start HTTP server
	go func() {
		var err error
		if s.config.TLSEnabled {
			err = s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			s.logger.Error("server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the WebSocket server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping websocket server")

	// Stop HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("failed to shutdown http server", zap.Error(err))
		}
	}

	// Stop provider
	if err := s.provider.Stop(ctx); err != nil {
		s.logger.Error("failed to stop provider", zap.Error(err))
		return err
	}

	return nil
}

// GetProvider returns the provider
func (s *Server) GetProvider() *Provider {
	return s.provider
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:        "0.0.0.0",
		Port:           8080,
		WebSocketPath:  "/ws",
		AuthFunc:       DefaultAuthFunc,
		TLSEnabled:     false,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		AllowedOrigins: []string{"*"},
	}
}
