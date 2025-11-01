# Orchestrator Architecture

This document describes the orchestrator system that manages multiple DAG engines with bi-directional communication, triggers, and monitoring.

## Architecture Overview

The orchestrator system is built with polymorphism and extensibility in mind, allowing easy addition of new trigger types, monitoring backends, and engine implementations.

```
┌─────────────────────────────────────────────────────────────┐
│                      Orchestrator                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Engine 1     │  │   Engine 2      │  │   Engine 3   │      │
│  │ ┌──────────┐ │  │ ┌──────────┐  │  │ ┌──────────┐ │      │
│  │ │ Inbound    │ │  │ │ Inbound    │ │  │ │ Inbound  │ │      │
│  │ │ Outbound   │ │  │ │ Outbound   │ │  │ │ Outbound │ │      │
│  │ └──────────┘ │  │ └──────────┘  │  │ └──────────┘ │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │   Monitor System    │
                    │  (Event Channel)   │
                    └────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
  ┌──────▼──────┐    ┌────────▼────────┐  ┌───────▼───────┐
  │ Cron Trigger   │    │  HTTP Trigger      │  │ Future Trigger   │
  └─────────────┘    └─────────────────┘  └───────────────┘
```

## Core Components

### 1. Orchestrator (`orchestrator/orchestrator.go`)

The main orchestrator manages:
- Multiple DAG engines with bi-directional channel communication
- Workflow registration and execution
- Engine lifecycle management
- Request/response handling

**Key Methods:**
- `RegisterEngine(id, engine)` - Register a new engine
- `RegisterWorkflow(workflow)` - Register a workflow definition
- `ExecuteWorkflow(ctx, workflowID, inputs)` - Execute a workflow on an available engine
- `GetEngineOutboundChannel(engineID)` - Get engine's outbound channel for monitoring

### 2. Engine Wrapper (`orchestrator/engine_wrapper.go`)

Wraps each DAG engine with:
- **Inbound Channel**: Receives commands from orchestrator (workflow requests, stop, pause, resume)
- **Outbound Channel**: Sends responses and status updates to orchestrator/monitor
- Status tracking (idle, running, paused, error, stopped)
- Current workflow tracking

**Message Types:**
- `MsgTypeWorkflowRequest` - Execute a workflow
- `MsgTypeWorkflowResponse` - Workflow execution result
- `MsgTypeStatusUpdate` - Engine status change
- `MsgTypeHealthCheck` - Health check request
- `MsgTypeStop/Pause/Resume` - Control commands

### 3. Workflow System (`orchestrator/workflow.go`)

**Workflow** represents a reusable workflow definition:
- Contains a builder function that creates a configured DAG engine
- Can be executed multiple times with different inputs
- Metadata support for additional information

### 4. Trigger System (`orchestrator/trigger.go`)

Polymorphic trigger interface supporting multiple trigger types:

```go
type Trigger interface {
    ID() string
    Start(ctx, orchestrator) error
    Stop() error
    IsActive() bool
    Type() string
}
```

**Implemented Triggers:**

#### Cron Trigger (`orchestrator/cron_trigger.go`)
- Executes workflows on a schedule using cron syntax
- Supports dynamic input builders
- Example: `*/30 * * * * *` (every 30 seconds)

#### HTTP Trigger (`orchestrator/http_trigger.go`)
- Executes workflows via HTTP POST requests
- Accepts JSON payloads with inputs
- Returns workflow execution results as JSON

**Extending with New Triggers:**
1. Implement the `Trigger` interface
2. Embed `BaseTrigger` for common functionality
3. Register with orchestrator using `Start(ctx, orchestrator)`

### 5. Monitoring System (`orchestrator/monitor.go`)

The monitoring system provides:
- **Dedicated event channel** for all monitoring events
- **Subscription model** - multiple subscribers can listen to events
- **Automatic engine monitoring** - collects engine states periodically
- **Event types:**
  - `workflow_started`
  - `workflow_completed`
  - `workflow_failed`
  - `engine_status`
  - `status_update`
  - `health_check_response`

