package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/robfig/cron/v3"
)

// CronTrigger executes workflows on a schedule using cron syntax.
type CronTrigger struct {
	*BaseTrigger
	schedule      string
	workflowID    string
	cron          *cron.Cron
	cronID        cron.EntryID
	mu            sync.Mutex
	inputsBuilder func() map[string]interface{} // Optional function to build inputs dynamically
}

// CronTriggerConfig configures a cron trigger.
type CronTriggerConfig struct {
	ID            string
	Schedule      string                    // Cron expression (e.g., "0 */5 * * * *" for every 5 minutes)
	WorkflowID    string
	InputsBuilder func() map[string]interface{} // Optional: dynamic inputs based on trigger time
}

// NewCronTrigger creates a new cron trigger.
func NewCronTrigger(config CronTriggerConfig) (*CronTrigger, error) {
	// Validate cron expression
	c := cron.New(cron.WithSeconds())
	if _, err := c.AddFunc(config.Schedule, func() {}); err != nil {
		return nil, fmt.Errorf("invalid cron schedule: %w", err)
	}
	c.Stop()
	
	return &CronTrigger{
		BaseTrigger:   NewBaseTrigger(config.ID, "cron"),
		schedule:      config.Schedule,
		workflowID:    config.WorkflowID,
		cron:          cron.New(cron.WithSeconds()),
		inputsBuilder: config.InputsBuilder,
	}, nil
}

// Start begins the cron trigger.
func (ct *CronTrigger) Start(ctx context.Context, executor WorkflowExecutor) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if ct.active {
		return fmt.Errorf("cron trigger %s is already active", ct.id)
	}
	
	// Add cron job
	var err error
	ct.cronID, err = ct.cron.AddFunc(ct.schedule, func() {
		inputs := make(map[string]interface{})
		if ct.inputsBuilder != nil {
			inputs = ct.inputsBuilder()
		}
		
		// Add trigger metadata
		inputs["_trigger_type"] = "cron"
		inputs["_trigger_id"] = ct.id
		inputs["_trigger_time"] = time.Now().Unix()
		
		// Execute workflow
		workflowCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		
		if _, err := executor.ExecuteWorkflow(workflowCtx, ct.workflowID, inputs); err != nil {
			// Log error (could be sent to monitoring system)
			fmt.Printf("Cron trigger %s failed to execute workflow %s: %v\n", ct.id, ct.workflowID, err)
		}
	})
	
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}
	
	ct.cron.Start()
	ct.setActive(true)
	
	// Stop when context is done
	go func() {
		<-ctx.Done()
		ct.Stop()
	}()
	
	return nil
}

// Stop stops the cron trigger.
func (ct *CronTrigger) Stop() error {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if !ct.active {
		return nil
	}
	
	ct.cron.Stop()
	ct.setActive(false)
	
	return nil
}

