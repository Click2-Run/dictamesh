// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"time"

	"github.com/google/uuid"
)

// Event represents a generic event in the DictaMesh framework
type Event struct {
	// ID is the unique identifier for this event
	ID string `json:"id"`

	// Type is the event type (e.g., "entity.created", "entity.updated")
	Type string `json:"type"`

	// Source identifies the source system or adapter
	Source string `json:"source"`

	// Subject is the entity or resource this event is about
	Subject string `json:"subject"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Data contains the event payload
	Data map[string]interface{} `json:"data"`

	// Metadata contains additional metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// CorrelationID links related events
	CorrelationID string `json:"correlation_id,omitempty"`

	// CausationID identifies what caused this event
	CausationID string `json:"causation_id,omitempty"`
}

// Common event types
const (
	// Entity events
	EventTypeEntityCreated     = "entity.created"
	EventTypeEntityUpdated     = "entity.updated"
	EventTypeEntityDeleted     = "entity.deleted"
	EventTypeEntityRead        = "entity.read"

	// Relationship events
	EventTypeRelationshipCreated = "relationship.created"
	EventTypeRelationshipDeleted = "relationship.deleted"

	// Schema events
	EventTypeSchemaRegistered = "schema.registered"
	EventTypeSchemaUpdated    = "schema.updated"

	// Cache events
	EventTypeCacheInvalidated = "cache.invalidated"
	EventTypeCacheWarmed      = "cache.warmed"

	// System events
	EventTypeAdapterStarted = "adapter.started"
	EventTypeAdapterStopped = "adapter.stopped"
	EventTypeHealthChanged  = "health.changed"
)

// NewEvent creates a new event
func NewEvent(eventType, source, subject string, data map[string]interface{}) *Event {
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		Subject:   subject,
		Timestamp: time.Now().UTC(),
		Data:      data,
		Metadata:  make(map[string]string),
	}
}

// WithCorrelationID sets the correlation ID
func (e *Event) WithCorrelationID(id string) *Event {
	e.CorrelationID = id
	return e
}

// WithCausationID sets the causation ID
func (e *Event) WithCausationID(id string) *Event {
	e.CausationID = id
	return e
}

// WithMetadata adds metadata
func (e *Event) WithMetadata(key, value string) *Event {
	e.Metadata[key] = value
	return e
}

// generateEventID generates a unique event ID using UUID v4
func generateEventID() string {
	return uuid.New().String()
}
