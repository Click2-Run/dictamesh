// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package resources

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/click2-run/dictamesh/pkg/adapter"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes/connector"
)

// BaseResourceManager provides common functionality for resource managers
type BaseResourceManager struct {
	// resourceType is the resource type (e.g., "pod", "deployment")
	resourceType string

	// gvk is the GroupVersionKind for this resource
	gvk schema.GroupVersionKind

	// clusterClient is the Kubernetes cluster client
	clusterClient *connector.ClusterClient
}

// NewBaseResourceManager creates a new base resource manager
func NewBaseResourceManager(
	resourceType string,
	gvk schema.GroupVersionKind,
	clusterClient *connector.ClusterClient,
) *BaseResourceManager {
	return &BaseResourceManager{
		resourceType:  resourceType,
		gvk:           gvk,
		clusterClient: clusterClient,
	}
}

// GetResourceType returns the resource type
func (b *BaseResourceManager) GetResourceType() string {
	return b.resourceType
}

// GetGroupVersionKind returns the GVK for this resource
func (b *BaseResourceManager) GetGroupVersionKind() schema.GroupVersionKind {
	return b.gvk
}

// Get retrieves a resource by namespace and name
func (b *BaseResourceManager) Get(
	ctx context.Context,
	cluster, namespace, name string,
) (*adapter.Resource, error) {
	// To be implemented by specific resource managers
	return nil, nil
}

// List lists resources in a namespace
func (b *BaseResourceManager) List(
	ctx context.Context,
	cluster, namespace string,
	opts *adapter.ListOptions,
) (*adapter.ResourceList, error) {
	// To be implemented by specific resource managers
	return nil, nil
}

// Create creates a new resource
func (b *BaseResourceManager) Create(
	ctx context.Context,
	cluster string,
	resource *adapter.Resource,
) (*adapter.Resource, error) {
	// To be implemented by specific resource managers
	return nil, nil
}

// Update updates an existing resource
func (b *BaseResourceManager) Update(
	ctx context.Context,
	cluster string,
	resource *adapter.Resource,
) (*adapter.Resource, error) {
	// To be implemented by specific resource managers
	return nil, nil
}

// Delete deletes a resource
func (b *BaseResourceManager) Delete(
	ctx context.Context,
	cluster, namespace, name string,
) error {
	// To be implemented by specific resource managers
	return nil
}

// GetRelationships discovers relationships for this resource
func (b *BaseResourceManager) GetRelationships(
	ctx context.Context,
	cluster, namespace, name string,
) ([]*kubernetes.Relationship, error) {
	// To be implemented by specific resource managers
	return nil, nil
}
