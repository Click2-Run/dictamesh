// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Mock handler for testing
type mockHandler struct {
	called      int
	shouldError bool
	err         error
}

func (m *mockHandler) Handle(ctx context.Context, event *Event) error {
	m.called++
	if m.shouldError {
		return m.err
	}
	return nil
}

func TestDeadLetterMessage(t *testing.T) {
	event := NewEvent(
		EventTypeEntityCreated,
		"test-source",
		"test-subject",
		map[string]interface{}{"key": "value"},
	)

	dlqMsg := &DeadLetterMessage{
		ID:            "dlq-123",
		OriginalEvent: event,
		OriginalTopic: "test.topic",
		Error:         "processing failed",
		RetryCount:    3,
		FirstFailedAt: time.Now().Add(-5 * time.Minute),
		LastFailedAt:  time.Now(),
		ConsumerGroup: "test-consumer-group",
		Metadata:      map[string]string{"reason": "timeout"},
	}

	if dlqMsg.ID != "dlq-123" {
		t.Error("DLQ message ID should be set")
	}

	if dlqMsg.OriginalEvent.ID != event.ID {
		t.Error("DLQ message should contain original event")
	}

	if dlqMsg.RetryCount != 3 {
		t.Error("DLQ message should track retry count")
	}
}

func TestRetryableError(t *testing.T) {
	originalErr := fmt.Errorf("test error")

	retryableErr := NewRetryableError(originalErr, 100*time.Millisecond)
	if !retryableErr.Retryable {
		t.Error("Should be retryable")
	}
	if retryableErr.RetryAfter != 100*time.Millisecond {
		t.Error("Retry after should be set")
	}
	if retryableErr.Error() != "test error" {
		t.Error("Should preserve error message")
	}

	nonRetryableErr := NewNonRetryableError(originalErr)
	if nonRetryableErr.Retryable {
		t.Error("Should not be retryable")
	}
}

func TestRetryableErrorUnwrap(t *testing.T) {
	originalErr := fmt.Errorf("test error")
	retryableErr := NewRetryableError(originalErr, time.Second)

	unwrapped := retryableErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("Should unwrap to original error")
	}
}
