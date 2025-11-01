package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/gbasilveira/dag-engine/dagengine"
)

// Orchestrator manages multiple DAG engines with bi-directional communication.
type Orchestrator struct {
	engines     map[string]*EngineWrapper
	workflows   map[string]*Workflow
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	requestID   int64
	requestIDMu sync.Mutex
}

// NewOrchestrator creates a new orchestrator instance.
func NewOrchestrator(ctx context.Context) *Orchestrator {
	orchestratorCtx, cancel := context.WithCancel(ctx)
	return &Orchestrator{
		engines:   make(map[string]*EngineWrapper),
		workflows: make(map[string]*Workflow),
		ctx:       orchestratorCtx,
		cancel:    cancel,
	}
}

// RegisterEngine registers a new engine with the orchestrator.
func (o *Orchestrator) RegisterEngine(engineID string, engine *dagengine.DAGEngine) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if _, exists := o.engines[engineID]; exists {
		return fmt.Errorf("engine %s already registered", engineID)
	}
	
	wrapper := NewEngineWrapper(engineID, engine)
	o.engines[engineID] = wrapper
	wrapper.Start(o.ctx)
	
	// Start monitoring outbound messages from this engine
	o.wg.Add(1)
	go o.monitorEngineMessages(wrapper)
	
	return nil
}

// UnregisterEngine removes an engine from the orchestrator.
func (o *Orchestrator) UnregisterEngine(engineID string) error {
	o.mu.Lock()
	wrapper, exists := o.engines[engineID]
	if exists {
		delete(o.engines, engineID)
	}
	o.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("engine %s not found", engineID)
	}
	
	wrapper.Stop()
	return nil
}

// RegisterWorkflow registers a workflow that can be executed.
func (o *Orchestrator) RegisterWorkflow(workflow *Workflow) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if _, exists := o.workflows[workflow.ID]; exists {
		return fmt.Errorf("workflow %s already registered", workflow.ID)
	}
	
	o.workflows[workflow.ID] = workflow
	return nil
}

// ExecuteWorkflow executes a workflow on an available engine.
func (o *Orchestrator) ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResponse, error) {
	o.mu.RLock()
	workflow, exists := o.workflows[workflowID]
	o.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	// Find an available engine
	var selectedEngine *EngineWrapper
	o.mu.RLock()
	for _, wrapper := range o.engines {
		wrapper.mu.RLock()
		if wrapper.Status == StatusIdle {
			selectedEngine = wrapper
			wrapper.mu.RUnlock()
			break
		}
		wrapper.mu.RUnlock()
	}
	o.mu.RUnlock()
	
	if selectedEngine == nil {
		return nil, fmt.Errorf("no available engine for workflow %s", workflowID)
	}
	
	// Build the workflow engine
	engine, err := workflow.Builder()
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow engine: %w", err)
	}
	
	// Replace the engine in the wrapper
	selectedEngine.mu.Lock()
	selectedEngine.Engine = engine
	selectedEngine.mu.Unlock()
	
	// Preprocess the DAG
	if err := engine.PreprocessDAG(); err != nil {
		return nil, fmt.Errorf("failed to preprocess DAG: %w", err)
	}
	
	// Generate request ID
	requestID := o.nextRequestID()
	
	// Create request message
	msg := &EngineMessage{
		Type:      MsgTypeWorkflowRequest,
		EngineID:  selectedEngine.ID,
		Timestamp: time.Now(),
		RequestID: requestID,
		Payload: map[string]interface{}{
			"workflow_id": workflowID,
			"inputs":      inputs,
		},
	}
	
	// Send message and wait for response
	if err := selectedEngine.SendMessage(msg); err != nil {
		return nil, err
	}
	
	// Wait for response (with timeout)
	timeout := 5 * time.Minute // Default timeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	
	select {
	case responseMsg := <-selectedEngine.Outbound:
		if responseMsg.Type == MsgTypeWorkflowResponse && responseMsg.RequestID == requestID {
			payload, ok := responseMsg.Payload.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid response payload")
			}
			
			success, _ := payload["success"].(bool)
			errVal, _ := payload["error"]
			duration, _ := payload["duration"].(int64)
			
			var err error
			if errVal != nil {
				if errStr, ok := errVal.(string); ok {
					err = fmt.Errorf(errStr)
				} else if errInterface, ok := errVal.(error); ok {
					err = errInterface
				}
			}
			
			if !success && err == nil {
				err = fmt.Errorf("workflow execution failed")
			}
			
			return &WorkflowResponse{
				WorkflowID: workflowID,
				Success:    success && err == nil,
				Duration:   duration,
				Metadata:   payload,
			}, err
		}
	case <-time.After(timeout):
		return nil, fmt.Errorf("workflow execution timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	return nil, fmt.Errorf("unexpected response")
}

// GetEngineState returns the state of a specific engine.
func (o *Orchestrator) GetEngineState(engineID string) (*EngineState, error) {
	o.mu.RLock()
	wrapper, exists := o.engines[engineID]
	o.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("engine %s not found", engineID)
	}
	
	return wrapper.GetState(), nil
}

// GetAllEngineStates returns the state of all engines.
func (o *Orchestrator) GetAllEngineStates() map[string]*EngineState {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	states := make(map[string]*EngineState)
	for id, wrapper := range o.engines {
		states[id] = wrapper.GetState()
	}
	
	return states
}

// GetEngineOutboundChannel returns the outbound channel for an engine (for monitoring).
func (o *Orchestrator) GetEngineOutboundChannel(engineID string) (chan *EngineMessage, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	wrapper, exists := o.engines[engineID]
	if !exists {
		return nil, fmt.Errorf("engine %s not found", engineID)
	}
	
	return wrapper.Outbound, nil
}

// Stop gracefully stops the orchestrator and all engines.
func (o *Orchestrator) Stop() {
	o.mu.Lock()
	engines := make([]*EngineWrapper, 0, len(o.engines))
	for _, wrapper := range o.engines {
		engines = append(engines, wrapper)
	}
	o.mu.Unlock()
	
	for _, wrapper := range engines {
		wrapper.Stop()
	}
	
	o.cancel()
	o.wg.Wait()
}

// monitorEngineMessages monitors outbound messages from an engine.
func (o *Orchestrator) monitorEngineMessages(wrapper *EngineWrapper) {
	defer o.wg.Done()
	
	for {
		select {
		case <-o.ctx.Done():
			return
		case msg, ok := <-wrapper.Outbound:
			if !ok {
				return
			}
			// Messages are consumed here but can be forwarded to monitoring system
			// The monitoring system will attach its own listener
			_ = msg
		}
	}
}

// nextRequestID generates a unique request ID.
func (o *Orchestrator) nextRequestID() string {
	o.requestIDMu.Lock()
	defer o.requestIDMu.Unlock()
	o.requestID++
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), o.requestID)
}

