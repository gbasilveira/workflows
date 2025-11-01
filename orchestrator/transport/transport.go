package transport

import (
	"context"
	"time"
)

// EngineInfo contains information about an engine
type EngineInfo struct {
	ID       string
	Address  string
	Port     int
	Capacity int
	Metadata map[string]string
	LastSeen time.Time
}

// Transport defines the interface for engine communication
type Transport interface {
	// Connect establishes connection to an engine
	Connect(ctx context.Context, engine *EngineInfo) (Connection, error)
	
	// Close closes all connections
	Close() error
}

// Connection represents a connection to an engine
type Connection interface {
	// ExecuteWorkflow executes a workflow on the engine
	ExecuteWorkflow(ctx context.Context, req *WorkflowRequest) (*WorkflowResponse, error)
	
	// ExecuteSubWorkflow executes a sub-workflow
	ExecuteSubWorkflow(ctx context.Context, req *SubWorkflowRequest) (*SubWorkflowResponse, error)
	
	// HealthCheck checks engine health
	HealthCheck(ctx context.Context) (*HealthCheckResponse, error)
	
	// StopWorkflow stops a running workflow
	StopWorkflow(ctx context.Context, executionID string) error
	
	// GetEngineStatus gets engine status
	GetEngineStatus(ctx context.Context) (*EngineStatusResponse, error)
	
	// StreamEvents streams workflow events
	StreamEvents(ctx context.Context, executionID string) (<-chan *WorkflowEvent, error)
	
	// Close closes the connection
	Close() error
}

// ServiceDiscovery defines the interface for discovering engines
type ServiceDiscovery interface {
	// Discover starts discovering engines
	Discover(ctx context.Context) (<-chan []*EngineInfo, error)
	
	// Watch watches for engine changes
	Watch(ctx context.Context, onChange func([]*EngineInfo)) error
	
	// Close stops discovery
	Close() error
}

// WorkflowRequest represents a workflow execution request
type WorkflowRequest struct {
	WorkflowID        string
	WorkflowVersion   string
	ExecutionID       string
	Inputs           map[string]string
	ParentWorkflowID string
	ParentExecutionID string
	TimeoutSeconds   int64
}

// WorkflowResponse represents a workflow execution response
type WorkflowResponse struct {
	ExecutionID       string
	Success           bool
	ErrorMessage      string
	Outputs           map[string]string
	DurationNanos     int64
	NodeResults       []*NodeResult
}

// NodeResult represents the result of a node execution
type NodeResult struct {
	NodeID      string
	Status      string
	Outputs     map[string]string
	ErrorMessage string
}

// SubWorkflowRequest represents a sub-workflow execution request
type SubWorkflowRequest struct {
	SubWorkflowID     string
	SubWorkflowVersion string
	ParentWorkflowID  string
	ParentExecutionID string
	ExecutionID       string
	Inputs            map[string]string
	TimeoutSeconds    int64
}

// SubWorkflowResponse represents a sub-workflow execution response
type SubWorkflowResponse struct {
	ExecutionID   string
	Success        bool
	ErrorMessage   string
	Outputs        map[string]string
	DurationNanos  int64
}

// HealthCheckResponse contains health check information
type HealthCheckResponse struct {
	Healthy        bool
	Status         string
	ActiveWorkflows int
	Capacity       int
	Metadata       map[string]string
}

// EngineStatusResponse contains engine status
type EngineStatusResponse struct {
	EngineID       string
	Status         string
	ActiveWorkflows int
	Capacity       int
	RunningWorkflows []string
	Metadata       map[string]string
}

// WorkflowEvent represents a workflow execution event
type WorkflowEvent struct {
	EventType    string
	ExecutionID  string
	WorkflowID    string
	NodeID        string
	Status        string
	Data          map[string]string
	Timestamp     int64
}

