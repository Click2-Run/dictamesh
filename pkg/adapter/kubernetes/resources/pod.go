// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda

package resources

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/click2-run/dictamesh/pkg/adapter"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes"
	"github.com/click2-run/dictamesh/pkg/adapter/kubernetes/connector"
)

// PodManager manages Kubernetes Pod resources
type PodManager struct {
	*BaseResourceManager
}

// NewPodManager creates a new Pod resource manager
func NewPodManager(clusterClient *connector.ClusterClient) *PodManager {
	return &PodManager{
		BaseResourceManager: NewBaseResourceManager(
			"pod",
			schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			clusterClient,
		),
	}
}

// Get retrieves a Pod by namespace and name
func (m *PodManager) Get(
	ctx context.Context,
	cluster, namespace, name string,
) (*adapter.Resource, error) {
	pod, err := m.clusterClient.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	return m.toResource(pod, cluster), nil
}

// List lists Pods in a namespace
func (m *PodManager) List(
	ctx context.Context,
	cluster, namespace string,
	opts *adapter.ListOptions,
) (*adapter.ResourceList, error) {
	listOpts := metav1.ListOptions{}

	if opts != nil && opts.PageSize > 0 {
		listOpts.Limit = int64(opts.PageSize)
	}

	podList, err := m.clusterClient.Clientset.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	resources := make([]*adapter.Resource, 0, len(podList.Items))
	for i := range podList.Items {
		resources = append(resources, m.toResource(&podList.Items[i], cluster))
	}

	return &adapter.ResourceList{
		Items:    resources,
		Total:    len(resources),
		Page:     1,
		PageSize: len(resources),
		HasMore:  false,
	}, nil
}

// Delete deletes a Pod
func (m *PodManager) Delete(
	ctx context.Context,
	cluster, namespace, name string,
) error {
	err := m.clusterClient.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	return nil
}

// toResource converts a Kubernetes Pod to an adapter Resource
func (m *PodManager) toResource(pod *corev1.Pod, cluster string) *adapter.Resource {
	attributes := map[string]interface{}{
		"cluster":    cluster,
		"namespace":  pod.Namespace,
		"name":       pod.Name,
		"uid":        string(pod.UID),
		"labels":     pod.Labels,
		"annotations": pod.Annotations,
		"phase":      string(pod.Status.Phase),
		"node_name":  pod.Spec.NodeName,
		"pod_ip":     pod.Status.PodIP,
		"host_ip":    pod.Status.HostIP,
	}

	if pod.Status.StartTime != nil {
		attributes["start_time"] = pod.Status.StartTime.Time
	}

	// Add container information
	containers := make([]map[string]interface{}, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		containers = append(containers, map[string]interface{}{
			"name":  container.Name,
			"image": container.Image,
		})
	}
	attributes["containers"] = containers

	// Create resource ID
	resourceID := fmt.Sprintf("%s:%s:%s", cluster, pod.Namespace, pod.Name)

	return &adapter.Resource{
		ID:         resourceID,
		Type:       "kubernetes.pod",
		Attributes: attributes,
		Metadata: &adapter.ResourceMetadata{
			CreatedAt: pod.CreationTimestamp.Time,
			Source:    cluster,
			Version:   pod.ResourceVersion,
		},
		Raw: pod,
	}
}

// GetRelationships discovers relationships for a Pod
func (m *PodManager) GetRelationships(
	ctx context.Context,
	cluster, namespace, name string,
) ([]*kubernetes.Relationship, error) {
	pod, err := m.clusterClient.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	relationships := make([]*kubernetes.Relationship, 0)

	// Owner references (e.g., ReplicaSet, StatefulSet, DaemonSet, Job)
	for _, owner := range pod.OwnerReferences {
		relationships = append(relationships, &kubernetes.Relationship{
			Type:       "owned_by",
			SourceType: "pod",
			SourceID:   fmt.Sprintf("%s:%s:%s", cluster, namespace, name),
			TargetType: owner.Kind,
			TargetID:   fmt.Sprintf("%s:%s:%s", cluster, namespace, owner.Name),
			Metadata: map[string]interface{}{
				"controller": owner.Controller != nil && *owner.Controller,
			},
		})
	}

	// Node relationship
	if pod.Spec.NodeName != "" {
		relationships = append(relationships, &kubernetes.Relationship{
			Type:       "runs_on",
			SourceType: "pod",
			SourceID:   fmt.Sprintf("%s:%s:%s", cluster, namespace, name),
			TargetType: "node",
			TargetID:   fmt.Sprintf("%s::%s", cluster, pod.Spec.NodeName),
		})
	}

	return relationships, nil
}
