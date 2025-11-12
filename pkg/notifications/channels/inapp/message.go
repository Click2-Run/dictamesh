// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"time"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	// Client -> Server messages
	MessageTypeAck         MessageType = "ack"         // Acknowledge notification receipt
	MessageTypeRead        MessageType = "read"        // Mark notification as read
	MessageTypePing        MessageType = "ping"        // Client ping
	MessageTypeSubscribe   MessageType = "subscribe"   // Subscribe to channels/topics
	MessageTypeUnsubscribe MessageType = "unsubscribe" // Unsubscribe from channels/topics

	// Server -> Client messages
	MessageTypeNotification MessageType = "notification" // New notification
	MessageTypePong         MessageType = "pong"         // Server pong response
	MessageTypeStatus       MessageType = "status"       // Connection status update
	MessageTypeError        MessageType = "error"        // Error message
	MessageTypeWelcome      MessageType = "welcome"      // Initial connection message
)

// Message represents a WebSocket message
type Message struct {
	// Message metadata
	Type      MessageType            `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`

	// Message payload
	Data map[string]interface{} `json:"data,omitempty"`

	// Error information (for error messages)
	Error *MessageError `json:"error,omitempty"`
}

// MessageError represents an error in a message
type MessageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NotificationMessage represents a notification delivered via WebSocket
type NotificationMessage struct {
	NotificationID string                 `json:"notification_id"`
	Subject        string                 `json:"subject"`
	Body           string                 `json:"body"`
	Priority       string                 `json:"priority"`
	Category       string                 `json:"category,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	ActionURL      string                 `json:"action_url,omitempty"`
	IconURL        string                 `json:"icon_url,omitempty"`
}

// NewNotificationMessage creates a notification message
func NewNotificationMessage(notification *NotificationMessage) *Message {
	return &Message{
		Type:      MessageTypeNotification,
		ID:        notification.NotificationID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"notification_id": notification.NotificationID,
			"subject":         notification.Subject,
			"body":            notification.Body,
			"priority":        notification.Priority,
			"category":        notification.Category,
			"data":            notification.Data,
			"created_at":      notification.CreatedAt,
			"expires_at":      notification.ExpiresAt,
			"action_url":      notification.ActionURL,
			"icon_url":        notification.IconURL,
		},
	}
}

// NewWelcomeMessage creates a welcome message
func NewWelcomeMessage(userID string, connectionID string) *Message {
	return &Message{
		Type:      MessageTypeWelcome,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_id":       userID,
			"connection_id": connectionID,
			"server_time":   time.Now().Unix(),
			"version":       "1.0",
		},
	}
}

// NewStatusMessage creates a status message
func NewStatusMessage(status string, details map[string]interface{}) *Message {
	data := map[string]interface{}{
		"status": status,
	}
	for k, v := range details {
		data[k] = v
	}

	return &Message{
		Type:      MessageTypeStatus,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// NewErrorMessage creates an error message
func NewErrorMessage(code, message, details string) *Message {
	return &Message{
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		Error: &MessageError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// NewPongMessage creates a pong message
func NewPongMessage() *Message {
	return &Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	}
}
