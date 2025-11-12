// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Hub maintains the set of active connections and broadcasts messages to connections
type Hub struct {
	// Registered connections indexed by connection ID
	connections map[string]*Connection

	// User connections indexed by user ID (one user can have multiple connections)
	userConnections map[string]map[string]*Connection

	// Channel subscriptions: channel -> connection IDs
	subscriptions map[string]map[string]bool

	// Register requests from connections
	register chan *Connection

	// Unregister requests from connections
	unregister chan *Connection

	// Broadcast message to specific user
	broadcast chan *BroadcastMessage

	// Broadcast to channel/topic
	broadcastChannel chan *ChannelBroadcast

	// Acknowledgment handler
	ackHandler AckHandler

	// Read receipt handler
	readHandler ReadHandler

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *HubMetrics
}

// BroadcastMessage represents a message to broadcast to a user
type BroadcastMessage struct {
	UserID  string
	Message *Message
}

// ChannelBroadcast represents a message to broadcast to a channel
type ChannelBroadcast struct {
	Channel string
	Message *Message
}

// AckHandler handles acknowledgment of notifications
type AckHandler func(userID, notificationID string)

// ReadHandler handles read receipts
type ReadHandler func(userID, notificationID string)

// HubMetrics holds hub statistics
type HubMetrics struct {
	mu                  sync.RWMutex
	TotalConnections    int64
	ActiveConnections   int64
	TotalMessages       int64
	TotalBroadcasts     int64
	TotalAcks           int64
	TotalReads          int64
	LastMessageTime     time.Time
	ConnectionsCreated  int64
	ConnectionsDropped  int64
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		connections:      make(map[string]*Connection),
		userConnections:  make(map[string]map[string]*Connection),
		subscriptions:    make(map[string]map[string]bool),
		register:         make(chan *Connection),
		unregister:       make(chan *Connection),
		broadcast:        make(chan *BroadcastMessage, 256),
		broadcastChannel: make(chan *ChannelBroadcast, 256),
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
		metrics:          &HubMetrics{},
	}
}

// Run starts the hub
func (h *Hub) Run() {
	h.logger.Info("starting websocket hub")

	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("shutting down websocket hub")
			return

		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case msg := <-h.broadcast:
			h.broadcastToUser(msg)

		case msg := <-h.broadcastChannel:
			h.broadcastToChannel(msg)
		}
	}
}

// registerConnection registers a new connection
func (h *Hub) registerConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to connections map
	h.connections[conn.ID] = conn

	// Add to user connections
	if h.userConnections[conn.UserID] == nil {
		h.userConnections[conn.UserID] = make(map[string]*Connection)
	}
	h.userConnections[conn.UserID][conn.ID] = conn

	// Update metrics
	h.metrics.mu.Lock()
	h.metrics.ActiveConnections++
	h.metrics.ConnectionsCreated++
	h.metrics.TotalConnections = int64(len(h.connections))
	h.metrics.mu.Unlock()

	h.logger.Info("connection registered",
		zap.String("connection_id", conn.ID),
		zap.String("user_id", conn.UserID),
		zap.Int("total_connections", len(h.connections)),
	)

	// Send welcome message
	conn.SendMessage(NewWelcomeMessage(conn.UserID, conn.ID))
}

// unregisterConnection unregisters a connection
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from connections
	if _, ok := h.connections[conn.ID]; ok {
		delete(h.connections, conn.ID)

		// Remove from user connections
		if userConns, ok := h.userConnections[conn.UserID]; ok {
			delete(userConns, conn.ID)
			if len(userConns) == 0 {
				delete(h.userConnections, conn.UserID)
			}
		}

		// Remove from all subscriptions
		for channel, subs := range h.subscriptions {
			delete(subs, conn.ID)
			if len(subs) == 0 {
				delete(h.subscriptions, channel)
			}
		}

		// Close connection
		conn.Close()

		// Update metrics
		h.metrics.mu.Lock()
		h.metrics.ActiveConnections--
		h.metrics.ConnectionsDropped++
		h.metrics.TotalConnections = int64(len(h.connections))
		h.metrics.mu.Unlock()

		h.logger.Info("connection unregistered",
			zap.String("connection_id", conn.ID),
			zap.String("user_id", conn.UserID),
			zap.Int("total_connections", len(h.connections)),
		)
	}
}

// broadcastToUser sends a message to all connections of a user
func (h *Hub) broadcastToUser(msg *BroadcastMessage) {
	h.mu.RLock()
	userConns := h.userConnections[msg.UserID]
	h.mu.RUnlock()

	if len(userConns) == 0 {
		h.logger.Debug("no active connections for user", zap.String("user_id", msg.UserID))
		return
	}

	// Update metrics
	h.metrics.mu.Lock()
	h.metrics.TotalBroadcasts++
	h.metrics.TotalMessages++
	h.metrics.LastMessageTime = time.Now()
	h.metrics.mu.Unlock()

	// Send to all user connections
	for _, conn := range userConns {
		if err := conn.SendMessage(msg.Message); err != nil {
			h.logger.Error("failed to send message",
				zap.String("connection_id", conn.ID),
				zap.Error(err),
			)
		}
	}

	h.logger.Debug("broadcasted message to user",
		zap.String("user_id", msg.UserID),
		zap.Int("connections", len(userConns)),
		zap.String("message_type", string(msg.Message.Type)),
	)
}

