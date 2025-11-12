// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package inapp

import (
	"errors"
	"time"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// Error definitions
var (
	ErrSendTimeout       = errors.New("send timeout")
	ErrConnectionClosed  = errors.New("connection closed")
	ErrInvalidMessage    = errors.New("invalid message format")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidToken      = errors.New("invalid token")
	ErrUserNotFound      = errors.New("user not found")
	ErrConnectionExists  = errors.New("connection already exists")
	ErrHubShuttingDown   = errors.New("hub is shutting down")
)
