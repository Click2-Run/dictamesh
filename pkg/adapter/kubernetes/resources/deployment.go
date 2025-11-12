// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package resources

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/click2-run/dictamesh/pkg/adapter"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes/connector"
)

// DeploymentManager manages Kubernetes Deployment resources
type DeploymentManager struct {
	*BaseResourceManager
}

// NewDeploymentManager creates a new Deployment resource manager
func NewDeploymentManager(clusterClient *connector.ClusterClient) *DeploymentManager {
	return &DeploymentManager{
		BaseResourceManager: NewBaseResourceManager(
			"deployment",
			schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			clusterClient,
		),
	}
}

// Get retrieves a Deployment by namespace and name
func (m *DeploymentManager) Get(
	ctx context.Context,
	cluster, namespace, name string,
) (*adapter.Resource, error) {
	deployment, err := m.clusterClient.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return m.toResource(deployment, cluster), nil
}

// List lists Deployments in a namespace
func (m *DeploymentManager) List(
	ctx context.Context,
	cluster, namespace string,
	opts *adapter.ListOptions,
) (*adapter.ResourceList, error) {
	listOpts := metav1.ListOptions{}

	if opts != nil && opts.PageSize > 0 {
		listOpts.Limit = int64(opts.PageSize)
	}

	deploymentList, err := m.clusterClient.Clientset.AppsV1().Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	resources := make([]*adapter.Resource, 0, len(deploymentList.Items))
	for i := range deploymentList.Items {
		resources = append(resources, m.toResource(&deploymentList.Items[i], cluster))
	}

	return &adapter.ResourceList{
		Items:    resources,
		Total:    len(resources),
		Page:     1,
		PageSize: len(resources),
		HasMore:  false,
	}, nil
}

// Delete deletes a Deployment
func (m *DeploymentManager) Delete(
	ctx context.Context,
	cluster, namespace, name string,
) error {
	err := m.clusterClient.Clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	return nil
}

// toResource converts a Kubernetes Deployment to an adapter Resource
func (m *DeploymentManager) toResource(deployment *appsv1.Deployment, cluster string) *adapter.Resource {
	var replicas int32
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	attributes := map[string]interface{}{
		"cluster":            cluster,
		"namespace":          deployment.Namespace,
		"name":               deployment.Name,
		"uid":                string(deployment.UID),
		"labels":             deployment.Labels,
		"annotations":        deployment.Annotations,
		"replicas":           replicas,
		"available_replicas": deployment.Status.AvailableReplicas,
		"ready_replicas":     deployment.Status.ReadyReplicas,
		"updated_replicas":   deployment.Status.UpdatedReplicas,
	}

	// Add selector
	if deployment.Spec.Selector != nil {
		attributes["selector"] = deployment.Spec.Selector.MatchLabels
	}

	// Add strategy
	attributes["strategy_type"] = string(deployment.Spec.Strategy.Type)

	// Create resource ID
	resourceID := fmt.Sprintf("%s:%s:%s", cluster, deployment.Namespace, deployment.Name)

	return &adapter.Resource{
		ID:         resourceID,
		Type:       "kubernetes.deployment",
		Attributes: attributes,
		Metadata: &adapter.ResourceMetadata{
			CreatedAt: deployment.CreationTimestamp.Time,
			Source:    cluster,
			Version:   deployment.ResourceVersion,
		},
		Raw: deployment,
	}
}

// GetRelationships discovers relationships for a Deployment
func (m *DeploymentManager) GetRelationships(
	ctx context.Context,
	cluster, namespace, name string,
) ([]*kubernetes.Relationship, error) {
	deployment, err := m.clusterClient.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	relationships := make([]*kubernetes.Relationship, 0)

	// Find ReplicaSets owned by this deployment
	if deployment.Spec.Selector != nil {
		labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
		replicaSets, err := m.clusterClient.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err == nil {
			for _, rs := range replicaSets.Items {
				// Check if this deployment owns the ReplicaSet
				for _, owner := range rs.OwnerReferences {
					if owner.UID == deployment.UID {
						relationships = append(relationships, &kubernetes.Relationship{
							Type:       "owns",
							SourceType: "deployment",
							SourceID:   fmt.Sprintf("%s:%s:%s", cluster, namespace, name),
							TargetType: "replicaset",
							TargetID:   fmt.Sprintf("%s:%s:%s", cluster, namespace, rs.Name),
						})
						break
					}
				}
			}
		}
	}

	return relationships, nil
}
