// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/click2-run/dictamesh/pkg/adapter"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes/connector"
	"github.com/click2-run/dictamesh/pkg/observability"
)

const (
	// AdapterName is the name of the Kubernetes adapter
	AdapterName = "kubernetes"

	// AdapterVersion is the version of the Kubernetes adapter
	AdapterVersion = "1.0.0"
)

// Adapter implements the DictaMesh adapter interface for Kubernetes
type Adapter struct {
	// Configuration
	config *Config

	// Observability
	obs *observability.Observability

	// Kubernetes clients (per cluster)
	clusters   map[string]*connector.ClusterClient
	clustersMu sync.RWMutex

	// Resource managers
	resourceManagers map[string]ResourceManager

	// State
	initialized bool
	running     bool
	mu          sync.RWMutex
}

// NewAdapter creates a new Kubernetes adapter
func NewAdapter(obs *observability.Observability) (*Adapter, error) {
	if obs == nil {
		return nil, fmt.Errorf("observability instance is required")
	}

	return &Adapter{
		obs:              obs,
		clusters:         make(map[string]*connector.ClusterClient),
		resourceManagers: make(map[string]ResourceManager),
	}, nil
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return AdapterName
}

// Version returns the adapter version
func (a *Adapter) Version() string {
	return AdapterVersion
}

// Initialize initializes the adapter with the provided configuration
func (a *Adapter) Initialize(ctx context.Context, config adapter.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return fmt.Errorf("adapter already initialized")
	}

	a.obs.Logger().InfoContext(ctx, "initializing Kubernetes adapter")

	// Type assert to Kubernetes config
	k8sConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type: expected *kubernetes.Config")
	}

	// Validate configuration
	if err := k8sConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	a.config = k8sConfig

	// Connect to all configured clusters
	for _, clusterConfig := range k8sConfig.Clusters {
		if err := a.connectCluster(ctx, clusterConfig); err != nil {
			a.obs.Logger().ErrorContext(ctx, "failed to connect to cluster",
				"cluster", clusterConfig.ID,
				"error", err,
			)
			// Continue with other clusters instead of failing completely
			continue
		}

		a.obs.Logger().InfoContext(ctx, "connected to cluster",
			"cluster", clusterConfig.ID,
			"name", clusterConfig.Name,
		)
	}

	if len(a.clusters) == 0 {
		return fmt.Errorf("failed to connect to any clusters")
	}

	a.initialized = true

	a.obs.Logger().InfoContext(ctx, "Kubernetes adapter initialized",
		"clusters", len(a.clusters),
	)

	return nil
}

// connectCluster establishes connection to a Kubernetes cluster
func (a *Adapter) connectCluster(ctx context.Context, clusterConfig ClusterConfig) error {
	client, err := connector.NewClusterClient(clusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create cluster client: %w", err)
	}

	// Perform health check
	if err := client.HealthCheck(ctx); err != nil {
		client.Close()
		return fmt.Errorf("cluster health check failed: %w", err)
	}

	a.clustersMu.Lock()
	a.clusters[clusterConfig.ID] = client
	a.clustersMu.Unlock()

	return nil
}

// Health checks the health of the adapter and all cluster connections
func (a *Adapter) Health(ctx context.Context) (*adapter.HealthStatus, error) {
	a.mu.RLock()
	initialized := a.initialized
	a.mu.RUnlock()

	if !initialized {
		return &adapter.HealthStatus{
			Status:    adapter.HealthStatusUnhealthy,
			Message:   "adapter not initialized",
			Timestamp: time.Now(),
		}, nil
	}

	// Check health of all clusters
	a.clustersMu.RLock()
	clusterCount := len(a.clusters)
	a.clustersMu.RUnlock()

	healthyClusters := 0
	unhealthyClusters := 0

	a.clustersMu.RLock()
	for clusterID, client := range a.clusters {
		if err := client.HealthCheck(ctx); err != nil {
			a.obs.Logger().WarnContext(ctx, "cluster health check failed",
				"cluster", clusterID,
				"error", err,
			)
			unhealthyClusters++
		} else {
			healthyClusters++
		}
	}
	a.clustersMu.RUnlock()

	status := adapter.HealthStatusHealthy
	message := fmt.Sprintf("%d/%d clusters healthy", healthyClusters, clusterCount)

	if unhealthyClusters > 0 {
		if healthyClusters == 0 {
			status = adapter.HealthStatusUnhealthy
			message = "all clusters unhealthy"
		} else {
			status = adapter.HealthStatusDegraded
			message = fmt.Sprintf("%d/%d clusters unhealthy", unhealthyClusters, clusterCount)
		}
	}

	return &adapter.HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"total_clusters":     clusterCount,
			"healthy_clusters":   healthyClusters,
			"unhealthy_clusters": unhealthyClusters,
		},
	}, nil
}

