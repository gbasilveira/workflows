package orchestrator

import (
	"fmt"
	"sync"
)

// WorkflowVersion represents a versioned workflow
type WorkflowVersion struct {
	WorkflowID string
	Version    string // Semantic version (e.g., "1.2.3")
	Definition *WorkflowDefinition
	CreatedAt  int64
	Dependencies []string // Workflow IDs this depends on
	Dependents  []string  // Workflow IDs that depend on this
	mu         sync.RWMutex
}

// WorkflowDefinition holds the actual workflow structure
type WorkflowDefinition struct {
	WorkflowID string
	Version    string
	Name       string
	Nodes      []NodeDefinition
	Metadata   map[string]interface{}
}

// NodeDefinition represents a node in the workflow
type NodeDefinition struct {
	NodeID      string
	Dependencies []string
	ExecutorType string
	ExecutorCode string
	ExecutorConfig map[string]interface{}
	Metadata    map[string]interface{}
}

// VersionManager manages workflow versions and their dependencies
type VersionManager struct {
	versions   map[string]map[string]*WorkflowVersion // workflowID -> version -> WorkflowVersion
	latest     map[string]*WorkflowVersion             // workflowID -> latest version
	mu         sync.RWMutex
}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	return &VersionManager{
		versions: make(map[string]map[string]*WorkflowVersion),
		latest:   make(map[string]*WorkflowVersion),
	}
}

// RegisterVersion registers a new workflow version
func (vm *VersionManager) RegisterVersion(version *WorkflowVersion) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	if version.WorkflowID == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}
	
	if version.Version == "" {
		return fmt.Errorf("workflow version cannot be empty")
	}
	
	if vm.versions[version.WorkflowID] == nil {
		vm.versions[version.WorkflowID] = make(map[string]*WorkflowVersion)
	}
	
	vm.versions[version.WorkflowID][version.Version] = version
	
	// Update latest if this is newer
	if latest, exists := vm.latest[version.WorkflowID]; !exists || isNewerVersion(version.Version, latest.Version) {
		vm.latest[version.WorkflowID] = version
	}
	
	// Build dependency graph
	vm.updateDependencies(version)
	
	return nil
}

// GetVersion retrieves a specific workflow version
func (vm *VersionManager) GetVersion(workflowID, version string) (*WorkflowVersion, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	if version == "" || version == "latest" {
		if latest, exists := vm.latest[workflowID]; exists {
			return latest, nil
		}
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	versions, exists := vm.versions[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	v, exists := versions[version]
	if !exists {
		return nil, fmt.Errorf("workflow %s version %s not found", workflowID, version)
	}
	
	return v, nil
}

// GetLatestVersion retrieves the latest version of a workflow
func (vm *VersionManager) GetLatestVersion(workflowID string) (*WorkflowVersion, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	latest, exists := vm.latest[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	return latest, nil
}

// CanUpdate checks if a workflow can be safely updated
func (vm *VersionManager) CanUpdate(workflowID, newVersion string) (bool, []string, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	version, exists := vm.versions[workflowID]
	if !exists {
		return true, nil, nil // New workflow, can always add
	}
	
	currentLatest := vm.latest[workflowID]
	if currentLatest == nil {
		return true, nil, nil
	}
	
	// Check if there are dependents
	dependents := currentLatest.Dependents
	if len(dependents) > 0 {
		return false, dependents, fmt.Errorf("workflow %s has %d dependents, cannot update safely", workflowID, len(dependents))
	}
	
	// Check if new version is actually newer
	if !isNewerVersion(newVersion, currentLatest.Version) {
		return false, nil, fmt.Errorf("version %s is not newer than current %s", newVersion, currentLatest.Version)
	}
	
	// Check if this version already exists
	if _, exists := version[newVersion]; exists {
		return false, nil, fmt.Errorf("version %s already exists", newVersion)
	}
	
	return true, nil, nil
}

// GetDependents returns all workflows that depend on this workflow
func (vm *VersionManager) GetDependents(workflowID string) []string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	latest, exists := vm.latest[workflowID]
	if !exists {
		return nil
	}
	
	return latest.Dependents
}

// updateDependencies updates the dependency graph
func (vm *VersionManager) updateDependencies(version *WorkflowVersion) {
	// Find workflows that depend on this one
	version.mu.Lock()
	defer version.mu.Unlock()
	
	version.Dependents = make([]string, 0)
	
	// Check all workflows to find dependents
	for wfID, versions := range vm.versions {
		if wfID == version.WorkflowID {
			continue
		}
		
		for _, v := range versions {
			for _, dep := range v.Dependencies {
				if dep == version.WorkflowID {
					// This workflow depends on the version being registered
					version.Dependents = append(version.Dependents, wfID)
					break
				}
			}
		}
	}
	
	// Update dependents' dependency lists
	for _, wfID := range version.Dependents {
		if versions, exists := vm.versions[wfID]; exists {
			for _, v := range versions {
				// Check if this version's dependencies include our workflow
				for _, dep := range v.Dependencies {
					if dep == version.WorkflowID {
						// Update the dependent's dependency list
						v.mu.Lock()
						// Dependencies are already set, we just track them
						v.mu.Unlock()
					}
				}
			}
		}
	}
}

// isNewerVersion compares two semantic versions (simplified)
func isNewerVersion(v1, v2 string) bool {
	// Simple comparison - in production, use a proper semver library
	// For now, assume lexicographic comparison for simplicity
	return v1 > v2
}

