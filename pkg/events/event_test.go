// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	data := map[string]interface{}{
		"key":   "value",
		"count": 42,
	}

	event := NewEvent(
		EventTypeEntityCreated,
		"test-source",
		"test-subject",
		data,
	)

	// Verify event fields
	if event.ID == "" {
		t.Error("Event ID should not be empty")
	}

	if event.Type != EventTypeEntityCreated {
		t.Errorf("Expected type %s, got %s", EventTypeEntityCreated, event.Type)
	}

	if event.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got %s", event.Source)
	}

	if event.Subject != "test-subject" {
		t.Errorf("Expected subject 'test-subject', got %s", event.Subject)
	}

	if event.Data == nil {
		t.Error("Event data should not be nil")
	}

	if event.Data["key"] != "value" {
		t.Error("Event data should contain the correct values")
	}

	if event.Metadata == nil {
		t.Error("Event metadata should be initialized")
	}

	if event.Timestamp.IsZero() {
		t.Error("Event timestamp should be set")
	}

	// Check timestamp is recent
	if time.Since(event.Timestamp) > 1*time.Second {
		t.Error("Event timestamp should be recent")
	}
}

func TestEventWithCorrelationID(t *testing.T) {
	event := NewEvent(
		EventTypeEntityCreated,
		"test-source",
		"test-subject",
		nil,
	).WithCorrelationID("correlation-123")

	if event.CorrelationID != "correlation-123" {
		t.Errorf("Expected correlation ID 'correlation-123', got %s", event.CorrelationID)
	}
}

func TestEventWithCausationID(t *testing.T) {
	event := NewEvent(
		EventTypeEntityCreated,
		"test-source",
		"test-subject",
		nil,
	).WithCausationID("causation-456")

	if event.CausationID != "causation-456" {
		t.Errorf("Expected causation ID 'causation-456', got %s", event.CausationID)
	}
}

func TestEventWithMetadata(t *testing.T) {
	event := NewEvent(
		EventTypeEntityCreated,
		"test-source",
		"test-subject",
		nil,
	).WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	if event.Metadata["key1"] != "value1" {
		t.Error("Metadata should contain key1")
	}

	if event.Metadata["key2"] != "value2" {
		t.Error("Metadata should contain key2")
	}
}

func TestEventChaining(t *testing.T) {
	event := NewEvent(
		EventTypeEntityCreated,
		"test-source",
		"test-subject",
		nil,
	).WithCorrelationID("correlation-123").
		WithCausationID("causation-456").
		WithMetadata("key", "value")

	if event.CorrelationID != "correlation-123" {
		t.Error("Correlation ID should be set")
	}

	if event.CausationID != "causation-456" {
		t.Error("Causation ID should be set")
	}

	if event.Metadata["key"] != "value" {
		t.Error("Metadata should be set")
	}
}

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	id2 := generateEventID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id2 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	// Check UUID format (basic check)
	if len(id1) != 36 {
		t.Errorf("Expected UUID format (length 36), got length %d", len(id1))
	}
}

func TestEventTypes(t *testing.T) {
	// Test that event type constants are defined
	types := []string{
		EventTypeEntityCreated,
		EventTypeEntityUpdated,
		EventTypeEntityDeleted,
		EventTypeEntityRead,
		EventTypeRelationshipCreated,
		EventTypeRelationshipDeleted,
		EventTypeSchemaRegistered,
		EventTypeSchemaUpdated,
		EventTypeCacheInvalidated,
		EventTypeCacheWarmed,
		EventTypeAdapterStarted,
		EventTypeAdapterStopped,
		EventTypeHealthChanged,
	}

	for _, eventType := range types {
		if eventType == "" {
			t.Error("Event type constant should not be empty")
		}
	}
}
