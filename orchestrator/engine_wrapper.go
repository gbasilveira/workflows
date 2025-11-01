package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/gbasilveira/dag-engine/dagengine"
)

// EngineWrapper wraps a DAG engine with communication channels.
type EngineWrapper struct {
	ID            string
	Engine        *dagengine.DAGEngine
	Inbound       chan *EngineMessage
	Outbound      chan *EngineMessage
	Status        EngineStatus
	mu            sync.RWMutex
	currentCtx    context.Context
	currentCancel context.CancelFunc
	currentWorkflow string
	wg            sync.WaitGroup
}

// NewEngineWrapper creates a new engine wrapper with communication channels.
func NewEngineWrapper(id string, engine *dagengine.DAGEngine) *EngineWrapper {
	return &EngineWrapper{
		ID:       id,
		Engine:   engine,
		Inbound:  make(chan *EngineMessage, 100),
		Outbound: make(chan *EngineMessage, 100),
		Status:   StatusIdle,
	}
}

// Start begins the engine's message processing loop.
func (ew *EngineWrapper) Start(ctx context.Context) {
	ew.wg.Add(1)
	go ew.processMessages(ctx)
}

// Stop gracefully stops the engine wrapper.
func (ew *EngineWrapper) Stop() {
	ew.mu.Lock()
	if ew.currentCancel != nil {
		ew.currentCancel()
	}
	ew.mu.Unlock()
	
	// Send stop message to itself
	select {
	case ew.Inbound <- &EngineMessage{
		Type:      MsgTypeStop,
		EngineID:  ew.ID,
		Timestamp: time.Now(),
	}:
	default:
	}
	
	ew.wg.Wait()
	close(ew.Inbound)
	close(ew.Outbound)
}

// SendMessage sends a message to the engine.
func (ew *EngineWrapper) SendMessage(msg *EngineMessage) error {
	select {
	case ew.Inbound <- msg:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message to engine %s", ew.ID)
	}
}

// GetState returns the current state of the engine.
func (ew *EngineWrapper) GetState() *EngineState {
	ew.mu.RLock()
	defer ew.mu.RUnlock()
	
	nodeCount := 0
	if ew.Engine != nil {
		// Note: Direct access for monitoring - acceptable race condition
		// For production, consider adding a NodeCount() method to the engine
		nodeCount = len(ew.Engine.Nodes)
	}
	
	var startTime *time.Time
	if ew.Status == StatusRunning && ew.currentCtx != nil {
		now := time.Now()
		startTime = &now
	}
	
	return &EngineState{
		EngineID:       ew.ID,
		Status:         ew.Status,
		CurrentWorkflow: ew.currentWorkflow,
		TotalNodes:     nodeCount,
		LastUpdate:     time.Now(),
		StartTime:      startTime,
		Metadata:       make(map[string]interface{}),
	}
}

// processMessages handles incoming messages for the engine.
func (ew *EngineWrapper) processMessages(ctx context.Context) {
	defer ew.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			ew.mu.Lock()
			ew.Status = StatusStopped
			ew.mu.Unlock()
			return
			
		case msg, ok := <-ew.Inbound:
			if !ok {
				return
			}
			
			ew.handleMessage(ctx, msg)
		}
	}
}

// handleMessage processes a single message.
func (ew *EngineWrapper) handleMessage(ctx context.Context, msg *EngineMessage) {
	switch msg.Type {
	case MsgTypeWorkflowRequest:
		ew.handleWorkflowRequest(ctx, msg)
	case MsgTypeStop:
		ew.mu.Lock()
		ew.Status = StatusStopped
		if ew.currentCancel != nil {
			ew.currentCancel()
		}
		ew.mu.Unlock()
	case MsgTypePause:
		ew.mu.Lock()
		if ew.Status == StatusRunning {
			ew.Status = StatusPaused
			if ew.currentCancel != nil {
				ew.currentCancel()
			}
		}
		ew.mu.Unlock()
	case MsgTypeResume:
		ew.mu.Lock()
		if ew.Status == StatusPaused {
			ew.Status = StatusIdle
		}
		ew.mu.Unlock()
	case MsgTypeHealthCheck:
		ew.handleHealthCheck(msg)
	}
}

// handleWorkflowRequest executes a workflow in the engine.
func (ew *EngineWrapper) handleWorkflowRequest(ctx context.Context, msg *EngineMessage) {
	// Extract workflow ID from message payload
	workflowID := ""
	if payload, ok := msg.Payload.(map[string]interface{}); ok {
		if id, exists := payload["workflow_id"].(string); exists {
			workflowID = id
		}
	}
	
	ew.mu.Lock()
	ew.Status = StatusRunning
	ew.currentWorkflow = workflowID
	workflowCtx, cancel := context.WithCancel(ctx)
	ew.currentCtx = workflowCtx
	ew.currentCancel = cancel
	startTime := time.Now()
	ew.mu.Unlock()
	
	go func() {
		defer func() {
			ew.mu.Lock()
			ew.Status = StatusIdle
			ew.currentWorkflow = ""
			ew.currentCtx = nil
			ew.currentCancel = nil
			ew.mu.Unlock()
		}()
		
		// Execute the workflow
		err := ew.Engine.Run(workflowCtx)
		duration := time.Since(startTime)
		
		// Send response
		response := &EngineMessage{
			Type:      MsgTypeWorkflowResponse,
			EngineID:  ew.ID,
			Timestamp: time.Now(),
			RequestID: msg.RequestID,
			Payload: map[string]interface{}{
				"success":  err == nil,
				"error":    err,
				"duration": duration.Nanoseconds(),
			},
		}
		
		select {
		case ew.Outbound <- response:
		case <-workflowCtx.Done():
		}
		
		// Send status update
		statusMsg := &EngineMessage{
			Type:      MsgTypeStatusUpdate,
			EngineID:  ew.ID,
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"status":   "idle",
				"duration": duration.Nanoseconds(),
			},
		}
		
		select {
		case ew.Outbound <- statusMsg:
		default:
		}
	}()
}

// handleHealthCheck responds to health check requests.
func (ew *EngineWrapper) handleHealthCheck(msg *EngineMessage) {
	state := ew.GetState()
	response := &EngineMessage{
		Type:      MsgTypeHealthResponse,
		EngineID:  ew.ID,
		Timestamp: time.Now(),
		RequestID: msg.RequestID,
		Payload:   state,
	}
	
	select {
	case ew.Outbound <- response:
	default:
	}
}

