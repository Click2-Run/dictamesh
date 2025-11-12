// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package kubernetes

import (
	"errors"
	"fmt"
	"time"

	"github.com/click2-run/dictamesh/pkg/adapter"
)

// Config represents the Kubernetes adapter configuration
type Config struct {
	// Clusters to connect to
	Clusters []ClusterConfig `json:"clusters" yaml:"clusters"`

	// DefaultNamespace for operations
	DefaultNamespace string `json:"default_namespace" yaml:"default_namespace"`

	// WatchResources specifies which resource types to watch (empty = all)
	WatchResources []string `json:"watch_resources" yaml:"watch_resources"`

	// EnableCache enables resource caching
	EnableCache bool `json:"enable_cache" yaml:"enable_cache"`

	// CacheTTL is the cache time-to-live
	CacheTTL time.Duration `json:"cache_ttl" yaml:"cache_ttl"`

	// EnableRBAC enables RBAC enforcement
	EnableRBAC bool `json:"enable_rbac" yaml:"enable_rbac"`

	// EnableRelationships enables relationship discovery
	EnableRelationships bool `json:"enable_relationships" yaml:"enable_relationships"`

	// ResyncPeriod for informers
	ResyncPeriod time.Duration `json:"resync_period" yaml:"resync_period"`

	// WorkerPoolSize for event processing
	WorkerPoolSize int `json:"worker_pool_size" yaml:"worker_pool_size"`

	// EnableMutations enables create, update, delete operations
	EnableMutations bool `json:"enable_mutations" yaml:"enable_mutations"`

	// EnableCustomResources enables custom resource support
	EnableCustomResources bool `json:"enable_custom_resources" yaml:"enable_custom_resources"`

	// CustomResourceDefinitions to watch
	CustomResourceDefinitions []CRDConfig `json:"custom_resource_definitions" yaml:"custom_resource_definitions"`
}

// NewConfig creates a new Kubernetes adapter configuration from a map
func NewConfig(data map[string]interface{}) (*Config, error) {
	cfg := &Config{
		DefaultNamespace:      "default",
		EnableCache:           true,
		CacheTTL:              5 * time.Minute,
		EnableRBAC:            false,
		EnableRelationships:   true,
		ResyncPeriod:          10 * time.Minute,
		WorkerPoolSize:        10,
		EnableMutations:       false,
		EnableCustomResources: false,
	}

	// Parse clusters configuration
	if clustersData, ok := data["clusters"].([]interface{}); ok {
		for _, clusterData := range clustersData {
			if clusterMap, ok := clusterData.(map[string]interface{}); ok {
				cluster, err := parseClusterConfig(clusterMap)
				if err != nil {
					return nil, fmt.Errorf("failed to parse cluster config: %w", err)
				}
				cfg.Clusters = append(cfg.Clusters, cluster)
			}
		}
	}

	// Parse other configuration options
	if defaultNs, ok := data["default_namespace"].(string); ok {
		cfg.DefaultNamespace = defaultNs
	}

	if watchResources, ok := data["watch_resources"].([]interface{}); ok {
		for _, res := range watchResources {
			if resStr, ok := res.(string); ok {
				cfg.WatchResources = append(cfg.WatchResources, resStr)
			}
		}
	}

	if enableCache, ok := data["enable_cache"].(bool); ok {
		cfg.EnableCache = enableCache
	}

	if cacheTTL, ok := data["cache_ttl"].(string); ok {
		if d, err := time.ParseDuration(cacheTTL); err == nil {
			cfg.CacheTTL = d
		}
	}

	if enableRBAC, ok := data["enable_rbac"].(bool); ok {
		cfg.EnableRBAC = enableRBAC
	}

	if enableRelationships, ok := data["enable_relationships"].(bool); ok {
		cfg.EnableRelationships = enableRelationships
	}

	if resyncPeriod, ok := data["resync_period"].(string); ok {
		if d, err := time.ParseDuration(resyncPeriod); err == nil {
			cfg.ResyncPeriod = d
		}
	}

	if workerPoolSize, ok := data["worker_pool_size"].(int); ok {
		cfg.WorkerPoolSize = workerPoolSize
	} else if workerPoolSize, ok := data["worker_pool_size"].(float64); ok {
		cfg.WorkerPoolSize = int(workerPoolSize)
	}

	if enableMutations, ok := data["enable_mutations"].(bool); ok {
		cfg.EnableMutations = enableMutations
	}

	if enableCustomResources, ok := data["enable_custom_resources"].(bool); ok {
		cfg.EnableCustomResources = enableCustomResources
	}

	return cfg, nil
}

