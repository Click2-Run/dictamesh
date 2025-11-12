// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package connector

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes"
)

// ClusterClient wraps Kubernetes client-go clients for a single cluster
type ClusterClient struct {
	// Config is the cluster configuration
	Config kubernetes.ClusterConfig

	// Clientset is the typed Kubernetes clientset
	Clientset kubernetes.Interface

	// DynamicClient is the dynamic client for custom resources
	DynamicClient dynamic.Interface

	// RestConfig is the REST configuration
	RestConfig *rest.Config

	// mu protects the client state
	mu sync.RWMutex

	// connected indicates if the client is connected
	connected bool
}

// NewClusterClient creates a new cluster client
func NewClusterClient(config kubernetes.ClusterConfig) (*ClusterClient, error) {
	client := &ClusterClient{
		Config: config,
	}

	if err := client.Connect(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to cluster %s: %w", config.ID, err)
	}

	return client, nil
}

// Connect establishes connection to the Kubernetes cluster
func (c *ClusterClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build REST config based on auth method
	restConfig, err := c.buildRestConfig()
	if err != nil {
		return fmt.Errorf("failed to build rest config: %w", err)
	}

	// Apply QPS and burst settings
	restConfig.QPS = c.Config.QPS
	restConfig.Burst = c.Config.Burst
	restConfig.Timeout = c.Config.Timeout

	// Create clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	c.RestConfig = restConfig
	c.Clientset = clientset
	c.DynamicClient = dynamicClient
	c.connected = true

	return nil
}

// buildRestConfig builds Kubernetes REST configuration based on auth method
func (c *ClusterClient) buildRestConfig() (*rest.Config, error) {
	switch c.Config.AuthMethod {
	case kubernetes.AuthMethodKubeconfig:
		return c.buildKubeconfigRestConfig()

	case kubernetes.AuthMethodInCluster:
		return c.buildInClusterRestConfig()

	case kubernetes.AuthMethodToken, kubernetes.AuthMethodServiceAccount:
		return c.buildTokenRestConfig()

	default:
		return nil, fmt.Errorf("unsupported auth method: %s", c.Config.AuthMethod)
	}
}

// buildKubeconfigRestConfig builds REST config from kubeconfig file
func (c *ClusterClient) buildKubeconfigRestConfig() (*rest.Config, error) {
	// Check if kubeconfig file exists
	if _, err := os.Stat(c.Config.KubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig file not found: %s", c.Config.KubeconfigPath)
	}

	// Load kubeconfig
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: c.Config.KubeconfigPath,
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if c.Config.KubeconfigContext != "" {
		configOverrides.CurrentContext = c.Config.KubeconfigContext
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return restConfig, nil
}

// buildInClusterRestConfig builds REST config for in-cluster authentication
func (c *ClusterClient) buildInClusterRestConfig() (*rest.Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	return restConfig, nil
}

// buildTokenRestConfig builds REST config using bearer token
func (c *ClusterClient) buildTokenRestConfig() (*rest.Config, error) {
	if c.Config.APIServerURL == "" {
		return nil, fmt.Errorf("api_server_url is required for token auth")
	}

	if c.Config.ServiceAccountToken == "" {
		return nil, fmt.Errorf("service_account_token is required for token auth")
	}

	restConfig := &rest.Config{
		Host:        c.Config.APIServerURL,
		BearerToken: c.Config.ServiceAccountToken,
	}

	// Apply TLS configuration
	if c.Config.TLSConfig.Insecure {
		restConfig.TLSClientConfig = rest.TLSClientConfig{
			Insecure: true,
		}
	} else {
		tlsConfig := rest.TLSClientConfig{}

		if c.Config.TLSConfig.CAFile != "" {
			tlsConfig.CAFile = c.Config.TLSConfig.CAFile
		}
		if len(c.Config.TLSConfig.CAData) > 0 {
			tlsConfig.CAData = c.Config.TLSConfig.CAData
		}

		if c.Config.TLSConfig.CertFile != "" {
			tlsConfig.CertFile = c.Config.TLSConfig.CertFile
		}
		if len(c.Config.TLSConfig.CertData) > 0 {
			tlsConfig.CertData = c.Config.TLSConfig.CertData
		}

		if c.Config.TLSConfig.KeyFile != "" {
			tlsConfig.KeyFile = c.Config.TLSConfig.KeyFile
		}
		if len(c.Config.TLSConfig.KeyData) > 0 {
			tlsConfig.KeyData = c.Config.TLSConfig.KeyData
		}

		restConfig.TLSClientConfig = tlsConfig
	}

	return restConfig, nil
}

// IsConnected returns true if the client is connected
func (c *ClusterClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// HealthCheck performs a health check on the cluster
func (c *ClusterClient) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("client not connected")
	}

	// Try to get server version as a simple health check
	_, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the client connection
func (c *ClusterClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	c.Clientset = nil
	c.DynamicClient = nil
	c.RestConfig = nil

	return nil
}

// GetClusterInfo retrieves information about the cluster
func (c *ClusterClient) GetClusterInfo(ctx context.Context) (*kubernetes.ClusterInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	// Get server version
	version, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	// Get nodes
	nodes, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get namespaces
	namespaces, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return &kubernetes.ClusterInfo{
		ID:             c.Config.ID,
		Name:           c.Config.Name,
		Environment:    c.Config.Environment,
		Region:         c.Config.Region,
		Version:        version.GitVersion,
		Status:         "connected",
		NodeCount:      len(nodes.Items),
		NamespaceCount: len(namespaces.Items),
		LastSeen:       time.Now(),
	}, nil
}
