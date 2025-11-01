package orchestrator

import (
	"fmt"
	"sync"
	"time"
)

// WorkflowManager manages workflow definitions, versions, and metadata
type WorkflowManager struct {
	versions   *VersionManager
	builders   map[string]WorkflowBuilder // workflowID -> builder function
	metadata   map[string]map[string]interface{} // workflowID -> metadata
	mu         sync.RWMutex
}

// WorkflowBuilder is a function that builds a workflow definition
type WorkflowBuilder func() (*WorkflowDefinition, error)

// NewWorkflowManager creates a new workflow manager
func NewWorkflowManager() *WorkflowManager {
	return &WorkflowManager{
		versions: NewVersionManager(),
		builders: make(map[string]WorkflowBuilder),
		metadata: make(map[string]map[string]interface{}),
	}
}

// RegisterWorkflow registers a workflow with a builder function
func (wm *WorkflowManager) RegisterWorkflow(workflowID, version string, builder WorkflowBuilder, metadata map[string]interface{}) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	// Check if we can update
	if canUpdate, dependents, err := wm.versions.CanUpdate(workflowID, version); !canUpdate {
		if len(dependents) > 0 {
			return fmt.Errorf("cannot update workflow %s: has dependents %v", workflowID, dependents)
		}
		return err
	}
	
	// Build the definition to validate
	def, err := builder()
	if err != nil {
		return fmt.Errorf("failed to build workflow definition: %w", err)
	}
	
	// Register version
	wfVersion := &WorkflowVersion{
		WorkflowID:   workflowID,
		Version:      version,
		Definition:  def,
		CreatedAt:   time.Now().Unix(),
		Dependencies: extractDependencies(def),
	}
	
	if err := wm.versions.RegisterVersion(wfVersion); err != nil {
		return fmt.Errorf("failed to register version: %w", err)
	}
	
	// Store builder and metadata
	wm.builders[workflowID] = builder
	if metadata != nil {
		wm.metadata[workflowID] = metadata
	} else {
		wm.metadata[workflowID] = make(map[string]interface{})
	}
	
	return nil
}

// GetWorkflowDefinition retrieves a workflow definition
func (wm *WorkflowManager) GetWorkflowDefinition(workflowID, version string) (*WorkflowDefinition, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	wfVersion, err := wm.versions.GetVersion(workflowID, version)
	if err != nil {
		return nil, err
	}
	
	return wfVersion.Definition, nil
}

// BuildWorkflow builds a workflow using its registered builder
func (wm *WorkflowManager) BuildWorkflow(workflowID string) (*WorkflowDefinition, error) {
	wm.mu.RLock()
	builder, exists := wm.builders[workflowID]
	wm.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	return builder()
}

// GetLatestVersion returns the latest version of a workflow
func (wm *WorkflowManager) GetLatestVersion(workflowID string) (string, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	version, err := wm.versions.GetLatestVersion(workflowID)
	if err != nil {
		return "", err
	}
	
	return version.Version, nil
}

// HasWorkflow checks if a workflow exists
func (wm *WorkflowManager) HasWorkflow(workflowID string) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	_, exists := wm.builders[workflowID]
	return exists
}

// GetMetadata retrieves workflow metadata
func (wm *WorkflowManager) GetMetadata(workflowID string) map[string]interface{} {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	meta, exists := wm.metadata[workflowID]
	if !exists {
		return make(map[string]interface{})
	}
	
	// Return a copy
	result := make(map[string]interface{})
	for k, v := range meta {
		result[k] = v
	}
	
	return result
}

// extractDependencies extracts workflow dependencies from definition
func extractDependencies(def *WorkflowDefinition) []string {
	depMap := make(map[string]bool)
	
	// Check for sub-workflow references in metadata or node configs
	for _, node := range def.Nodes {
		if nodeType, ok := node.Metadata["workflow_type"].(string); ok && nodeType == "sub-workflow" {
			if wfID, ok := node.Metadata["workflow_id"].(string); ok {
				depMap[wfID] = true
			}
		}
	}
	
	deps := make([]string, 0, len(depMap))
	for dep := range depMap {
		deps = append(deps, dep)
	}
	
	return deps
}

