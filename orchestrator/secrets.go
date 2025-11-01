package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// SecretsManager manages Kubernetes secrets for workflows
type SecretsManager struct {
	client    kubernetes.Interface
	namespace string
	cache     map[string]*secretCacheEntry
	mu        sync.RWMutex
	ttl       time.Duration
}

type secretCacheEntry struct {
	secret   *corev1.Secret
	expires  time.Time
}

// NewSecretsManager creates a new secrets manager
func NewSecretsManager(client kubernetes.Interface, namespace string) *SecretsManager {
	return &SecretsManager{
		client:    client,
		namespace: namespace,
		cache:     make(map[string]*secretCacheEntry),
		ttl:       5 * time.Minute, // Cache TTL
	}
}

// NewSecretsManagerFromConfig creates a secrets manager from K8s config
func NewSecretsManagerFromConfig(config *rest.Config, namespace string) (*SecretsManager, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return NewSecretsManager(client, namespace), nil
}

// GetSecret retrieves a secret from Kubernetes
func (sm *SecretsManager) GetSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	// Use default namespace if not specified
	if namespace == "" {
		namespace = sm.namespace
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s/%s", namespace, name)
	sm.mu.RLock()
	if entry, exists := sm.cache[cacheKey]; exists {
		if time.Now().Before(entry.expires) {
			secret := entry.secret
			sm.mu.RUnlock()
			return secret, nil
		}
	}
	sm.mu.RUnlock()

	// Fetch from Kubernetes
	secret, err := sm.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, name, err)
	}

	// Cache the secret
	sm.mu.Lock()
	sm.cache[cacheKey] = &secretCacheEntry{
		secret:  secret,
		expires: time.Now().Add(sm.ttl),
	}
	sm.mu.Unlock()

	return secret, nil
}

// GetSecretValue retrieves a specific key value from a secret
func (sm *SecretsManager) GetSecretValue(ctx context.Context, name, namespace, key string) ([]byte, error) {
	secret, err := sm.GetSecret(ctx, name, namespace)
	if err != nil {
		return nil, err
	}

	value, exists := secret.Data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found in secret %s/%s", key, namespace, name)
	}

	return value, nil
}

// ClearCache clears the secret cache
func (sm *SecretsManager) ClearCache() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.cache = make(map[string]*secretCacheEntry)
}

// ExtractSecretsFromConfig extracts secret references from workflow configuration
func ExtractSecretsFromConfig(config map[string]interface{}) []SecretRef {
	if config == nil {
		return nil
	}

	configMeta, ok := config["configuration"].(map[string]interface{})
	if !ok {
		return nil
	}

	secretsMeta, ok := configMeta["secrets"].([]interface{})
	if !ok {
		return nil
	}

	secrets := make([]SecretRef, 0, len(secretsMeta))
	for _, sec := range secretsMeta {
		secMap, ok := sec.(map[string]interface{})
		if !ok {
			continue
		}

		ref := SecretRef{}
		if name, ok := secMap["name"].(string); ok {
			ref.Name = name
		}
		if namespace, ok := secMap["namespace"].(string); ok {
			ref.Namespace = namespace
		}
		if mountPath, ok := secMap["mount_path"].(string); ok {
			ref.MountPath = mountPath
		}
		if keys, ok := secMap["keys"].(map[string]interface{}); ok {
			ref.Keys = make(map[string]string)
			for k, v := range keys {
				if vStr, ok := v.(string); ok {
					ref.Keys[k] = vStr
				}
			}
		}

		secrets = append(secrets, ref)
	}

	return secrets
}

// SecretRef represents a reference to a Kubernetes secret
type SecretRef struct {
	Name      string
	Namespace string
	MountPath string
	Keys      map[string]string // Map of secret keys to env var names
}

// InjectSecretsIntoEnv injects secret values into environment variables
func (sm *SecretsManager) InjectSecretsIntoEnv(ctx context.Context, secrets []SecretRef) (map[string]string, error) {
	env := make(map[string]string)

	for _, ref := range secrets {
		if len(ref.Keys) > 0 {
			// Map specific keys to env vars
			for secretKey, envVar := range ref.Keys {
				value, err := sm.GetSecretValue(ctx, ref.Name, ref.Namespace, secretKey)
				if err != nil {
					return nil, fmt.Errorf("failed to get secret key %s from %s/%s: %w", 
						secretKey, ref.Namespace, ref.Name, err)
				}
				env[envVar] = string(value)
			}
		} else {
			// Inject all keys with prefix
			secret, err := sm.GetSecret(ctx, ref.Name, ref.Namespace)
			if err != nil {
				return nil, err
			}

			for key, value := range secret.Data {
				envVar := fmt.Sprintf("SECRET_%s_%s", ref.Name, key)
				env[envVar] = string(value)
			}
		}
	}

	return env, nil
}

