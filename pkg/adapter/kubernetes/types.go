// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package kubernetes

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AuthMethod represents authentication methods for Kubernetes clusters
type AuthMethod string

const (
	// AuthMethodKubeconfig uses kubeconfig file for authentication
	AuthMethodKubeconfig AuthMethod = "kubeconfig"

	// AuthMethodServiceAccount uses service account token
	AuthMethodServiceAccount AuthMethod = "service_account"

	// AuthMethodToken uses bearer token
	AuthMethodToken AuthMethod = "token"

	// AuthMethodOIDC uses OIDC authentication
	AuthMethodOIDC AuthMethod = "oidc"

	// AuthMethodInCluster uses in-cluster service account
	AuthMethodInCluster AuthMethod = "in_cluster"
)

// ClusterConfig represents configuration for a single Kubernetes cluster
type ClusterConfig struct {
	// ID is the unique cluster identifier
	ID string `json:"id" yaml:"id"`

	// Name is the human-readable cluster name
	Name string `json:"name" yaml:"name"`

	// Environment is the cluster environment (dev, staging, prod)
	Environment string `json:"environment" yaml:"environment"`

	// Region is the cluster region/zone
	Region string `json:"region" yaml:"region"`

	// AuthMethod specifies the authentication method
	AuthMethod AuthMethod `json:"auth_method" yaml:"auth_method"`

	// KubeconfigPath is the path to kubeconfig file (for kubeconfig auth)
	KubeconfigPath string `json:"kubeconfig_path" yaml:"kubeconfig_path"`

	// KubeconfigContext is the context to use (for kubeconfig auth)
	KubeconfigContext string `json:"kubeconfig_context" yaml:"kubeconfig_context"`

	// ServiceAccountToken is the service account token (for token auth)
	ServiceAccountToken string `json:"service_account_token" yaml:"service_account_token"`

	// APIServerURL is the Kubernetes API server URL
	APIServerURL string `json:"api_server_url" yaml:"api_server_url"`

	// TLSConfig contains TLS configuration
	TLSConfig TLSConfig `json:"tls_config" yaml:"tls_config"`

	// QPS is the queries per second limit
	QPS float32 `json:"qps" yaml:"qps"`

	// Burst is the burst limit for requests
	Burst int `json:"burst" yaml:"burst"`

	// Timeout is the request timeout
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// TLSConfig contains TLS configuration for cluster connection
type TLSConfig struct {
	// Insecure skips TLS verification (not recommended for production)
	Insecure bool `json:"insecure" yaml:"insecure"`

	// CAFile is the path to CA certificate file
	CAFile string `json:"ca_file" yaml:"ca_file"`

	// CertFile is the path to client certificate file
	CertFile string `json:"cert_file" yaml:"cert_file"`

	// KeyFile is the path to client key file
	KeyFile string `json:"key_file" yaml:"key_file"`

	// CAData is the CA certificate data (base64 encoded)
	CAData []byte `json:"ca_data" yaml:"ca_data"`

	// CertData is the client certificate data (base64 encoded)
	CertData []byte `json:"cert_data" yaml:"cert_data"`

	// KeyData is the client key data (base64 encoded)
	KeyData []byte `json:"key_data" yaml:"key_data"`
}

// ClusterInfo contains information about a connected cluster
type ClusterInfo struct {
	// ID is the cluster identifier
	ID string `json:"id"`

	// Name is the cluster name
	Name string `json:"name"`

	// Environment is the cluster environment
	Environment string `json:"environment"`

	// Region is the cluster region
	Region string `json:"region"`

	// Version is the Kubernetes version
	Version string `json:"version"`

	// Status is the cluster connection status
	Status string `json:"status"`

	// NodeCount is the number of nodes
	NodeCount int `json:"node_count"`

	// NamespaceCount is the number of namespaces
	NamespaceCount int `json:"namespace_count"`

	// LastSeen is when the cluster was last seen
	LastSeen time.Time `json:"last_seen"`
}

// CRDConfig represents configuration for a Custom Resource Definition
type CRDConfig struct {
	// Group is the API group
	Group string `json:"group" yaml:"group"`

	// Version is the API version
	Version string `json:"version" yaml:"version"`

	// Kind is the resource kind
	Kind string `json:"kind" yaml:"kind"`

	// Plural is the plural name
	Plural string `json:"plural" yaml:"plural"`

	// Namespaced indicates if the resource is namespaced
	Namespaced bool `json:"namespaced" yaml:"namespaced"`
}

// ResourceManager defines the interface for managing Kubernetes resources
type ResourceManager interface {
	// GetResourceType returns the resource type (e.g., "pod", "deployment")
	GetResourceType() string

	// GetGroupVersionKind returns the GVK for this resource
	GetGroupVersionKind() schema.GroupVersionKind
}

// Relationship represents a relationship between Kubernetes resources
type Relationship struct {
	// Type is the relationship type
	Type string `json:"type"`

	// SourceType is the source resource type
	SourceType string `json:"source_type"`

	// SourceID is the source resource ID
	SourceID string `json:"source_id"`

	// TargetType is the target resource type
	TargetType string `json:"target_type"`

	// TargetID is the target resource ID
	TargetID string `json:"target_id"`

	// Metadata contains additional relationship metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EventFilter represents filters for Kubernetes events
type EventFilter struct {
	// Clusters to filter by
	Clusters []string `json:"clusters,omitempty"`

	// Namespaces to filter by
	Namespaces []string `json:"namespaces,omitempty"`

	// ResourceTypes to filter by
	ResourceTypes []string `json:"resource_types,omitempty"`

	// LabelSelector for filtering by labels
	LabelSelector string `json:"label_selector,omitempty"`

	// FieldSelector for filtering by fields
	FieldSelector string `json:"field_selector,omitempty"`
}
