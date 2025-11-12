// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/click2-run/dictamesh/pkg/notifications/channels/inapp"
	"go.uber.org/zap"
)

// Example demonstrates basic WebSocket server setup
func Example_basicServer() {
	// Create logger
	logger, _ := zap.NewDevelopment()

	// Configure provider
	providerConfig := &inapp.Config{
		Enabled:           true,
		Transport:         "websocket",
		PersistenceDays:   30,
		MaxUnread:         100,
		WebSocketPath:     "/ws",
		WebSocketPingTime: 30 * time.Second,
	}

	// Create provider
	provider := inapp.NewProvider(providerConfig, logger)

	// Configure server
	serverConfig := inapp.DefaultServerConfig()
	serverConfig.Port = 8080

	// Create and start server
	server := inapp.NewServer(provider, serverConfig, logger)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		panic(err)
	}

	fmt.Println("WebSocket server started on :8080")

	// In a real application, you would keep the server running
	// For this example, we'll stop it immediately
	server.Stop(ctx)
}

// Example demonstrates sending a notification
func Example_sendNotification() {
	logger, _ := zap.NewDevelopment()

	config := &inapp.Config{
		Enabled:   true,
		Transport: "websocket",
	}

	provider := inapp.NewProvider(config, logger)
	ctx := context.Background()
	provider.Start(ctx)
	defer provider.Stop(ctx)

	// Send notification
	notification := &inapp.Notification{
		ID:       "notif-123",
		UserID:   "user-456",
		Subject:  "Welcome!",
		Body:     "Welcome to DictaMesh",
		Priority: "normal",
		Data: map[string]interface{}{
			"source": "system",
		},
		CreatedAt: time.Now(),
	}

	err := provider.Send(ctx, notification)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Notification sent successfully")
	}
}

// Example demonstrates custom authentication
func Example_customAuth() {
	logger, _ := zap.NewDevelopment()

	// Custom authentication function
	customAuth := func(r *http.Request) (string, error) {
		// Extract and validate JWT token
		token := r.URL.Query().Get("token")

		// In production, validate the token properly
		if token == "" {
			return "", inapp.ErrUnauthorized
		}

		// Extract user ID from token
		userID := token // Simplified for example

		return userID, nil
	}

	// Configure with custom auth
	providerConfig := &inapp.Config{
		Enabled:   true,
		Transport: "websocket",
	}

	provider := inapp.NewProvider(providerConfig, logger)

	serverConfig := inapp.DefaultServerConfig()
	serverConfig.AuthFunc = customAuth

	server := inapp.NewServer(provider, serverConfig, logger)

	ctx := context.Background()
	server.Start(ctx)
	defer server.Stop(ctx)

	fmt.Println("Server with custom auth started")
}

// Example demonstrates broadcasting to multiple users
func Example_broadcast() {
	logger, _ := zap.NewDevelopment()

	config := &inapp.Config{
		Enabled:   true,
		Transport: "websocket",
	}

	provider := inapp.NewProvider(config, logger)
	ctx := context.Background()
	provider.Start(ctx)
	defer provider.Stop(ctx)

	// Broadcast to multiple users
	users := []string{"user-1", "user-2", "user-3"}

	for _, userID := range users {
		notification := &inapp.Notification{
			ID:       fmt.Sprintf("notif-%s", userID),
			UserID:   userID,
			Subject:  "System Alert",
			Body:     "Important system maintenance scheduled",
			Priority: "high",
			CreatedAt: time.Now(),
		}

		provider.Send(ctx, notification)
	}

	fmt.Println("Broadcast sent to all users")
}
