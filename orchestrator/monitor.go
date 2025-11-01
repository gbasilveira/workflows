package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MonitorEvent represents an event captured by the monitoring system.
type MonitorEvent struct {
	EventType   string                 // "workflow_started", "workflow_completed", "workflow_failed", "engine_status", etc.
	Timestamp   time.Time
	EngineID    string
	WorkflowID  string
	ExecutionID string
	Data        map[string]interface{}
	Severity    EventSeverity
}

// EventSeverity represents the severity level of an event.
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityError    EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// Monitor represents the monitoring system for workflows and engines.
type Monitor struct {
	events       chan *MonitorEvent
	subscribers  []chan *MonitorEvent
	subscribersMu sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	orchestrator interface {
		GetSubWorkflowCoordinator() *SubWorkflowCoordinator
	}
	grpcStreams  map[string]context.CancelFunc // engineID -> cancel function
	grpcStreamsMu sync.RWMutex
}

// NewMonitor creates a new monitoring system.
func NewMonitor(ctx context.Context, orchestrator interface {
	GetSubWorkflowCoordinator() *SubWorkflowCoordinator
}) *Monitor {
	monitorCtx, cancel := context.WithCancel(ctx)
	return &Monitor{
		events:       make(chan *MonitorEvent, 1000),
		subscribers:  make([]chan *MonitorEvent, 0),
		ctx:          monitorCtx,
		cancel:       cancel,
		orchestrator: orchestrator,
		grpcStreams:  make(map[string]context.CancelFunc),
	}
}

// Start begins monitoring the orchestrator.
func (m *Monitor) Start() error {
	// Start event processor
	m.wg.Add(1)
	go m.processEvents()
	
	return nil
}

// Stop stops the monitoring system.
func (m *Monitor) Stop() {
	m.cancel()
	close(m.events)
	
	// Cancel all gRPC streams
	m.grpcStreamsMu.Lock()
	for _, cancel := range m.grpcStreams {
		cancel()
	}
	m.grpcStreams = nil
	m.grpcStreamsMu.Unlock()
	
	m.wg.Wait()
	
	// Close all subscriber channels
	m.subscribersMu.Lock()
	for _, sub := range m.subscribers {
		close(sub)
	}
	m.subscribers = nil
	m.subscribersMu.Unlock()
}

// Subscribe creates a new subscription channel for monitor events.
func (m *Monitor) Subscribe() <-chan *MonitorEvent {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	
	subChan := make(chan *MonitorEvent, 100)
	m.subscribers = append(m.subscribers, subChan)
	return subChan
}

// GetEventChannel returns the internal event channel (for direct access).
func (m *Monitor) GetEventChannel() chan *MonitorEvent {
	return m.events
}

// RecordEvent records a new monitoring event.
func (m *Monitor) RecordEvent(event *MonitorEvent) {
	select {
	case m.events <- event:
	default:
		// Channel is full, log warning (could also use a separate error channel)
		fmt.Printf("Monitor: event channel full, dropping event: %+v\n", event)
	}
}

// AttachToEngineGRPC attaches monitoring to an engine's gRPC event stream.
func (m *Monitor) AttachToEngineGRPC(engineID string, eventStream <-chan *WorkflowEvent) error {
	m.grpcStreamsMu.Lock()
	streamCtx, cancel := context.WithCancel(m.ctx)
	m.grpcStreams[engineID] = cancel
	m.grpcStreamsMu.Unlock()
	
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			m.grpcStreamsMu.Lock()
			delete(m.grpcStreams, engineID)
			m.grpcStreamsMu.Unlock()
		}()
		
		for {
			select {
			case <-streamCtx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}
				
				// Convert workflow event to monitor event
				severity := SeverityInfo
				if event.Status == "FAILED" || event.Status == "ERROR" {
					severity = SeverityError
				}
				
				monitorEvent := &MonitorEvent{
					EventType:   event.EventType,
					Timestamp:   time.Unix(event.Timestamp, 0),
					EngineID:    engineID,
					WorkflowID:  event.WorkflowID,
					ExecutionID: event.ExecutionID,
					Data:        convertMap(event.Data),
					Severity:    severity,
				}
				
				m.RecordEvent(monitorEvent)
			}
		}
	}()
	
	return nil
}

// processEvents processes events and distributes them to subscribers.
func (m *Monitor) processEvents() {
	defer m.wg.Done()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case event, ok := <-m.events:
			if !ok {
				return
			}
			
			// Distribute to all subscribers
			m.subscribersMu.RLock()
			for _, sub := range m.subscribers {
				select {
				case sub <- event:
				default:
					// Subscriber channel is full, skip
				}
			}
			m.subscribersMu.RUnlock()
		}
	}
}

// convertMap converts map[string]string to map[string]interface{}
func convertMap(data map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}
	return result
}

// WorkflowEvent represents a workflow execution event (from gRPC stream)
type WorkflowEvent struct {
	EventType   string
	ExecutionID string
	WorkflowID  string
	NodeID      string
	Status      string
	Data        map[string]string
	Timestamp   int64
}
