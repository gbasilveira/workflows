package orchestrator

import (
	"context"
	"github.com/gbasilveira/dag-engine/dagengine"
)

// Workflow represents a workflow definition that can be executed by an engine.
type Workflow struct {
	ID          string
	Name        string
	Description string
	// Builder function that creates a configured DAG engine for this workflow
	Builder func() (*dagengine.DAGEngine, error)
	// Metadata for workflows
	Metadata map[string]interface{}
}

// WorkflowRequest represents a request to execute a workflow.
type WorkflowRequest struct {
	WorkflowID string
	Inputs     map[string]interface{}
	Context    context.Context
	// Response channel for the result
	Response chan *WorkflowResponse
	// Error channel for errors
	Error chan error
}

// WorkflowResponse represents the result of a workflow execution.
type WorkflowResponse struct {
	WorkflowID string
	Success    bool
	Outputs    map[string]interface{}
	Duration   int64 // nanoseconds
	Metadata   map[string]interface{}
}

