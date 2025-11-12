// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

module github.com/click2-run/dictamesh/pkg/adapter/kubernetes

go 1.21

require (
	github.com/click2-run/dictamesh/pkg/adapter v0.0.0
	github.com/click2-run/dictamesh/pkg/observability v0.0.0
	k8s.io/api v0.28.4
	k8s.io/apimachinery v0.28.4
	k8s.io/client-go v0.28.4
)

replace github.com/click2-run/dictamesh/pkg/adapter => ../

replace github.com/click2-run/dictamesh/pkg/observability => ../../observability
