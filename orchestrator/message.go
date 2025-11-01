package orchestrator

import "time"

// MessageType represents the type of message exchanged between orchestrator and engines.
type MessageType string

const (
	MsgTypeWorkflowRequest  MessageType = "workflow_request"
	MsgTypeWorkflowResponse MessageType = "workflow_response"
	MsgTypeStatusUpdate     MessageType = "status_update"
	MsgTypeHealthCheck      MessageType = "health_check"
	MsgTypeHealthResponse   MessageType = "health_response"
	MsgTypeStop             MessageType = "stop"
	MsgTypePause            MessageType = "pause"
	MsgTypeResume           MessageType = "resume"
)

// EngineMessage represents a message sent to or received from an engine.
type EngineMessage struct {
	Type      MessageType
	EngineID  string
	Timestamp time.Time
	Payload   interface{}
	RequestID string // For tracking request/response pairs
}

// EngineStatus represents the current status of an engine.
type EngineStatus string

const (
	StatusIdle      EngineStatus = "idle"
	StatusRunning   EngineStatus = "running"
	StatusPaused    EngineStatus = "paused"
	StatusError     EngineStatus = "error"
	StatusStopped   EngineStatus = "stopped"
)

// EngineState represents the full state of an engine.
type EngineState struct {
	EngineID      string
	Status        EngineStatus
	CurrentWorkflow string
	ActiveNodes   int
	TotalNodes    int
	StartTime     *time.Time
	LastUpdate    time.Time
	Error         error
	Metadata      map[string]interface{}
}

