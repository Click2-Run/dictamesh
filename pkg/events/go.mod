// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

module github.com/click2-run/dictamesh/pkg/events

go 1.21

require (
	github.com/confluentinc/confluent-kafka-go/v2 v2.3.0
	github.com/google/uuid v1.6.0
	github.com/linkedin/goavro/v2 v2.12.0
	github.com/click2-run/dictamesh/pkg/observability v0.0.0
	go.uber.org/zap v1.26.0
)

replace github.com/click2-run/dictamesh/pkg/observability => ../observability
