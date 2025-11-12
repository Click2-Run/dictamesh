// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Handler handles WebSocket connections
type Handler struct {
	hub *Hub

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// Authentication function
	authFunc AuthFunc

	// Logger
	logger *zap.Logger
}

// AuthFunc authenticates a WebSocket connection and returns the user ID
type AuthFunc func(r *http.Request) (string, error)

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, authFunc AuthFunc, logger *zap.Logger) *Handler {
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking for production
				// For now, allow all origins for development
				return true
			},
		},
		authFunc: authFunc,
		logger:   logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate the request
	userID, err := h.authFunc(r)
	if err != nil {
		h.logger.Error("authentication failed", zap.Error(err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade connection to WebSocket
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	// Generate connection ID
	connectionID, err := generateConnectionID()
	if err != nil {
		h.logger.Error("failed to generate connection ID", zap.Error(err))
		ws.Close()
		return
	}

	// Create connection
	conn := NewConnection(connectionID, userID, ws, h.hub, h.logger)

	// Register connection with hub
	h.hub.register <- conn

	// Start connection pumps in goroutines
	go conn.WritePump()
	go conn.ReadPump()

	h.logger.Info("websocket connection established",
		zap.String("connection_id", connectionID),
		zap.String("user_id", userID),
		zap.String("remote_addr", r.RemoteAddr),
	)
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// DefaultAuthFunc provides a simple token-based authentication
// In production, this should be replaced with proper JWT or session validation
func DefaultAuthFunc(r *http.Request) (string, error) {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Check query parameter as fallback
		token := r.URL.Query().Get("token")
		if token == "" {
			return "", ErrUnauthorized
		}
		authHeader = "Bearer " + token
	}

	// Extract token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidToken
	}

	token := parts[1]

	// TODO: Implement proper token validation
	// For now, we'll use the token as the user ID
	// In production, validate the JWT/session and extract the user ID
	if token == "" {
		return "", ErrInvalidToken
	}

	// For development/testing, extract user ID from token
	// In production, this should validate and decode a JWT
	userID := token

	return userID, nil
}

// SSRCompatibleHandler wraps the WebSocket handler to be SSR-compatible
// It checks if the request is a WebSocket upgrade request
func SSRCompatibleHandler(h *Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a WebSocket upgrade request
		if !isWebSocketUpgrade(r) {
			// Return a friendly message for non-WebSocket requests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUpgradeRequired)
			w.Write([]byte(`{"error":"WebSocket connection required","message":"Please use a WebSocket client to connect to this endpoint"}`))
			return
		}

		// Handle WebSocket connection
		h.ServeHTTP(w, r)
	}
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade request
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// HealthCheckHandler returns a handler for health checks
func HealthCheckHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := hub.GetMetrics()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Simple health response
		response := map[string]interface{}{
			"status":              "healthy",
			"active_connections":  metrics.ActiveConnections,
			"total_connections":   metrics.TotalConnections,
			"connections_created": metrics.ConnectionsCreated,
			"connections_dropped": metrics.ConnectionsDropped,
			"total_messages":      metrics.TotalMessages,
			"total_broadcasts":    metrics.TotalBroadcasts,
			"last_message_time":   metrics.LastMessageTime,
		}

		// Simple JSON encoding (avoiding dependency)
		w.Write([]byte(`{`))
		first := true
		for k, v := range response {
			if !first {
				w.Write([]byte(`,`))
			}
			first = false
			w.Write([]byte(`"` + k + `":`))
			switch val := v.(type) {
			case string:
				w.Write([]byte(`"` + val + `"`))
			case int64:
				w.Write([]byte(string(rune(val))))
			default:
				w.Write([]byte(`null`))
			}
		}
		w.Write([]byte(`}`))
	}
}

// MetricsHandler returns a handler for Prometheus-style metrics
func MetricsHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := hub.GetMetrics()

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		// Prometheus-style metrics
		w.Write([]byte("# HELP dictamesh_websocket_connections_active Number of active WebSocket connections\n"))
		w.Write([]byte("# TYPE dictamesh_websocket_connections_active gauge\n"))
		w.Write([]byte("dictamesh_websocket_connections_active " + string(rune(metrics.ActiveConnections)) + "\n\n"))

		w.Write([]byte("# HELP dictamesh_websocket_connections_total Total number of WebSocket connections created\n"))
		w.Write([]byte("# TYPE dictamesh_websocket_connections_total counter\n"))
		w.Write([]byte("dictamesh_websocket_connections_total " + string(rune(metrics.ConnectionsCreated)) + "\n\n"))

		w.Write([]byte("# HELP dictamesh_websocket_messages_total Total number of messages sent\n"))
		w.Write([]byte("# TYPE dictamesh_websocket_messages_total counter\n"))
		w.Write([]byte("dictamesh_websocket_messages_total " + string(rune(metrics.TotalMessages)) + "\n\n"))

		w.Write([]byte("# HELP dictamesh_websocket_broadcasts_total Total number of broadcasts\n"))
		w.Write([]byte("# TYPE dictamesh_websocket_broadcasts_total counter\n"))
		w.Write([]byte("dictamesh_websocket_broadcasts_total " + string(rune(metrics.TotalBroadcasts)) + "\n\n"))
	}
}
