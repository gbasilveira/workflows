package orchestrator

import "context"

// WorkflowExecutor defines the interface for executing workflows
type WorkflowExecutor interface {
	ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResponse, error)
}

// Trigger represents an event source that can initiate workflow execution.
type Trigger interface {
	// ID returns the unique identifier for this trigger.
	ID() string
	
	// Start begins listening for trigger events.
	Start(ctx context.Context, executor WorkflowExecutor) error
	
	// Stop stops the trigger from listening for events.
	Stop() error
	
	// IsActive returns whether the trigger is currently active.
	IsActive() bool
	
	// Type returns the type of trigger (e.g., "cron", "http", "webhook").
	Type() string
}

// TriggerEvent represents an event from a trigger that should execute a workflow.
type TriggerEvent struct {
	TriggerID  string
	WorkflowID string
	Inputs     map[string]interface{}
	Metadata   map[string]interface{}
	Context    context.Context
}

// BaseTrigger provides common functionality for all triggers.
type BaseTrigger struct {
	id        string
	active    bool
	triggerType string
}

// NewBaseTrigger creates a new base trigger.
func NewBaseTrigger(id, triggerType string) *BaseTrigger {
	return &BaseTrigger{
		id:          id,
		triggerType: triggerType,
		active:      false,
	}
}

// ID returns the trigger's ID.
func (bt *BaseTrigger) ID() string {
	return bt.id
}

// Type returns the trigger's type.
func (bt *BaseTrigger) Type() string {
	return bt.triggerType
}

// IsActive returns whether the trigger is active.
func (bt *BaseTrigger) IsActive() bool {
	return bt.active
}

// setActive sets the active state.
func (bt *BaseTrigger) setActive(active bool) {
	bt.active = active
}

