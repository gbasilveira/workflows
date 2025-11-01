package spec

import (
	"fmt"
	"regexp"
	"strings"
)

// Validate validates a WorkflowSpec and returns any errors
func (ws *WorkflowSpec) Validate() error {
	var errors []string

	// Validate API version
	if ws.APIVersion == "" {
		errors = append(errors, "apiVersion is required")
	} else if ws.APIVersion != "workflows/v1" {
		errors = append(errors, fmt.Sprintf("unsupported apiVersion: %s (expected: workflows/v1)", ws.APIVersion))
	}

	// Validate kind
	if ws.Kind == "" {
		errors = append(errors, "kind is required")
	} else if ws.Kind != "Workflow" {
		errors = append(errors, fmt.Sprintf("unsupported kind: %s (expected: Workflow)", ws.Kind))
	}

	// Validate metadata
	if err := ws.Metadata.Validate(); err != nil {
		errors = append(errors, fmt.Sprintf("metadata: %v", err))
	}

	// Validate spec
	if err := ws.Spec.Validate(); err != nil {
		errors = append(errors, fmt.Sprintf("spec: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate validates WorkflowMetadata
func (wm *WorkflowMetadata) Validate() error {
	if wm.ID == "" {
		return fmt.Errorf("id is required")
	}

	if !isValidID(wm.ID) {
		return fmt.Errorf("id must be alphanumeric with hyphens and underscores only: %s", wm.ID)
	}

	if wm.Name == "" {
		return fmt.Errorf("name is required")
	}

	if wm.Version == "" {
		return fmt.Errorf("version is required")
	}

	if !isValidVersion(wm.Version) {
		return fmt.Errorf("version must be a valid semantic version: %s", wm.Version)
	}

	return nil
}

// Validate validates WorkflowSpecDef
func (wsd *WorkflowSpecDef) Validate() error {
	if len(wsd.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}

	// Validate all nodes
	nodeIDs := make(map[string]bool)
	for i, node := range wsd.Nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("node[%d]: %v", i, err)
		}

		// Check for duplicate node IDs
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// Validate dependencies reference existing nodes
	for i, node := range wsd.Nodes {
		for _, dep := range node.Dependencies {
			if !nodeIDs[dep] {
				return fmt.Errorf("node[%d] depends on non-existent node: %s", i, dep)
			}
		}
	}

	// Validate triggers if present
	if wsd.Triggers != nil {
		if err := wsd.Triggers.Validate(); err != nil {
			return fmt.Errorf("triggers: %v", err)
		}
	}

	// Validate configuration if present
	if wsd.Configuration != nil {
		if err := wsd.Configuration.Validate(); err != nil {
			return fmt.Errorf("configuration: %v", err)
		}
	}

	return nil
}

// Validate validates NodeSpec
func (ns *NodeSpec) Validate() error {
	if ns.ID == "" {
		return fmt.Errorf("id is required")
	}

	if !isValidID(ns.ID) {
		return fmt.Errorf("id must be alphanumeric with hyphens and underscores only: %s", ns.ID)
	}

	if err := ns.Executor.Validate(); err != nil {
		return fmt.Errorf("executor: %v", err)
	}

	return nil
}

// Validate validates ExecutorSpec
func (es *ExecutorSpec) Validate() error {
	if es.Type == "" {
		return fmt.Errorf("type is required")
	}

	supportedTypes := map[string]bool{
		"lua":   true,
		"shell": true,
	}

	if !supportedTypes[es.Type] {
		return fmt.Errorf("unsupported executor type: %s (supported: lua, shell)", es.Type)
	}

	if es.Type == "lua" && es.Code == "" {
		return fmt.Errorf("code is required for lua executor")
	}

	return nil
}

// Validate validates TriggersSpec
func (ts *TriggersSpec) Validate() error {
	if ts.HTTP != nil {
		if err := ts.HTTP.Validate(); err != nil {
			return fmt.Errorf("http: %v", err)
		}
	}

	if ts.Cron != nil {
		if err := ts.Cron.Validate(); err != nil {
			return fmt.Errorf("cron: %v", err)
		}
	}

	return nil
}

// Validate validates HTTPTriggerSpec
func (hts *HTTPTriggerSpec) Validate() error {
	if hts.Port <= 0 || hts.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535: %d", hts.Port)
	}

	if hts.Path == "" {
		return fmt.Errorf("path is required")
	}

	if !strings.HasPrefix(hts.Path, "/") {
		return fmt.Errorf("path must start with '/': %s", hts.Path)
	}

	if hts.Method != "" {
		validMethods := map[string]bool{
			"GET":    true,
			"POST":   true,
			"PUT":    true,
			"DELETE": true,
			"PATCH":  true,
		}
		if !validMethods[strings.ToUpper(hts.Method)] {
			return fmt.Errorf("invalid HTTP method: %s", hts.Method)
		}
	}

	return nil
}

// Validate validates CronTriggerSpec
func (cts *CronTriggerSpec) Validate() error {
	if cts.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}

	// Basic cron validation (6 fields with seconds)
	parts := strings.Fields(cts.Schedule)
	if len(parts) != 6 {
		return fmt.Errorf("cron schedule must have 6 fields (seconds minutes hours day month weekday): %s", cts.Schedule)
	}

	return nil
}

// Validate validates ConfigSpec
func (cs *ConfigSpec) Validate() error {
	for i, secret := range cs.Secrets {
		if err := secret.Validate(); err != nil {
			return fmt.Errorf("secrets[%d]: %v", i, err)
		}
	}

	return nil
}

// Validate validates SecretRef
func (sr *SecretRef) Validate() error {
	if sr.Name == "" {
		return fmt.Errorf("name is required")
	}

	if !isValidK8sName(sr.Name) {
		return fmt.Errorf("secret name must be a valid Kubernetes name: %s", sr.Name)
	}

	return nil
}

// Helper functions

// isValidID checks if an ID is valid (alphanumeric, hyphens, underscores)
func isValidID(id string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, id)
	return matched
}

// isValidVersion checks if a version string is a valid semantic version
func isValidVersion(version string) bool {
	matched, _ := regexp.MatchString(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?(\+[a-zA-Z0-9]+)?$`, version)
	return matched
}

// isValidK8sName checks if a string is a valid Kubernetes name
// K8s names: lowercase alphanumeric, hyphens, must start and end with alphanumeric, max 253 chars
func isValidK8sName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, name)
	return matched
}
