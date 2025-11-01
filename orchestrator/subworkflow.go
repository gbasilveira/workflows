package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SubWorkflowExecution tracks a sub-workflow execution
type SubWorkflowExecution struct {
	ExecutionID       string
	SubWorkflowID     string
	SubWorkflowVersion string
	ParentWorkflowID  string
	ParentExecutionID string
	Status            string
	Outputs           map[string]interface{}
	Error             error
	StartTime         time.Time
	EndTime           *time.Time
}

// SubWorkflowCoordinator coordinates sub-workflow executions
type SubWorkflowCoordinator struct {
	activeSubWorkflows map[string]*SubWorkflowExecution // executionID -> execution
	parentToChildren   map[string][]string              // parentExecutionID -> childExecutionIDs
	mu                 sync.RWMutex
	orchestrator       interface {
		ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResponse, error)
	}
}

// NewSubWorkflowCoordinator creates a new sub-workflow coordinator
func NewSubWorkflowCoordinator(orch interface {
	ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResponse, error)
}) *SubWorkflowCoordinator {
	return &SubWorkflowCoordinator{
		activeSubWorkflows: make(map[string]*SubWorkflowExecution),
		parentToChildren:   make(map[string][]string),
		orchestrator:       orch,
	}
}

// ExecuteSubWorkflow executes a sub-workflow through the orchestrator
func (swc *SubWorkflowCoordinator) ExecuteSubWorkflow(ctx context.Context, 
	subWorkflowID, subWorkflowVersion, parentWorkflowID, parentExecutionID string,
	inputs map[string]interface{}) (string, error) {
	
	// Generate execution ID for sub-workflow
	executionID := fmt.Sprintf("%s-sub-%d", parentExecutionID, time.Now().UnixNano())
	
	swc.mu.Lock()
	
	// Create sub-workflow execution record
	exec := &SubWorkflowExecution{
		ExecutionID:       executionID,
		SubWorkflowID:     subWorkflowID,
		SubWorkflowVersion: subWorkflowVersion,
		ParentWorkflowID:  parentWorkflowID,
		ParentExecutionID: parentExecutionID,
		Status:            "PENDING",
		StartTime:         time.Now(),
	}
	
	swc.activeSubWorkflows[executionID] = exec
	
	// Track parent-child relationship
	if swc.parentToChildren[parentExecutionID] == nil {
		swc.parentToChildren[parentExecutionID] = make([]string, 0)
	}
	swc.parentToChildren[parentExecutionID] = append(swc.parentToChildren[parentExecutionID], executionID)
	
	swc.mu.Unlock()
	
	// Add parent context information to inputs
	if inputs == nil {
		inputs = make(map[string]interface{})
	}
	inputs["_parent_workflow_id"] = parentWorkflowID
	inputs["_parent_execution_id"] = parentExecutionID
	
	// Execute sub-workflow through orchestrator
	go func() {
		defer func() {
			swc.mu.Lock()
			now := time.Now()
			exec.EndTime = &now
			swc.mu.Unlock()
		}()
		
		swc.mu.Lock()
		exec.Status = "RUNNING"
		swc.mu.Unlock()
		
		response, err := swc.orchestrator.ExecuteWorkflow(ctx, subWorkflowID, inputs)
		
		swc.mu.Lock()
		if err != nil {
			exec.Status = "FAILED"
			exec.Error = err
		} else {
			exec.Status = "COMPLETED"
			exec.Outputs = response.Outputs
		}
		swc.mu.Unlock()
	}()
	
	return executionID, nil
}

// GetSubWorkflowStatus returns the status of a sub-workflow
func (swc *SubWorkflowCoordinator) GetSubWorkflowStatus(executionID string) (*SubWorkflowExecution, error) {
	swc.mu.RLock()
	defer swc.mu.RUnlock()
	
	exec, exists := swc.activeSubWorkflows[executionID]
	if !exists {
		return nil, fmt.Errorf("sub-workflow %s not found", executionID)
	}
	
	// Return a copy
	return &SubWorkflowExecution{
		ExecutionID:       exec.ExecutionID,
		SubWorkflowID:     exec.SubWorkflowID,
		SubWorkflowVersion: exec.SubWorkflowVersion,
		ParentWorkflowID:  exec.ParentWorkflowID,
		ParentExecutionID: exec.ParentExecutionID,
		Status:            exec.Status,
		Outputs:           exec.Outputs,
		Error:             exec.Error,
		StartTime:         exec.StartTime,
		EndTime:           exec.EndTime,
	}, nil
}

// GetChildrenExecutions returns all child execution IDs for a parent
func (swc *SubWorkflowCoordinator) GetChildrenExecutions(parentExecutionID string) []string {
	swc.mu.RLock()
	defer swc.mu.RUnlock()
	
	children, exists := swc.parentToChildren[parentExecutionID]
	if !exists {
		return nil
	}
	
	result := make([]string, len(children))
	copy(result, children)
	return result
}

// CancelChildren cancels all child workflows for a parent
func (swc *SubWorkflowCoordinator) CancelChildren(parentExecutionID string) error {
	swc.mu.RLock()
	children := swc.parentToChildren[parentExecutionID]
	swc.mu.RUnlock()
	
	for _, childID := range children {
		swc.mu.RLock()
		exec, exists := swc.activeSubWorkflows[childID]
		swc.mu.RUnlock()
		
		if exists && exec.Status == "RUNNING" {
			// TODO: Cancel through orchestrator
			swc.mu.Lock()
			exec.Status = "CANCELLED"
			swc.mu.Unlock()
		}
	}
	
	return nil
}

// HandleWorkflowUpdate handles updates when a parent workflow changes
func (swc *SubWorkflowCoordinator) HandleWorkflowUpdate(workflowID, oldVersion, newVersion string) error {
	swc.mu.RLock()
	defer swc.mu.RUnlock()
	
	// Check if any active sub-workflows depend on this workflow
	for _, exec := range swc.activeSubWorkflows {
		if exec.SubWorkflowID == workflowID && exec.Status == "RUNNING" {
			// Sub-workflow is using old version
			// Policy: cancel or allow to complete?
			// For now, log warning
			// In production, implement proper version migration strategy
		}
	}
	
	return nil
}

