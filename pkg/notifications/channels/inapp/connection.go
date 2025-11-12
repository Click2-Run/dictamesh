// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Connection represents a WebSocket client connection
type Connection struct {
	// Connection identification
	ID     string
	UserID string

	// WebSocket connection
	ws *websocket.Conn

	// Message channels
	send chan []byte

	// Hub reference
	hub *Hub

	// Connection metadata
	metadata map[string]interface{}

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// State management
	mu         sync.RWMutex
	lastPing   time.Time
	lastPong   time.Time
	connected  bool

	// Logger
	logger *zap.Logger
}

// NewConnection creates a new WebSocket connection
func NewConnection(id, userID string, ws *websocket.Conn, hub *Hub, logger *zap.Logger) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	return &Connection{
		ID:        id,
		UserID:    userID,
		ws:        ws,
		send:      make(chan []byte, 256),
		hub:       hub,
		metadata:  make(map[string]interface{}),
		ctx:       ctx,
		cancel:    cancel,
		lastPing:  time.Now(),
		lastPong:  time.Now(),
		connected: true,
		logger:    logger.With(zap.String("connection_id", id), zap.String("user_id", userID)),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Connection) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.lastPong = time.Now()
		c.mu.Unlock()
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("websocket read error", zap.Error(err))
				}
				return
			}

			// Process incoming message
			if err := c.handleMessage(message); err != nil {
				c.logger.Error("failed to handle message", zap.Error(err))
			}
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.ws.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.mu.Lock()
			c.lastPing = time.Now()
			c.mu.Unlock()

			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Connection) SendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	case <-time.After(5 * time.Second):
		return ErrSendTimeout
	case <-c.ctx.Done():
		return ErrConnectionClosed
	}
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	c.cancel()
	close(c.send)

	return c.ws.Close()
}

// IsConnected returns whether the connection is active
func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetMetadata returns connection metadata
func (c *Connection) GetMetadata() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metadata := make(map[string]interface{})
	for k, v := range c.metadata {
		metadata[k] = v
	}
	return metadata
}

// SetMetadata sets connection metadata
func (c *Connection) SetMetadata(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metadata[key] = value
}

// handleMessage processes incoming messages from the client
func (c *Connection) handleMessage(data []byte) error {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	c.logger.Debug("received message", zap.String("type", string(msg.Type)))

	// Handle different message types
	switch msg.Type {
	case MessageTypeAck:
		// Handle acknowledgment
		if notifID, ok := msg.Data["notification_id"].(string); ok {
			c.hub.HandleAck(c.UserID, notifID)
		}
	case MessageTypeRead:
		// Handle read receipt
		if notifID, ok := msg.Data["notification_id"].(string); ok {
			c.hub.HandleRead(c.UserID, notifID)
		}
	case MessageTypePing:
		// Respond with pong
		c.SendMessage(&Message{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
		})
	case MessageTypeSubscribe:
		// Handle subscription to channels/topics
		if channels, ok := msg.Data["channels"].([]interface{}); ok {
			for _, ch := range channels {
				if channel, ok := ch.(string); ok {
					c.hub.Subscribe(c.ID, channel)
				}
			}
		}
	case MessageTypeUnsubscribe:
		// Handle unsubscription
		if channels, ok := msg.Data["channels"].([]interface{}); ok {
			for _, ch := range channels {
				if channel, ok := ch.(string); ok {
					c.hub.Unsubscribe(c.ID, channel)
				}
			}
		}
	default:
		c.logger.Warn("unknown message type", zap.String("type", string(msg.Type)))
	}

	return nil
}

// GetStats returns connection statistics
func (c *Connection) GetStats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return ConnectionStats{
		ConnectionID: c.ID,
		UserID:       c.UserID,
		LastPing:     c.lastPing,
		LastPong:     c.lastPong,
		Connected:    c.connected,
		QueueSize:    len(c.send),
	}
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	ConnectionID string
	UserID       string
	LastPing     time.Time
	LastPong     time.Time
	Connected    bool
	QueueSize    int
}
