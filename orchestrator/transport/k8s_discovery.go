package transport

import (
	"context"
	"fmt"
	"time"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesDiscovery implements ServiceDiscovery using Kubernetes
type KubernetesDiscovery struct {
	client       kubernetes.Interface
	namespace    string
	serviceName  string
	labelSelector string
	stopCh       chan struct{}
}

// NewKubernetesDiscovery creates a new Kubernetes service discovery
func NewKubernetesDiscovery(namespace, serviceName, labelSelector string, inCluster bool) (*KubernetesDiscovery, error) {
	var config *rest.Config
	var err error
	
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", "")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s config: %w", err)
	}
	
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	
	return &KubernetesDiscovery{
		client:        client,
		namespace:     namespace,
		serviceName:   serviceName,
		labelSelector: labelSelector,
		stopCh:        make(chan struct{}),
	}, nil
}

// Discover starts discovering engines and returns a channel
func (kd *KubernetesDiscovery) Discover(ctx context.Context) (<-chan []*EngineInfo, error) {
	engineCh := make(chan []*EngineInfo, 10)
	
	go func() {
		defer close(engineCh)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		// Initial discovery
		engines, err := kd.discoverEngines()
		if err == nil {
			select {
			case engineCh <- engines:
			case <-ctx.Done():
				return
			case <-kd.stopCh:
				return
			}
		}
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-kd.stopCh:
				return
			case <-ticker.C:
				engines, err := kd.discoverEngines()
				if err == nil {
					select {
					case engineCh <- engines:
					case <-ctx.Done():
						return
					case <-kd.stopCh:
						return
					}
				}
			}
		}
	}()
	
	return engineCh, nil
}

// Watch watches for engine changes using Kubernetes watch API
func (kd *KubernetesDiscovery) Watch(ctx context.Context, onChange func([]*EngineInfo)) error {
	selector, err := labels.Parse(kd.labelSelector)
	if err != nil {
		return fmt.Errorf("invalid label selector: %w", err)
	}
	
	watcher, err := kd.client.CoreV1().Pods(kd.namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Stop()
	
	// Initial discovery
	engines, err := kd.discoverEngines()
	if err == nil {
		onChange(engines)
	}
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-kd.stopCh:
			return nil
		case event := <-watcher.ResultChan():
			// Pod changed, rediscover
			engines, err := kd.discoverEngines()
			if err == nil {
				onChange(engines)
			}
		}
	}
}

// Close stops discovery
func (kd *KubernetesDiscovery) Close() error {
	close(kd.stopCh)
	return nil
}

// discoverEngines discovers engines from Kubernetes pods
func (kd *KubernetesDiscovery) discoverEngines() ([]*EngineInfo, error) {
	selector, err := labels.Parse(kd.labelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %w", err)
	}
	
	pods, err := kd.client.CoreV1().Pods(kd.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	
	engines := make([]*EngineInfo, 0, len(pods.Items))
	
	for _, pod := range pods.Items {
		// Check if pod is running
		if pod.Status.Phase != "Running" {
			continue
		}
		
		// Extract engine information from pod
		engineID := pod.Name
		if id, ok := pod.Labels["engine-id"]; ok {
			engineID = id
		}
		
		// Get pod IP
		address := pod.Status.PodIP
		if address == "" {
			continue
		}
		
		// Get port from service or pod annotation
		port := 50051 // Default gRPC port
		if portStr, ok := pod.Annotations["workflow-engine/port"]; ok {
			if parsedPort, err := fmt.Sscanf(portStr, "%d", &port); err != nil || parsedPort != 1 {
				port = 50051
			}
		}
		
		// Get capacity from annotation or label
		capacity := 10 // Default
		if capStr, ok := pod.Annotations["workflow-engine/capacity"]; ok {
			if parsedCap, err := fmt.Sscanf(capStr, "%d", &capacity); err != nil || parsedCap != 1 {
				capacity = 10
			}
		}
		
		// Extract metadata
		metadata := make(map[string]string)
		for k, v := range pod.Labels {
			if k != "app" && k != "engine-id" {
				metadata[k] = v
			}
		}
		
		engines = append(engines, &EngineInfo{
			ID:       engineID,
			Address:  address,
			Port:     port,
			Capacity: capacity,
			Metadata: metadata,
			LastSeen: time.Now(),
		})
	}
	
	return engines, nil
}

