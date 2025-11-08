// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"context"
	"fmt"
	"time"

	"github.com/click2-run/dictamesh/pkg/observability"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TopicManager manages Kafka topic lifecycle (create, delete, list, describe)
type TopicManager struct {
	adminClient *kafka.AdminClient
	config      *Config
	logger      *observability.Logger
}

// NewTopicManager creates a new topic manager
func NewTopicManager(cfg *Config, logger *observability.Logger) (*TopicManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": cfg.BootstrapServersString(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}

	logger.Info("Kafka admin client created",
		"bootstrap_servers", cfg.BootstrapServersString(),
	)

	return &TopicManager{
		adminClient: adminClient,
		config:      cfg,
		logger:      logger,
	}, nil
}

// CreateTopics creates multiple topics
func (tm *TopicManager) CreateTopics(ctx context.Context, topics []TopicConfiguration) error {
	if len(topics) == 0 {
		return nil
	}

	specs := make([]kafka.TopicSpecification, len(topics))
	for i, topic := range topics {
		specs[i] = kafka.TopicSpecification{
			Topic:             topic.Name,
			NumPartitions:     topic.Partitions,
			ReplicationFactor: topic.ReplicationFactor,
			Config: map[string]string{
				"retention.ms":   fmt.Sprintf("%d", topic.RetentionMs),
				"cleanup.policy": topic.CleanupPolicy,
			},
		}
	}

	results, err := tm.adminClient.CreateTopics(ctx, specs)
	if err != nil {
		return fmt.Errorf("failed to create topics: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError &&
			result.Error.Code() != kafka.ErrTopicAlreadyExists {
			tm.logger.Error("failed to create topic",
				"topic", result.Topic,
				"error", result.Error,
			)
			return fmt.Errorf("failed to create topic %s: %w", result.Topic, result.Error)
		}

		if result.Error.Code() == kafka.ErrTopicAlreadyExists {
			tm.logger.Info("topic already exists", "topic", result.Topic)
		} else {
			tm.logger.Info("topic created", "topic", result.Topic)
		}
	}

	return nil
}

// CreateTopic creates a single topic
func (tm *TopicManager) CreateTopic(ctx context.Context, topic TopicConfiguration) error {
	return tm.CreateTopics(ctx, []TopicConfiguration{topic})
}

// DeleteTopics deletes multiple topics
func (tm *TopicManager) DeleteTopics(ctx context.Context, topicNames []string) error {
	if len(topicNames) == 0 {
		return nil
	}

	results, err := tm.adminClient.DeleteTopics(ctx, topicNames)
	if err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			tm.logger.Error("failed to delete topic",
				"topic", result.Topic,
				"error", result.Error,
			)
			return fmt.Errorf("failed to delete topic %s: %w", result.Topic, result.Error)
		}

		tm.logger.Info("topic deleted", "topic", result.Topic)
	}

	return nil
}

// DeleteTopic deletes a single topic
func (tm *TopicManager) DeleteTopic(ctx context.Context, topicName string) error {
	return tm.DeleteTopics(ctx, []string{topicName})
}

// ListTopics lists all topics
func (tm *TopicManager) ListTopics(ctx context.Context) ([]string, error) {
	metadata, err := tm.adminClient.GetMetadata(nil, false, int(5*time.Second.Milliseconds()))
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	topics := make([]string, 0, len(metadata.Topics))
	for topicName := range metadata.Topics {
		topics = append(topics, topicName)
	}

	return topics, nil
}

// DescribeTopic describes a topic (partitions, replicas, etc.)
func (tm *TopicManager) DescribeTopic(ctx context.Context, topicName string) (*TopicMetadata, error) {
	metadata, err := tm.adminClient.GetMetadata(&topicName, false, int(5*time.Second.Milliseconds()))
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	topicMetadata, ok := metadata.Topics[topicName]
	if !ok {
		return nil, fmt.Errorf("topic %s not found", topicName)
	}

	partitions := make([]PartitionMetadata, len(topicMetadata.Partitions))
	for i, partition := range topicMetadata.Partitions {
		replicas := make([]int32, len(partition.Replicas))
		for j, replica := range partition.Replicas {
			replicas[j] = replica
		}

		isrs := make([]int32, len(partition.Isrs))
		for j, isr := range partition.Isrs {
			isrs[j] = isr
		}

		partitions[i] = PartitionMetadata{
			ID:       partition.ID,
			Leader:   partition.Leader,
			Replicas: replicas,
			ISRs:     isrs,
		}
	}

	return &TopicMetadata{
		Name:       topicName,
		Partitions: partitions,
	}, nil
}

// EnsureStandardTopics ensures all standard DictaMesh topics exist
func (tm *TopicManager) EnsureStandardTopics(ctx context.Context) error {
	topics := GetStandardTopics(tm.config)
	tm.logger.Info("ensuring standard topics exist", "count", len(topics))
	return tm.CreateTopics(ctx, topics)
}

// TopicExists checks if a topic exists
func (tm *TopicManager) TopicExists(ctx context.Context, topicName string) (bool, error) {
	topics, err := tm.ListTopics(ctx)
	if err != nil {
		return false, err
	}

	for _, topic := range topics {
		if topic == topicName {
			return true, nil
		}
	}

	return false, nil
}

// Close closes the topic manager
func (tm *TopicManager) Close() {
	tm.adminClient.Close()
	tm.logger.Info("Kafka admin client closed")
}

// TopicMetadata represents metadata about a topic
type TopicMetadata struct {
	Name       string
	Partitions []PartitionMetadata
}

// PartitionMetadata represents metadata about a partition
type PartitionMetadata struct {
	ID       int32
	Leader   int32
	Replicas []int32
	ISRs     []int32 // In-Sync Replicas
}
