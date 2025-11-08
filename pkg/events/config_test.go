// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package events

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	if len(cfg.BootstrapServers) == 0 {
		t.Error("Bootstrap servers should be configured")
	}

	if cfg.SchemaRegistryURL == "" {
		t.Error("Schema registry URL should be configured")
	}

	if cfg.Producer.ClientID == "" {
		t.Error("Producer client ID should be configured")
	}

	if cfg.Consumer.GroupID == "" {
		t.Error("Consumer group ID should be configured")
	}

	if cfg.Topics.Prefix == "" {
		t.Error("Topic prefix should be configured")
	}
}

func TestProductionConfig(t *testing.T) {
	cfg := ProductionConfig()

	if cfg == nil {
		t.Fatal("ProductionConfig should not return nil")
	}

	// Production should have multiple bootstrap servers
	if len(cfg.BootstrapServers) < 3 {
		t.Error("Production config should have at least 3 bootstrap servers")
	}

	// Production should have higher replication
	if cfg.Topics.ReplicationFactor < 3 {
		t.Error("Production config should have replication factor of at least 3")
	}

	// Production should use SASL_SSL
	if cfg.Security.Protocol != "SASL_SSL" {
		t.Errorf("Expected SASL_SSL, got %s", cfg.Security.Protocol)
	}

	// Production should disable auto-commit
	if cfg.Consumer.EnableAutoCommit {
		t.Error("Production config should disable auto-commit")
	}

	// Production should enable idempotence
	if !cfg.Producer.EnableIdempotence {
		t.Error("Production config should enable idempotence")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty bootstrap servers",
			cfg: &Config{
				BootstrapServers: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid producer acks",
			cfg: &Config{
				BootstrapServers: []string{"localhost:9092"},
				Producer: ProducerConfig{
					Acks: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid consumer auto offset reset",
			cfg: &Config{
				BootstrapServers: []string{"localhost:9092"},
				Producer: ProducerConfig{
					Acks: "all",
				},
				Consumer: ConsumerConfig{
					AutoOffsetReset: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid partitions",
			cfg: &Config{
				BootstrapServers: []string{"localhost:9092"},
				Producer: ProducerConfig{
					Acks: "all",
				},
				Consumer: ConsumerConfig{
					AutoOffsetReset: "earliest",
				},
				Topics: TopicConfig{
					NumPartitions: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid replication factor",
			cfg: &Config{
				BootstrapServers: []string{"localhost:9092"},
				Producer: ProducerConfig{
					Acks: "all",
				},
				Consumer: ConsumerConfig{
					AutoOffsetReset: "earliest",
				},
				Topics: TopicConfig{
					NumPartitions:     12,
					ReplicationFactor: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBootstrapServersString(t *testing.T) {
	cfg := &Config{
		BootstrapServers: []string{"server1:9092", "server2:9092", "server3:9092"},
	}

	expected := "server1:9092,server2:9092,server3:9092"
	result := cfg.BootstrapServersString()

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestTopicName(t *testing.T) {
	cfg := &Config{
		Topics: TopicConfig{
			Prefix: "dictamesh.",
		},
	}

	result := cfg.TopicName("entity.changed")
	expected := "dictamesh.entity.changed"

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with empty prefix
	cfg.Topics.Prefix = ""
	result = cfg.TopicName("entity.changed")
	expected = "entity.changed"

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetProducerConfig(t *testing.T) {
	cfg := DefaultConfig()
	producerCfg := cfg.GetProducerConfig()

	if producerCfg == nil {
		t.Fatal("Producer config should not be nil")
	}

	// Check required fields
	if _, ok := producerCfg["bootstrap.servers"]; !ok {
		t.Error("Producer config should have bootstrap.servers")
	}

	if _, ok := producerCfg["client.id"]; !ok {
		t.Error("Producer config should have client.id")
	}

	if _, ok := producerCfg["acks"]; !ok {
		t.Error("Producer config should have acks")
	}
}

func TestGetConsumerConfig(t *testing.T) {
	cfg := DefaultConfig()
	consumerCfg := cfg.GetConsumerConfig()

	if consumerCfg == nil {
		t.Fatal("Consumer config should not be nil")
	}

	// Check required fields
	if _, ok := consumerCfg["bootstrap.servers"]; !ok {
		t.Error("Consumer config should have bootstrap.servers")
	}

	if _, ok := consumerCfg["group.id"]; !ok {
		t.Error("Consumer config should have group.id")
	}

	if _, ok := consumerCfg["auto.offset.reset"]; !ok {
		t.Error("Consumer config should have auto.offset.reset")
	}
}

func TestProducerConfigTimeouts(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Producer.RequestTimeout <= 0 {
		t.Error("Producer request timeout should be positive")
	}

	if cfg.Producer.RetryBackoff <= 0 {
		t.Error("Producer retry backoff should be positive")
	}
}

func TestConsumerConfigTimeouts(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Consumer.SessionTimeout <= 0 {
		t.Error("Consumer session timeout should be positive")
	}

	if cfg.Consumer.HeartbeatInterval <= 0 {
		t.Error("Consumer heartbeat interval should be positive")
	}

	if cfg.Consumer.MaxPollInterval <= 0 {
		t.Error("Consumer max poll interval should be positive")
	}

	// Heartbeat interval should be less than session timeout
	if cfg.Consumer.HeartbeatInterval >= cfg.Consumer.SessionTimeout {
		t.Error("Heartbeat interval should be less than session timeout")
	}
}

func TestTopicConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Topics.NumPartitions <= 0 {
		t.Error("Number of partitions should be positive")
	}

	if cfg.Topics.ReplicationFactor <= 0 {
		t.Error("Replication factor should be positive")
	}

	if cfg.Topics.RetentionMs <= 0 {
		t.Error("Retention should be positive")
	}

	// Check retention is at least 1 day
	minRetention := int64(24 * 60 * 60 * 1000)
	if cfg.Topics.RetentionMs < minRetention {
		t.Error("Retention should be at least 1 day")
	}
}