// Shutdown gracefully shuts down the adapter
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.initialized {
		return nil
	}

	a.obs.Logger().InfoContext(ctx, "shutting down Kubernetes adapter")

	// Close all cluster connections
	a.clustersMu.Lock()
	for clusterID, client := range a.clusters {
		if err := client.Close(); err != nil {
			a.obs.Logger().ErrorContext(ctx, "failed to close cluster connection",
				"cluster", clusterID,
				"error", err,
			)
		}
	}
	a.clusters = make(map[string]*connector.ClusterClient)
	a.clustersMu.Unlock()

	a.initialized = false
	a.running = false

	a.obs.Logger().InfoContext(ctx, "Kubernetes adapter shut down")

	return nil
}

// GetCapabilities returns the capabilities supported by this adapter
func (a *Adapter) GetCapabilities() []adapter.Capability {
	capabilities := []adapter.Capability{
		adapter.CapabilityRead,
		adapter.CapabilityStream,
		adapter.CapabilityPagination,
	}

	if a.config != nil && a.config.EnableMutations {
		capabilities = append(capabilities, adapter.CapabilityWrite)
	}

	return capabilities
}

// AddCluster dynamically adds a new cluster to the adapter
func (a *Adapter) AddCluster(ctx context.Context, clusterConfig ClusterConfig) error {
	a.mu.RLock()
	initialized := a.initialized
	a.mu.RUnlock()

	if !initialized {
		return fmt.Errorf("adapter not initialized")
	}

	// Check if cluster already exists
	a.clustersMu.RLock()
	if _, exists := a.clusters[clusterConfig.ID]; exists {
		a.clustersMu.RUnlock()
		return fmt.Errorf("cluster %s already exists", clusterConfig.ID)
	}
	a.clustersMu.RUnlock()

	// Connect to the cluster
	if err := a.connectCluster(ctx, clusterConfig); err != nil {
		return fmt.Errorf("failed to add cluster: %w", err)
	}

	a.obs.Logger().InfoContext(ctx, "cluster added",
		"cluster", clusterConfig.ID,
		"name", clusterConfig.Name,
	)

	return nil
}

// RemoveCluster removes a cluster from the adapter
func (a *Adapter) RemoveCluster(ctx context.Context, clusterID string) error {
	a.clustersMu.Lock()
	defer a.clustersMu.Unlock()

	client, exists := a.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	// Close the cluster connection
	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to close cluster connection: %w", err)
	}

	delete(a.clusters, clusterID)

	a.obs.Logger().InfoContext(ctx, "cluster removed",
		"cluster", clusterID,
	)

	return nil
}

// ListClusters returns a list of all connected clusters
func (a *Adapter) ListClusters(ctx context.Context) ([]*ClusterInfo, error) {
	a.clustersMu.RLock()
	defer a.clustersMu.RUnlock()

	clusters := make([]*ClusterInfo, 0, len(a.clusters))

	for _, client := range a.clusters {
		info, err := client.GetClusterInfo(ctx)
		if err != nil {
			a.obs.Logger().WarnContext(ctx, "failed to get cluster info",
				"cluster", client.Config.ID,
				"error", err,
			)
			continue
		}
		clusters = append(clusters, info)
	}

	return clusters, nil
}

// GetCluster retrieves a cluster client by ID
func (a *Adapter) GetCluster(clusterID string) (*connector.ClusterClient, error) {
	a.clustersMu.RLock()
	defer a.clustersMu.RUnlock()

	client, exists := a.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	return client, nil
}
