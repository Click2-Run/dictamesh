// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Provider implements the in-app notification channel using WebSocket
type Provider struct {
	hub    *Hub
	config *Config
	logger *zap.Logger
}

// Config holds configuration for the in-app provider
type Config struct {
	Enabled           bool
	Transport         string        // websocket | sse | longpoll
	PersistenceDays   int
	MaxUnread         int
	WebSocketPath     string
	WebSocketPingTime time.Duration
}

// NewProvider creates a new in-app notification provider
func NewProvider(config *Config, logger *zap.Logger) *Provider {
	hub := NewHub(logger)

	// Set up handlers for acknowledgments and read receipts
	hub.SetAckHandler(func(userID, notificationID string) {
		logger.Debug("notification acknowledged",
			zap.String("user_id", userID),
			zap.String("notification_id", notificationID),
		)
		// TODO: Update notification status in database
	})

	hub.SetReadHandler(func(userID, notificationID string) {
		logger.Debug("notification marked as read",
			zap.String("user_id", userID),
			zap.String("notification_id", notificationID),
		)
		// TODO: Update notification status in database
	})

	return &Provider{
		hub:    hub,
		config: config,
		logger: logger,
	}
}

// Start starts the provider
func (p *Provider) Start(ctx context.Context) error {
	p.logger.Info("starting in-app notification provider")

	// Start the hub
	go p.hub.Run()

	return nil
}

// Stop stops the provider
func (p *Provider) Stop(ctx context.Context) error {
	p.logger.Info("stopping in-app notification provider")
	return p.hub.Shutdown(ctx)
}

// Send sends an in-app notification via WebSocket
func (p *Provider) Send(ctx context.Context, notification *Notification) error {
	if !p.config.Enabled {
		return fmt.Errorf("in-app notifications are disabled")
	}

	// Create notification message
	notifMsg := &NotificationMessage{
		NotificationID: notification.ID,
		Subject:        notification.Subject,
		Body:           notification.Body,
		Priority:       notification.Priority,
		Category:       notification.Category,
		Data:           notification.Data,
		CreatedAt:      notification.CreatedAt,
		ExpiresAt:      notification.ExpiresAt,
		ActionURL:      notification.ActionURL,
		IconURL:        notification.IconURL,
	}

	msg := NewNotificationMessage(notifMsg)

	// Send to user via hub
	p.hub.SendToUser(notification.UserID, msg)

	p.logger.Info("in-app notification sent",
		zap.String("notification_id", notification.ID),
		zap.String("user_id", notification.UserID),
	)

	return nil
}

// GetHub returns the WebSocket hub
func (p *Provider) GetHub() *Hub {
	return p.hub
}

// GetChannel returns the channel name
func (p *Provider) GetChannel() string {
	return "IN_APP"
}

// HealthCheck performs a health check
func (p *Provider) HealthCheck(ctx context.Context) error {
	if !p.config.Enabled {
		return fmt.Errorf("provider is disabled")
	}

	// Check if hub is running
	if p.hub == nil {
		return fmt.Errorf("hub is not initialized")
	}

	return nil
}

// GetMetrics returns provider metrics
func (p *Provider) GetMetrics() *HubMetrics {
	return p.hub.GetMetrics()
}

// Notification represents a notification to be sent
type Notification struct {
	ID        string
	UserID    string
	Subject   string
	Body      string
	Priority  string
	Category  string
	Data      map[string]interface{}
	CreatedAt time.Time
	ExpiresAt *time.Time
	ActionURL string
	IconURL   string
}