// parseClusterConfig parses a cluster configuration from a map
func parseClusterConfig(data map[string]interface{}) (ClusterConfig, error) {
	cluster := ClusterConfig{
		AuthMethod: AuthMethodKubeconfig,
		QPS:        50.0,
		Burst:      100,
		Timeout:    30 * time.Second,
	}

	// Required fields
	if id, ok := data["id"].(string); ok {
		cluster.ID = id
	} else {
		return cluster, errors.New("cluster id is required")
	}

	if name, ok := data["name"].(string); ok {
		cluster.Name = name
	} else {
		cluster.Name = cluster.ID
	}

	// Optional fields
	if env, ok := data["environment"].(string); ok {
		cluster.Environment = env
	}

	if region, ok := data["region"].(string); ok {
		cluster.Region = region
	}

	if authMethod, ok := data["auth_method"].(string); ok {
		cluster.AuthMethod = AuthMethod(authMethod)
	}

	if kubeconfigPath, ok := data["kubeconfig_path"].(string); ok {
		cluster.KubeconfigPath = kubeconfigPath
	}

	if kubeconfigContext, ok := data["kubeconfig_context"].(string); ok {
		cluster.KubeconfigContext = kubeconfigContext
	}

	if token, ok := data["service_account_token"].(string); ok {
		cluster.ServiceAccountToken = token
	}

	if apiServerURL, ok := data["api_server_url"].(string); ok {
		cluster.APIServerURL = apiServerURL
	}

	if qps, ok := data["qps"].(float64); ok {
		cluster.QPS = float32(qps)
	}

	if burst, ok := data["burst"].(int); ok {
		cluster.Burst = burst
	} else if burst, ok := data["burst"].(float64); ok {
		cluster.Burst = int(burst)
	}

	if timeout, ok := data["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeout); err == nil {
			cluster.Timeout = d
		}
	}

	// Parse TLS config
	if tlsData, ok := data["tls_config"].(map[string]interface{}); ok {
		cluster.TLSConfig = parseTLSConfig(tlsData)
	}

	return cluster, nil
}

// parseTLSConfig parses TLS configuration from a map
func parseTLSConfig(data map[string]interface{}) TLSConfig {
	tls := TLSConfig{}

	if insecure, ok := data["insecure"].(bool); ok {
		tls.Insecure = insecure
	}

	if caFile, ok := data["ca_file"].(string); ok {
		tls.CAFile = caFile
	}

	if certFile, ok := data["cert_file"].(string); ok {
		tls.CertFile = certFile
	}

	if keyFile, ok := data["key_file"].(string); ok {
		tls.KeyFile = keyFile
	}

	return tls
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Clusters) == 0 {
		return errors.New("at least one cluster must be configured")
	}

	for i, cluster := range c.Clusters {
		if cluster.ID == "" {
			return fmt.Errorf("cluster[%d]: id is required", i)
		}

		if cluster.Name == "" {
			return fmt.Errorf("cluster[%d]: name is required", i)
		}

		// Validate auth method specific requirements
		switch cluster.AuthMethod {
		case AuthMethodKubeconfig:
			if cluster.KubeconfigPath == "" {
				return fmt.Errorf("cluster[%d]: kubeconfig_path is required for kubeconfig auth", i)
			}
		case AuthMethodServiceAccount, AuthMethodToken:
			if cluster.ServiceAccountToken == "" {
				return fmt.Errorf("cluster[%d]: service_account_token is required for token auth", i)
			}
			if cluster.APIServerURL == "" {
				return fmt.Errorf("cluster[%d]: api_server_url is required for token auth", i)
			}
		case AuthMethodInCluster:
			// No additional validation needed
		default:
			return fmt.Errorf("cluster[%d]: unsupported auth method: %s", i, cluster.AuthMethod)
		}

		// Validate timeouts and limits
		if cluster.QPS <= 0 {
			return fmt.Errorf("cluster[%d]: qps must be greater than 0", i)
		}

		if cluster.Burst <= 0 {
			return fmt.Errorf("cluster[%d]: burst must be greater than 0", i)
		}

		if cluster.Timeout <= 0 {
			return fmt.Errorf("cluster[%d]: timeout must be greater than 0", i)
		}
	}

	if c.CacheTTL < 0 {
		return errors.New("cache_ttl cannot be negative")
	}

	if c.ResyncPeriod < 0 {
		return errors.New("resync_period cannot be negative")
	}

	if c.WorkerPoolSize <= 0 {
		return errors.New("worker_pool_size must be greater than 0")
	}

	return nil
}

// GetString retrieves a string configuration value
func (c *Config) GetString(key string) (string, error) {
	switch key {
	case "default_namespace":
		return c.DefaultNamespace, nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}

// GetInt retrieves an integer configuration value
func (c *Config) GetInt(key string) (int, error) {
	switch key {
	case "worker_pool_size":
		return c.WorkerPoolSize, nil
	default:
		return 0, fmt.Errorf("unknown configuration key: %s", key)
	}
}

// GetBool retrieves a boolean configuration value
func (c *Config) GetBool(key string) (bool, error) {
	switch key {
	case "enable_cache":
		return c.EnableCache, nil
	case "enable_rbac":
		return c.EnableRBAC, nil
	case "enable_relationships":
		return c.EnableRelationships, nil
	case "enable_mutations":
		return c.EnableMutations, nil
	case "enable_custom_resources":
		return c.EnableCustomResources, nil
	default:
		return false, fmt.Errorf("unknown configuration key: %s", key)
	}
}

// GetDuration retrieves a duration configuration value
func (c *Config) GetDuration(key string) (time.Duration, error) {
	switch key {
	case "cache_ttl":
		return c.CacheTTL, nil
	case "resync_period":
		return c.ResyncPeriod, nil
	default:
		return 0, fmt.Errorf("unknown configuration key: %s", key)
	}
}

// Ensure Config implements adapter.Config
var _ adapter.Config = (*Config)(nil)
