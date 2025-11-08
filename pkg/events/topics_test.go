// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"testing"
)

func TestTopicConstants(t *testing.T) {
	// Test that all topic constants are defined and non-empty
	topics := []string{
		TopicEntityChanged,
		TopicEntityRead,
		TopicRelationshipChanged,
		TopicSchemaChanged,
		TopicCacheInvalidation,
		TopicSystemEvents,
		TopicHealthEvents,
		TopicDeadLetter,
	}

	for _, topic := range topics {
		if topic == "" {
			t.Error("Topic constant should not be empty")
		}
	}

	// Check for uniqueness
	seen := make(map[string]bool)
	for _, topic := range topics {
		if seen[topic] {
			t.Errorf("Duplicate topic constant: %s", topic)
		}
		seen[topic] = true
	}
}

func TestGetStandardTopics(t *testing.T) {
	cfg := DefaultConfig()
	topics := GetStandardTopics(cfg)

	if len(topics) == 0 {
		t.Fatal("GetStandardTopics should return at least one topic")
	}

	// Check that all topics have required fields
	for _, topic := range topics {
		if topic.Name == "" {
			t.Error("Topic name should not be empty")
		}

		if topic.Partitions <= 0 {
			t.Error("Topic partitions should be positive")
		}

		if topic.ReplicationFactor <= 0 {
			t.Error("Topic replication factor should be positive")
		}

		if topic.RetentionMs <= 0 {
			t.Error("Topic retention should be positive")
		}

		if topic.CleanupPolicy == "" {
			t.Error("Topic cleanup policy should not be empty")
		}

		// Check cleanup policy is valid
		if topic.CleanupPolicy != "delete" && topic.CleanupPolicy != "compact" {
			t.Errorf("Invalid cleanup policy: %s", topic.CleanupPolicy)
		}
	}
}

func TestStandardTopicsHavePrefix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Topics.Prefix = "test."

	topics := GetStandardTopics(cfg)

	for _, topic := range topics {
		if len(topic.Name) < len(cfg.Topics.Prefix) {
			t.Errorf("Topic name %s should have prefix", topic.Name)
			continue
		}

		prefix := topic.Name[:len(cfg.Topics.Prefix)]
		if prefix != cfg.Topics.Prefix {
			t.Errorf("Topic %s should start with prefix %s", topic.Name, cfg.Topics.Prefix)
		}
	}
}

func TestStandardTopicsConfiguration(t *testing.T) {
	cfg := DefaultConfig()
	topics := GetStandardTopics(cfg)

	// Find specific topics and check their configuration
	var (
		entityTopic       *TopicConfiguration
		schemaTopic       *TopicConfiguration
		cacheInvalidation *TopicConfiguration
	)

	for i := range topics {
		switch {
		case topics[i].Name == cfg.TopicName(TopicEntityChanged):
			entityTopic = &topics[i]
		case topics[i].Name == cfg.TopicName(TopicSchemaChanged):
			schemaTopic = &topics[i]
		case topics[i].Name == cfg.TopicName(TopicCacheInvalidation):
			cacheInvalidation = &topics[i]
		}
	}

	// Check entity topic
	if entityTopic != nil {
		if entityTopic.CleanupPolicy != "delete" {
			t.Error("Entity topic should use delete cleanup policy")
		}
	}

	// Check schema topic - should use compaction
	if schemaTopic != nil {
		if schemaTopic.CleanupPolicy != "compact" {
			t.Error("Schema topic should use compact cleanup policy")
		}

		// Schema topic should have longer retention
		if schemaTopic.RetentionMs <= cfg.Topics.RetentionMs {
			t.Error("Schema topic should have longer retention than default")
		}
	}

	// Check cache invalidation - should have short retention
	if cacheInvalidation != nil {
		if cacheInvalidation.RetentionMs >= cfg.Topics.RetentionMs {
			t.Error("Cache invalidation topic should have shorter retention")
		}
	}
}

func TestTopicConfigurationFields(t *testing.T) {
	topic := TopicConfiguration{
		Name:              "test.topic",
		Partitions:        12,
		ReplicationFactor: 3,
		RetentionMs:       86400000, // 1 day
		CleanupPolicy:     "delete",
	}

	if topic.Name != "test.topic" {
		t.Error("Topic name should be set correctly")
	}

	if topic.Partitions != 12 {
		t.Error("Topic partitions should be set correctly")
	}

	if topic.ReplicationFactor != 3 {
		t.Error("Topic replication factor should be set correctly")
	}

	if topic.RetentionMs != 86400000 {
		t.Error("Topic retention should be set correctly")
	}

	if topic.CleanupPolicy != "delete" {
		t.Error("Topic cleanup policy should be set correctly")
	}
}
