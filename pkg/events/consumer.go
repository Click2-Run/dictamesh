// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/click2-run/dictamesh/pkg/observability"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
)

// EventHandler processes consumed events
type EventHandler func(ctx context.Context, event *Event) error

// Consumer wraps Kafka consumer with observability
type Consumer struct {
	consumer *kafka.Consumer
	config   *Config
	logger   *observability.Logger
	handler  EventHandler
	running  bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *Config, logger *observability.Logger, handler EventHandler) (*Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	kafkaConfig := cfg.GetConsumerConfig()

	// Create Kafka config map with all consumer settings
	configMap := &kafka.ConfigMap{}
	for key, value := range kafkaConfig {
		configMap.SetKey(key, value)
	}

	consumer, err := kafka.NewConsumer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	logger.Info("Kafka consumer created",
		"bootstrap_servers", cfg.BootstrapServersString(),
		"group_id", cfg.Consumer.GroupID,
	)

	return &Consumer{
		consumer: consumer,
		config:   cfg,
		logger:   logger,
		handler:  handler,
		running:  false,
	}, nil
}

// Subscribe subscribes to topics
func (c *Consumer) Subscribe(topics []string) error {
	fullTopics := make([]string, len(topics))
	for i, topic := range topics {
		fullTopics[i] = c.config.TopicName(topic)
	}

	if err := c.consumer.SubscribeTopics(fullTopics, nil); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	c.logger.Info("subscribed to topics", "topics", fullTopics)
	return nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	c.running = true

	c.logger.Info("starting consumer")

	for c.running {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopped by context")
			return ctx.Err()
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Safe type assertion - check if it's a Kafka error
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				c.logger.Error("failed to read message", "error", err)
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("failed to process message",
					"topic", *msg.TopicPartition.Topic,
					"partition", msg.TopicPartition.Partition,
					"offset", msg.TopicPartition.Offset,
					"error", err,
				)
			} else {
				// Commit offset on success
				if _, err := c.consumer.CommitMessage(msg); err != nil {
					c.logger.Error("failed to commit offset", "error", err)
				}
			}
		}
	}

	return nil
}

// processMessage processes a single message
func (c *Consumer) processMessage(ctx context.Context, msg *kafka.Message) error {
	start := time.Now()

	// Extract trace context from Kafka headers using OpenTelemetry propagator
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, &kafkaHeaderCarrier{headers: msg.Headers})

	// Deserialize event
	var event Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Add Kafka metadata to event for downstream handlers (especially DLQ)
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	event.Metadata["_kafka_topic"] = *msg.TopicPartition.Topic
	event.Metadata["_kafka_partition"] = fmt.Sprintf("%d", msg.TopicPartition.Partition)
	event.Metadata["_kafka_offset"] = fmt.Sprintf("%d", msg.TopicPartition.Offset)

	// Call handler with enriched context
	if err := c.handler(ctx, &event); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	duration := time.Since(start)
	c.logger.DebugContext(ctx, "event processed",
		"topic", *msg.TopicPartition.Topic,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// kafkaHeaderCarrier implements propagation.TextMapCarrier for Kafka headers
type kafkaHeaderCarrier struct {
	headers []kafka.Header
}

func (c *kafkaHeaderCarrier) Get(key string) string {
	for _, h := range c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeaderCarrier) Set(key string, value string) {
	// For extraction, we don't need to implement Set
	// This would be used for injection (which we do in the producer)
}

func (c *kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(c.headers))
	for i, h := range c.headers {
		keys[i] = h.Key
	}
	return keys
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	c.running = false
}

// Close closes the consumer
func (c *Consumer) Close() error {
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}
	c.logger.Info("Kafka consumer closed")
	return nil
}
