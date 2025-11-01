package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/gbasilveira/dag-engine/dagengine"
)

// EngineService represents an engine that can execute workflows
type EngineService struct {
	ID            string
	capacity      int
	activeWorkflows map[string]*WorkflowExecution
	mu            sync.RWMutex
}

// WorkflowExecution tracks a running workflow
type WorkflowExecution struct {
	ExecutionID   string
	WorkflowID    string
	Version       string
	Engine        *dagengine.DAGEngine
	StartTime     time.Time
	Status        string
	Context       context.Context
	Cancel        context.CancelFunc
}

// NewEngineService creates a new engine service
func NewEngineService(id string, capacity int) *EngineService {
	return &EngineService{
		ID:             id,
		capacity:       capacity,
		activeWorkflows: make(map[string]*WorkflowExecution),
	}
}

// ExecuteWorkflow executes a workflow on this engine
func (es *EngineService) ExecuteWorkflow(ctx context.Context, workflowID, version, executionID string, engine *dagengine.DAGEngine) error {
	es.mu.Lock()
	
	// Check capacity
	if len(es.activeWorkflows) >= es.capacity {
		es.mu.Unlock()
		return fmt.Errorf("engine at capacity (%d)", es.capacity)
	}
	
	// Create execution context
	execCtx, cancel := context.WithCancel(ctx)
	
	exec := &WorkflowExecution{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Version:     version,
		Engine:      engine,
		StartTime:   time.Now(),
		Status:      "RUNNING",
		Context:     execCtx,
		Cancel:      cancel,
	}
	
	es.activeWorkflows[executionID] = exec
	es.mu.Unlock()
	
	// Execute in goroutine
	go func() {
		defer func() {
			es.mu.Lock()
			delete(es.activeWorkflows, executionID)
			es.mu.Unlock()
		}()
		
		err := engine.Run(execCtx)
		
		es.mu.Lock()
		if err != nil {
			exec.Status = "FAILED"
		} else {
			exec.Status = "COMPLETED"
		}
		es.mu.Unlock()
	}()
	
	return nil
}

// StopWorkflow stops a running workflow
func (es *EngineService) StopWorkflow(executionID string) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	
	exec, exists := es.activeWorkflows[executionID]
	if !exists {
		return fmt.Errorf("workflow %s not found", executionID)
	}
	
	exec.Status = "CANCELLED"
	exec.Cancel()
	
	return nil
}

// GetActiveWorkflows returns the number of active workflows
func (es *EngineService) GetActiveWorkflows() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	return len(es.activeWorkflows)
}

// GetCapacity returns the engine capacity
func (es *EngineService) GetCapacity() int {
	return es.capacity
}

// GetWorkflowStatus returns the status of a workflow
func (es *EngineService) GetWorkflowStatus(executionID string) (string, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	exec, exists := es.activeWorkflows[executionID]
	if !exists {
		return "", fmt.Errorf("workflow %s not found", executionID)
	}
	
	return exec.Status, nil
}

// ListActiveWorkflows returns all active workflow IDs
func (es *EngineService) ListActiveWorkflows() []string {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	workflows := make([]string, 0, len(es.activeWorkflows))
	for id := range es.activeWorkflows {
		workflows = append(workflows, id)
	}
	
	return workflows
}