// broadcastToChannel sends a message to all subscribers of a channel
func (h *Hub) broadcastToChannel(msg *ChannelBroadcast) {
	h.mu.RLock()
	subs := h.subscriptions[msg.Channel]
	h.mu.RUnlock()

	if len(subs) == 0 {
		h.logger.Debug("no subscribers for channel", zap.String("channel", msg.Channel))
		return
	}

	// Update metrics
	h.metrics.mu.Lock()
	h.metrics.TotalBroadcasts++
	h.metrics.TotalMessages += int64(len(subs))
	h.metrics.LastMessageTime = time.Now()
	h.metrics.mu.Unlock()

	// Send to all subscribers
	h.mu.RLock()
	for connID := range subs {
		if conn, ok := h.connections[connID]; ok {
			if err := conn.SendMessage(msg.Message); err != nil {
				h.logger.Error("failed to send message",
					zap.String("connection_id", conn.ID),
					zap.Error(err),
				)
			}
		}
	}
	h.mu.RUnlock()

	h.logger.Debug("broadcasted message to channel",
		zap.String("channel", msg.Channel),
		zap.Int("subscribers", len(subs)),
		zap.String("message_type", string(msg.Message.Type)),
	)
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, msg *Message) {
	select {
	case h.broadcast <- &BroadcastMessage{UserID: userID, Message: msg}:
	case <-time.After(5 * time.Second):
		h.logger.Error("broadcast timeout", zap.String("user_id", userID))
	case <-h.ctx.Done():
		h.logger.Warn("hub is shutting down, cannot send message")
	}
}

// SendToChannel sends a message to a channel
func (h *Hub) SendToChannel(channel string, msg *Message) {
	select {
	case h.broadcastChannel <- &ChannelBroadcast{Channel: channel, Message: msg}:
	case <-time.After(5 * time.Second):
		h.logger.Error("channel broadcast timeout", zap.String("channel", channel))
	case <-h.ctx.Done():
		h.logger.Warn("hub is shutting down, cannot send message")
	}
}

// Subscribe subscribes a connection to a channel
func (h *Hub) Subscribe(connectionID, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscriptions[channel] == nil {
		h.subscriptions[channel] = make(map[string]bool)
	}
	h.subscriptions[channel][connectionID] = true

	h.logger.Debug("connection subscribed to channel",
		zap.String("connection_id", connectionID),
		zap.String("channel", channel),
	)
}

// Unsubscribe unsubscribes a connection from a channel
func (h *Hub) Unsubscribe(connectionID, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.subscriptions[channel]; ok {
		delete(subs, connectionID)
		if len(subs) == 0 {
			delete(h.subscriptions, channel)
		}
	}

	h.logger.Debug("connection unsubscribed from channel",
		zap.String("connection_id", connectionID),
		zap.String("channel", channel),
	)
}

// SetAckHandler sets the acknowledgment handler
func (h *Hub) SetAckHandler(handler AckHandler) {
	h.ackHandler = handler
}

// SetReadHandler sets the read receipt handler
func (h *Hub) SetReadHandler(handler ReadHandler) {
	h.readHandler = handler
}

// HandleAck handles acknowledgment of a notification
func (h *Hub) HandleAck(userID, notificationID string) {
	if h.ackHandler != nil {
		h.ackHandler(userID, notificationID)
	}

	h.metrics.mu.Lock()
	h.metrics.TotalAcks++
	h.metrics.mu.Unlock()

	h.logger.Debug("notification acknowledged",
		zap.String("user_id", userID),
		zap.String("notification_id", notificationID),
	)
}

// HandleRead handles read receipt of a notification
func (h *Hub) HandleRead(userID, notificationID string) {
	if h.readHandler != nil {
		h.readHandler(userID, notificationID)
	}

	h.metrics.mu.Lock()
	h.metrics.TotalReads++
	h.metrics.mu.Unlock()

	h.logger.Debug("notification read",
		zap.String("user_id", userID),
		zap.String("notification_id", notificationID),
	)
}

// GetUserConnections returns all connections for a user
func (h *Hub) GetUserConnections(userID string) []*Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var conns []*Connection
	if userConns, ok := h.userConnections[userID]; ok {
		for _, conn := range userConns {
			conns = append(conns, conn)
		}
	}
	return conns
}

// GetConnectionCount returns the number of active connections
func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// GetUserCount returns the number of unique users connected
func (h *Hub) GetUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userConnections)
}

// GetMetrics returns hub metrics
func (h *Hub) GetMetrics() *HubMetrics {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	return &HubMetrics{
		TotalConnections:   h.metrics.TotalConnections,
		ActiveConnections:  h.metrics.ActiveConnections,
		TotalMessages:      h.metrics.TotalMessages,
		TotalBroadcasts:    h.metrics.TotalBroadcasts,
		TotalAcks:          h.metrics.TotalAcks,
		TotalReads:         h.metrics.TotalReads,
		LastMessageTime:    h.metrics.LastMessageTime,
		ConnectionsCreated: h.metrics.ConnectionsCreated,
		ConnectionsDropped: h.metrics.ConnectionsDropped,
	}
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown(ctx context.Context) error {
	h.logger.Info("shutting down hub")
	h.cancel()

	// Close all connections
	h.mu.Lock()
	for _, conn := range h.connections {
		conn.Close()
	}
	h.mu.Unlock()

	return nil
}
