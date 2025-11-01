package main

import (
	"fmt"

	"github.com/gbasilveira/dag-engine/orchestrator"
	"github.com/gbasilveira/dag-engine/spec"
)

// ConvertYAMLToWorkflowDefinition converts a YAML WorkflowSpec to an internal WorkflowDefinition
func ConvertYAMLToWorkflowDefinition(yamlSpec *spec.WorkflowSpec) (*orchestrator.WorkflowDefinition, error) {
	if yamlSpec == nil {
		return nil, fmt.Errorf("yaml spec is nil")
	}

	nodes := make([]orchestrator.NodeDefinition, 0, len(yamlSpec.Spec.Nodes))

	for _, yamlNode := range yamlSpec.Spec.Nodes {
		node := orchestrator.NodeDefinition{
			NodeID:       yamlNode.ID,
			Dependencies: yamlNode.Dependencies,
			ExecutorType: yamlNode.Executor.Type,
			ExecutorCode: yamlNode.Executor.Code,
			ExecutorConfig: yamlNode.Executor.Config,
			Metadata:     yamlNode.Metadata,
		}

		nodes = append(nodes, node)
	}

	// Build metadata from YAML spec
	metadata := make(map[string]interface{})
	if yamlSpec.Metadata.Labels != nil {
		metadata["labels"] = yamlSpec.Metadata.Labels
	}
	if yamlSpec.Metadata.Annotations != nil {
		metadata["annotations"] = yamlSpec.Metadata.Annotations
	}
	if yamlSpec.Metadata.Description != "" {
		metadata["description"] = yamlSpec.Metadata.Description
	}

	// Add triggers to metadata (descriptive only)
	if yamlSpec.Spec.Triggers != nil {
		triggersMeta := make(map[string]interface{})
		if yamlSpec.Spec.Triggers.HTTP != nil {
			triggersMeta["http"] = map[string]interface{}{
				"port":   yamlSpec.Spec.Triggers.HTTP.Port,
				"path":   yamlSpec.Spec.Triggers.HTTP.Path,
				"method": yamlSpec.Spec.Triggers.HTTP.Method,
			}
		}
		if yamlSpec.Spec.Triggers.Cron != nil {
			triggersMeta["cron"] = map[string]interface{}{
				"schedule": yamlSpec.Spec.Triggers.Cron.Schedule,
				"timezone": yamlSpec.Spec.Triggers.Cron.Timezone,
			}
		}
		metadata["triggers"] = triggersMeta
	}

	// Add configuration to metadata
	if yamlSpec.Spec.Configuration != nil {
		configMeta := make(map[string]interface{})
		
		if yamlSpec.Spec.Configuration.Secrets != nil {
			secretsMeta := make([]map[string]interface{}, 0, len(yamlSpec.Spec.Configuration.Secrets))
			for _, secret := range yamlSpec.Spec.Configuration.Secrets {
				secretMeta := map[string]interface{}{
					"name": secret.Name,
				}
				if secret.Namespace != "" {
					secretMeta["namespace"] = secret.Namespace
				}
				if secret.MountPath != "" {
					secretMeta["mount_path"] = secret.MountPath
				}
				if secret.Keys != nil {
					secretMeta["keys"] = secret.Keys
				}
				secretsMeta = append(secretsMeta, secretMeta)
			}
			configMeta["secrets"] = secretsMeta
		}

		if yamlSpec.Spec.Configuration.Kubernetes != nil {
			k8sMeta := make(map[string]interface{})
			if yamlSpec.Spec.Configuration.Kubernetes.ResourceLimits != nil {
				k8sMeta["resource_limits"] = map[string]interface{}{
					"cpu":    yamlSpec.Spec.Configuration.Kubernetes.ResourceLimits.CPU,
					"memory": yamlSpec.Spec.Configuration.Kubernetes.ResourceLimits.Memory,
				}
			}
			if yamlSpec.Spec.Configuration.Kubernetes.Annotations != nil {
				k8sMeta["annotations"] = yamlSpec.Spec.Configuration.Kubernetes.Annotations
			}
			if yamlSpec.Spec.Configuration.Kubernetes.Labels != nil {
				k8sMeta["labels"] = yamlSpec.Spec.Configuration.Kubernetes.Labels
			}
			if yamlSpec.Spec.Configuration.Kubernetes.ServiceAccount != "" {
				k8sMeta["service_account"] = yamlSpec.Spec.Configuration.Kubernetes.ServiceAccount
			}
			configMeta["kubernetes"] = k8sMeta
		}

		if yamlSpec.Spec.Configuration.Env != nil {
			configMeta["env"] = yamlSpec.Spec.Configuration.Env
		}

		metadata["configuration"] = configMeta
	}

	def := &orchestrator.WorkflowDefinition{
		WorkflowID: yamlSpec.Metadata.ID,
		Version:    yamlSpec.Metadata.Version,
		Name:      yamlSpec.Metadata.Name,
		Nodes:     nodes,
		Metadata:  metadata,
	}

	return def, nil
}

