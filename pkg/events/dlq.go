// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/click2-run/dictamesh/pkg/observability"
	"github.com/google/uuid"
)

// DeadLetterQueue handles failed message processing
type DeadLetterQueue struct {
	producer *Producer
	config   *Config
	logger   *observability.Logger
}

// NewDeadLetterQueue creates a new dead letter queue handler
func NewDeadLetterQueue(cfg *Config, logger *observability.Logger) (*DeadLetterQueue, error) {
	producer, err := NewProducer(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ producer: %w", err)
	}

	return &DeadLetterQueue{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}, nil
}

// DeadLetterMessage represents a message that failed processing
type DeadLetterMessage struct {
	// ID is the unique identifier for this DLQ entry
	ID string `json:"id"`

	// OriginalEvent is the event that failed processing
	OriginalEvent *Event `json:"original_event"`

	// OriginalTopic is the topic the event was consumed from
	OriginalTopic string `json:"original_topic"`

	// Error is the error that occurred during processing
	Error string `json:"error"`

	// ErrorStackTrace is the stack trace of the error (if available)
	ErrorStackTrace string `json:"error_stack_trace,omitempty"`

	// RetryCount is the number of times processing was attempted
	RetryCount int `json:"retry_count"`

	// FirstFailedAt is when the first failure occurred
	FirstFailedAt time.Time `json:"first_failed_at"`

	// LastFailedAt is when the last failure occurred
	LastFailedAt time.Time `json:"last_failed_at"`

	// ConsumerGroup is the consumer group that failed to process
	ConsumerGroup string `json:"consumer_group"`

	// Metadata contains additional context
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Send sends a failed message to the dead letter queue
func (dlq *DeadLetterQueue) Send(ctx context.Context, msg *DeadLetterMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	if msg.FirstFailedAt.IsZero() {
		msg.FirstFailedAt = time.Now().UTC()
	}

	if msg.LastFailedAt.IsZero() {
		msg.LastFailedAt = time.Now().UTC()
	}

	// Create DLQ event
	dlqEvent := NewEvent(
		"message.failed",
		"dlq",
		msg.OriginalTopic,
		map[string]interface{}{
			"dead_letter_message": msg,
		},
	)

	// Add correlation from original event if available
	if msg.OriginalEvent != nil {
		dlqEvent.CorrelationID = msg.OriginalEvent.CorrelationID
		dlqEvent.WithMetadata("original_event_id", msg.OriginalEvent.ID)
		dlqEvent.WithMetadata("original_event_type", msg.OriginalEvent.Type)
	}

	dlqEvent.WithMetadata("retry_count", fmt.Sprintf("%d", msg.RetryCount))
	dlqEvent.WithMetadata("consumer_group", msg.ConsumerGroup)

	// Publish to DLQ topic
	if err := dlq.producer.Publish(ctx, TopicDeadLetter, dlqEvent); err != nil {
		dlq.logger.ErrorContext(ctx, "failed to send message to DLQ",
			"original_topic", msg.OriginalTopic,
			"error", err,
		)
		return fmt.Errorf("failed to send to DLQ: %w", err)
	}

	dlq.logger.WarnContext(ctx, "message sent to dead letter queue",
		"dlq_id", msg.ID,
		"original_topic", msg.OriginalTopic,
		"retry_count", msg.RetryCount,
		"error", msg.Error,
	)

	return nil
}

// SendWithOriginalMessage sends a failed message with the original Kafka message
func (dlq *DeadLetterQueue) SendWithOriginalMessage(
	ctx context.Context,
	originalEvent *Event,
	originalTopic string,
	err error,
	retryCount int,
	consumerGroup string,
) error {
	msg := &DeadLetterMessage{
		OriginalEvent: originalEvent,
		OriginalTopic: originalTopic,
		Error:         err.Error(),
		RetryCount:    retryCount,
		ConsumerGroup: consumerGroup,
		Metadata:      make(map[string]string),
	}

	return dlq.Send(ctx, msg)
}

// Close closes the dead letter queue
func (dlq *DeadLetterQueue) Close() error {
	return dlq.producer.Close()
}

// RetryableError indicates whether an error should be retried
type RetryableError struct {
	Err        error
	Retryable  bool
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, retryAfter time.Duration) *RetryableError {
	return &RetryableError{
		Err:        err,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

// NewNonRetryableError creates a new non-retryable error
func NewNonRetryableError(err error) *RetryableError {
	return &RetryableError{
		Err:       err,
		Retryable: false,
	}
}

// ConsumerWithDLQ wraps a consumer with automatic dead letter queue handling
type ConsumerWithDLQ struct {
	consumer    *Consumer
	dlq         *DeadLetterQueue
	config      *Config
	logger      *observability.Logger
	maxRetries  int
	retryDelay  time.Duration
	retryCounts map[string]int // event ID -> retry count
	userHandler EventHandler   // original user handler
}

// NewConsumerWithDLQ creates a consumer with dead letter queue support
func NewConsumerWithDLQ(
	cfg *Config,
	logger *observability.Logger,
	handler EventHandler,
	maxRetries int,
	retryDelay time.Duration,
) (*ConsumerWithDLQ, error) {
	// Create DLQ
	dlq, err := NewDeadLetterQueue(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ: %w", err)
	}

	// Create the ConsumerWithDLQ struct first (without consumer)
	cwdlq := &ConsumerWithDLQ{
		dlq:         dlq,
		config:      cfg,
		logger:      logger,
		maxRetries:  maxRetries,
		retryDelay:  retryDelay,
		retryCounts: make(map[string]int),
		userHandler: handler,
	}

	// Now create a wrapped handler that uses the ConsumerWithDLQ instance
	wrappedHandler := func(ctx context.Context, event *Event) error {
		// Extract topic from event metadata (will be added by consumer)
		topic := event.Metadata["_kafka_topic"]
		if topic == "" {
			topic = "unknown"
		}
		return cwdlq.handleEventWithRetry(ctx, event, cwdlq.userHandler, topic)
	}

	// Create consumer with wrapped handler
	consumer, err := NewConsumer(cfg, logger, wrappedHandler)
	if err != nil {
		dlq.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Set the consumer
	cwdlq.consumer = consumer

	return cwdlq, nil
}

// Subscribe subscribes to topics
func (c *ConsumerWithDLQ) Subscribe(topics []string) error {
	return c.consumer.Subscribe(topics)
}

// Start starts consuming with automatic DLQ handling
func (c *ConsumerWithDLQ) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

// Stop stops the consumer
func (c *ConsumerWithDLQ) Stop() {
	c.consumer.Stop()
}

// Close closes the consumer and DLQ
func (c *ConsumerWithDLQ) Close() error {
	if err := c.consumer.Close(); err != nil {
		c.logger.Error("failed to close consumer", "error", err)
	}
	if err := c.dlq.Close(); err != nil {
		c.logger.Error("failed to close DLQ", "error", err)
	}
	return nil
}

// handleEventWithRetry handles an event with retry logic and DLQ fallback
func (c *ConsumerWithDLQ) handleEventWithRetry(
	ctx context.Context,
	event *Event,
	handler EventHandler,
	topic string,
) error {
	var lastErr error

	// Get current retry count
	retryCount := c.retryCounts[event.ID]

	// Try processing
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		err := handler(ctx, event)
		if err == nil {
			// Success - clean up retry count
			delete(c.retryCounts, event.ID)
			return nil
		}

		lastErr = err

		// Check if error is retryable
		var retryableErr *RetryableError
		isRetryable := true
		retryAfter := c.retryDelay

		if retryErr, ok := err.(*RetryableError); ok {
			isRetryable = retryErr.Retryable
			if retryErr.RetryAfter > 0 {
				retryAfter = retryErr.RetryAfter
			}
		}

		if !isRetryable || attempt >= c.maxRetries {
			break
		}

		// Increment retry count
		retryCount++
		c.retryCounts[event.ID] = retryCount

		c.logger.WarnContext(ctx, "event processing failed, retrying",
			"event_id", event.ID,
			"attempt", attempt+1,
			"max_retries", c.maxRetries,
			"retry_after", retryAfter,
			"error", err,
		)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryAfter):
		}
	}

	// All retries exhausted - send to DLQ
	c.logger.ErrorContext(ctx, "event processing failed after all retries, sending to DLQ",
		"event_id", event.ID,
		"retry_count", retryCount,
		"error", lastErr,
	)

	dlqMsg := &DeadLetterMessage{
		OriginalEvent: event,
		OriginalTopic: topic,
		Error:         lastErr.Error(),
		RetryCount:    retryCount,
		ConsumerGroup: c.config.Consumer.GroupID,
	}

	if err := c.dlq.Send(ctx, dlqMsg); err != nil {
		c.logger.ErrorContext(ctx, "failed to send to DLQ", "error", err)
		return fmt.Errorf("failed to send to DLQ: %w", err)
	}

	// Clean up retry count
	delete(c.retryCounts, event.ID)

	// Return the original error
	return lastErr
}

// ProcessDeadLetterMessage processes a message from the dead letter queue
// This can be used to implement manual retry or investigation workflows
func ProcessDeadLetterMessage(data []byte) (*DeadLetterMessage, error) {
	var dlqMsg DeadLetterMessage
	if err := json.Unmarshal(data, &dlqMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ message: %w", err)
	}
	return &dlqMsg, nil
}