**Event Severities:**
- `info` - Normal operational events
- `warning` - Potentially problematic events
- `error` - Error conditions
- `critical` - Critical failures

**Usage:**
```go
monitor := orchestrator.NewMonitor(ctx, orch)
monitor.Start()

// Subscribe to events
sub := monitor.Subscribe()
go func() {
    for event := range sub {
        // Handle event
    }
}()

// Attach to specific engine
monitor.AttachToEngine("engine-1")
```

## Communication Flow

### Workflow Execution Flow

1. **Trigger Fires** (Cron/HTTP/Manual)
   ```
   Trigger → Orchestrator.ExecuteWorkflow()
   ```

2. **Orchestrator Selects Engine**
   ```
   Find idle engine → Create workflow instance → Preprocess DAG
   ```

3. **Send Workflow Request**
   ```
   Orchestrator → Engine.Inbound Channel → Engine.processMessages()
   ```

4. **Engine Executes**
   ```
   Engine.handleWorkflowRequest() → Engine.Engine.Run()
   ```

5. **Response Sent**
   ```
   Engine → Engine.Outbound Channel → Orchestrator/Monitor
   ```

6. **Monitoring**
   ```
   Engine.Outbound → Monitor.AttachToEngine() → Monitor.RecordEvent()
   Monitor → Subscribers → Event Processing
   ```

## Extensibility

### Adding a New Trigger Type

```go
type CustomTrigger struct {
    *BaseTrigger
    // Custom fields
}

func NewCustomTrigger(config CustomTriggerConfig) *CustomTrigger {
    return &CustomTrigger{
        BaseTrigger: NewBaseTrigger(config.ID, "custom"),
        // Initialize custom fields
    }
}

func (ct *CustomTrigger) Start(ctx context.Context, orchestrator *Orchestrator) error {
    // Implement trigger logic
    // Call orchestrator.ExecuteWorkflow() when triggered
}

func (ct *CustomTrigger) Stop() error {
    // Cleanup
}
```

### Adding Monitoring Backends

The monitor's subscription model allows multiple backends:

```go
// Example: Logging backend
logSub := monitor.Subscribe()
go func() {
    for event := range logSub {
        logger.Log(event)
    }
}()

// Example: Metrics backend
metricsSub := monitor.Subscribe()
go func() {
    for event := range metricsSub {
        metrics.Record(event)
    }
}()
```

## File Structure

```
orchestrator/
├── orchestrator.go      # Main orchestrator implementation
├── engine_wrapper.go    # Engine wrapper with channels
├── workflow.go          # Workflow definitions
├── message.go           # Message types for communication
├── trigger.go           # Trigger interface and base
├── cron_trigger.go      # Cron trigger implementation
├── http_trigger.go      # HTTP trigger implementation
└── monitor.go           # Monitoring system
```

## Example Usage

See `main.go` for a complete example showing:
- Creating an orchestrator with multiple engines
- Registering workflows
- Setting up cron and HTTP triggers
- Configuring monitoring with subscribers
- Manual workflow execution

## Channel Sizes

- **Engine Inbound/Outbound**: 100 messages (buffered)
- **Monitor Events**: 1000 events (buffered)
- **Monitor Subscribers**: 100 events per subscriber (buffered)

Adjust these based on your workload and memory constraints.

## Thread Safety

All components are designed for concurrent access:
- Orchestrator uses `sync.RWMutex` for engine/workflow maps
- EngineWrapper uses `sync.RWMutex` for state access
- Message channels are thread-safe
- All shared state is protected by mutexes

## Error Handling

- Engine failures are captured and reported via outbound messages
- Workflow execution errors are returned in `WorkflowResponse`
- Monitor events include error details for failed workflows
- Trigger errors are logged (consider forwarding to monitor)

