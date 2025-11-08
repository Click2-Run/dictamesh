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
)

// Producer wraps Kafka producer with observability
type Producer struct {
	producer *kafka.Producer
	config   *Config
	logger   *observability.Logger
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *Config, logger *observability.Logger) (*Producer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	kafkaConfig := cfg.GetProducerConfig()

	// Create Kafka config map with all producer settings
	configMap := &kafka.ConfigMap{}
	for key, value := range kafkaConfig {
		configMap.SetKey(key, value)
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	p := &Producer{
		producer: producer,
		config:   cfg,
		logger:   logger,
	}

	// Start delivery report handler
	go p.handleDeliveryReports()

	logger.Info("Kafka producer created",
		"bootstrap_servers", cfg.BootstrapServersString(),
	)

	return p, nil
}

// Publish publishes an event to a topic
func (p *Producer) Publish(ctx context.Context, topic string, event *Event) error {
	start := time.Now()

	// Serialize event
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Add topic prefix
	fullTopic := p.config.TopicName(topic)

	// Create Kafka message
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &fullTopic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(event.ID),
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "source", Value: []byte(event.Source)},
		},
	}

	// Add trace context headers if available
	traceID := observability.TraceID(ctx)
	if traceID != "" {
		msg.Headers = append(msg.Headers,
			kafka.Header{Key: "trace_id", Value: []byte(traceID)},
			kafka.Header{Key: "span_id", Value: []byte(observability.SpanID(ctx))},
		)
	}

	// Produce message
	deliveryChan := make(chan kafka.Event)
	if err := p.producer.Produce(msg, deliveryChan); err != nil {
		p.logger.ErrorContext(ctx, "failed to produce message",
			"topic", fullTopic,
			"error", err,
		)
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery confirmation
	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		p.logger.ErrorContext(ctx, "message delivery failed",
			"topic", fullTopic,
			"error", m.TopicPartition.Error,
		)
		return m.TopicPartition.Error
	}

	duration := time.Since(start)
	p.logger.DebugContext(ctx, "event published",
		"topic", fullTopic,
		"partition", m.TopicPartition.Partition,
		"offset", m.TopicPartition.Offset,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// handleDeliveryReports handles asynchronous delivery reports
func (p *Producer) handleDeliveryReports() {
	for e := range p.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				p.logger.Error("delivery failed",
					"topic", *ev.TopicPartition.Topic,
					"partition", ev.TopicPartition.Partition,
					"error", ev.TopicPartition.Error,
				)
			}
		}
	}
}

// Flush waits for all messages to be delivered
func (p *Producer) Flush(timeoutMs int) int {
	return p.producer.Flush(timeoutMs)
}

// Close closes the producer
func (p *Producer) Close() error {
	p.producer.Close()
	p.logger.Info("Kafka producer closed")
	return nil
}
