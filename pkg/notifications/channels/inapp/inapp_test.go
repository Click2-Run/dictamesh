// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestHub(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Test connection registration
	mockWs := &websocket.Conn{}
	conn := NewConnection("test-conn-1", "user-1", mockWs, hub, logger)

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	if hub.GetConnectionCount() != 1 {
		t.Errorf("expected 1 connection, got %d", hub.GetConnectionCount())
	}

	if hub.GetUserCount() != 1 {
		t.Errorf("expected 1 user, got %d", hub.GetUserCount())
	}

	// Test connection unregistration
	hub.unregister <- conn
	time.Sleep(100 * time.Millisecond)

	if hub.GetConnectionCount() != 0 {
		t.Errorf("expected 0 connections, got %d", hub.GetConnectionCount())
	}
}

func TestMessage(t *testing.T) {
	// Test notification message creation
	notif := &NotificationMessage{
		NotificationID: "notif-1",
		Subject:        "Test",
		Body:           "Test body",
		Priority:       "normal",
		CreatedAt:      time.Now(),
	}

	msg := NewNotificationMessage(notif)

	if msg.Type != MessageTypeNotification {
		t.Errorf("expected type %s, got %s", MessageTypeNotification, msg.Type)
	}

	if msg.Data["notification_id"] != "notif-1" {
		t.Errorf("expected notification_id notif-1, got %v", msg.Data["notification_id"])
	}

	// Test welcome message
	welcome := NewWelcomeMessage("user-1", "conn-1")
	if welcome.Type != MessageTypeWelcome {
		t.Errorf("expected type %s, got %s", MessageTypeWelcome, welcome.Type)
	}

	// Test error message
	errMsg := NewErrorMessage("ERR001", "Test error", "Details")
	if errMsg.Type != MessageTypeError {
		t.Errorf("expected type %s, got %s", MessageTypeError, errMsg.Type)
	}
	if errMsg.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestProvider(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := &Config{
		Enabled:           true,
		Transport:         "websocket",
		PersistenceDays:   30,
		MaxUnread:         100,
		WebSocketPath:     "/ws",
		WebSocketPingTime: 30 * time.Second,
	}

	provider := NewProvider(config, logger)
	ctx := context.Background()

	// Start provider
	err := provider.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start provider: %v", err)
	}
	defer provider.Stop(ctx)

	// Test health check
	err = provider.HealthCheck(ctx)
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}

	// Test channel name
	if provider.GetChannel() != "IN_APP" {
		t.Errorf("expected channel IN_APP, got %s", provider.GetChannel())
	}
}

func TestHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Simple auth function for testing
	authFunc := func(r *http.Request) (string, error) {
		token := r.URL.Query().Get("token")
		if token == "" {
			return "", ErrUnauthorized
		}
		return token, nil
	}

	handler := NewHandler(hub, authFunc, logger)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=test-user"

	// Connect WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	// Read welcome message
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Type != MessageTypeWelcome {
		t.Errorf("expected welcome message, got %s", msg.Type)
	}

	// Wait a bit for connection to register
	time.Sleep(100 * time.Millisecond)

	// Check hub stats
	if hub.GetConnectionCount() != 1 {
		t.Errorf("expected 1 connection in hub, got %d", hub.GetConnectionCount())
	}
}

func TestSSRCompatibility(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	authFunc := func(r *http.Request) (string, error) {
		return "test-user", nil
	}

	handler := NewHandler(hub, authFunc, logger)
	ssrHandler := SSRCompatibleHandler(handler)

	// Test regular HTTP request (non-WebSocket)
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	ssrHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUpgradeRequired {
		t.Errorf("expected status %d, got %d", http.StatusUpgradeRequired, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "WebSocket connection required") {
		t.Errorf("expected error message in body, got: %s", body)
	}
}

func TestHealthCheckHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)

	handler := HealthCheckHandler(hub)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %s", contentType)
	}
}

func TestConnectionStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	mockWs := &websocket.Conn{}

	conn := NewConnection("test-conn", "user-1", mockWs, hub, logger)

	stats := conn.GetStats()

	if stats.ConnectionID != "test-conn" {
		t.Errorf("expected connection_id test-conn, got %s", stats.ConnectionID)
	}

	if stats.UserID != "user-1" {
		t.Errorf("expected user_id user-1, got %s", stats.UserID)
	}

	if !stats.Connected {
		t.Error("expected connection to be connected")
	}
}

func TestHubMetrics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)

	metrics := hub.GetMetrics()

	if metrics == nil {
		t.Fatal("expected metrics to be non-nil")
	}

	if metrics.ActiveConnections != 0 {
		t.Errorf("expected 0 active connections, got %d", metrics.ActiveConnections)
	}
}

func TestSubscription(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create connection
	mockWs := &websocket.Conn{}
	conn := NewConnection("test-conn", "user-1", mockWs, hub, logger)
	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	// Subscribe to channel
	hub.Subscribe("test-conn", "alerts")
	hub.Subscribe("test-conn", "messages")

	// Unsubscribe from one channel
	hub.Unsubscribe("test-conn", "alerts")

	// Clean up
	hub.unregister <- conn
	time.Sleep(100 * time.Millisecond)
}
