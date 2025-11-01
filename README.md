# DAG Engine - Distributed Workflow Orchestration System

A powerful, distributed workflow orchestration system built in Go that executes Directed Acyclic Graph (DAG) workflows with support for distributed execution, multiple triggers, versioning, and comprehensive monitoring.

## Features

- **🔄 DAG Workflow Execution**: Execute complex workflows with dependencies between tasks
- **🚀 Distributed Architecture**: Deploy engines across multiple machines/containers with Kubernetes support
- **⚡ Multiple Triggers**: Support for cron schedules, HTTP endpoints, and extensible trigger system
- **📊 Workflow Versioning**: Semantic versioning with dependency tracking and safe update policies
- **🔀 Sub-Workflow Support**: Coordinate complex workflows with nested sub-workflows
- **⚖️ Load Balancing**: Consistent hashing and round-robin load balancing strategies
- **📡 gRPC Communication**: Efficient async communication between orchestrator and engines
- **🔍 Monitoring**: Comprehensive event monitoring with multiple subscribers
- **🐳 Kubernetes Native**: First-class Kubernetes support with automatic service discovery
- **🔌 Extensible**: Polymorphic design allows easy extension of triggers, transports, and executors

## Architecture

The system consists of two main components:

### 1. Orchestrator
- Manages workflow definitions and execution
- Discovers and load balances across engines
- Handles triggers (cron, HTTP)
- Coordinates sub-workflows
- Provides monitoring capabilities

### 2. Engines
- Independent services that execute workflows
- Can be deployed as Kubernetes pods
- Scale horizontally
- Report status back to orchestrator

```
┌─────────────────────────────────────────────────────────┐
│                    Orchestrator (K8s Pod)                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Workflow Manager │ Load Balancer │ Service Disc          │   │
│  └──────────────────────────────────────────────────┘   │
│                             │                                    │
│                        gRPC Transport                            │
│                             │                                   │
└────────────────────────┼───────────────────────────────┘
                             │
        ┌─────────────────┼──────────────────┐
        │                   │                      │
┌───────▼──────┐  ┌───────▼──────┐  ┌───────▼──────┐
│ Engine Pod 1    │  │ Engine Pod 2    │  │ Engine Pod 3    │
│ (K8s Pod)       │  │ (K8s Pod)       │  │ (K8s Pod)       │
└──────────────┘  └───────────────┘  └──────────────┘
```

## Installation

### Prerequisites

- Go 1.23.10 or later
- Kubernetes cluster (for distributed deployment)
- Protocol Buffers compiler (`protoc`) - for generating gRPC code

### Building from Source

```bash
# Clone the repository
git clone https://github.com/gbasilveira/workflows.git
cd workflows

# Install dependencies
go mod download

# Generate protobuf code (required for distributed mode)
./generate-proto.sh

# Build orchestrator
go build -o bin/orchestrator .

# Build engine
go build -o bin/engine ./cmd/engine
```

## Quick Start

### Local Development (Single Process)

```go
package main

import (
    "context"
    "github.com/gbasilveira/dag-engine/dagengine"
    "github.com/gbasilveira/dag-engine/orchestrator"
)

func main() {
    ctx := context.Background()
    
    // Create orchestrator
    orch := orchestrator.NewOrchestrator(ctx)
    defer orch.Stop()
    
    // Register an engine
    engine := dagengine.NewDAGEngine()
    orch.RegisterEngine("engine-1", engine)
    
    // Register a workflow
    workflow := &orchestrator.Workflow{
        ID: "my-workflow",
        Builder: func() (*dagengine.DAGEngine, error) {
            eng := dagengine.NewDAGEngine()
            // Add nodes...
            return eng, nil
        },
    }
    orch.RegisterWorkflow(workflow)
    
    // Execute workflow
    result, err := orch.ExecuteWorkflow(ctx, "my-workflow", map[string]interface{}{
        "input": "value",
    })
}
```

### Distributed Deployment (Kubernetes)

1. **Deploy Engines:**
```bash
kubectl apply -f deployments/engine-deployment.yaml
```

2. **Deploy Orchestrator:**
```bash
kubectl apply -f deployments/orchestrator-deployment.yaml
```

3. **Configure environment variables** (see Configuration section)

## Usage Examples

### Creating a Workflow

```go
workflow := &orchestrator.Workflow{
    ID:   "data-pipeline",
    Name: "Data Processing Pipeline",
    Builder: func() (*dagengine.DAGEngine, error) {
        engine := dagengine.NewDAGEngine()
        
        // Node A: Extract data
        nodeA := dagengine.NewNode("extract", nil, &dagengine.LuaExecutor{
            Code: `
                print("Extracting data...")
                return {data = "extracted"}
            `,
        })
        engine.AddNode(nodeA)
        
        // Node B: Transform data (depends on A)
        nodeB := dagengine.NewNode("transform", []string{"extract"}, &dagengine.LuaExecutor{
            Code: `print("Transforming data...")`,
        })
        engine.AddNode(nodeB)
        
        // Node C: Load data (depends on B)
        nodeC := dagengine.NewNode("load", []string{"transform"}, &dagengine.LuaExecutor{
            Code: `print("Loading data...")`,
        })
        engine.AddNode(nodeC)
        
        engine.PreprocessDAG()
        return engine, nil
    },
}

orch.RegisterWorkflow(workflow)
```

### Using Cron Trigger

```go
cronTrigger, _ := orchestrator.NewCronTrigger(orchestrator.CronTriggerConfig{
    ID:         "daily-job",
    Schedule:   "0 0 * * *", // Daily at midnight
    WorkflowID: "data-pipeline",
    InputsBuilder: func() map[string]interface{} {
        return map[string]interface{}{
            "date": time.Now().Format("2006-01-02"),
        }
    },
})

cronTrigger.Start(ctx, orch)
```

### Using HTTP Trigger

```go
httpTrigger := orchestrator.NewHTTPTrigger(orchestrator.HTTPTriggerConfig{
    ID:         "http-api",
    Port:       ":8080",
    Path:       "/trigger/workflow",
    WorkflowID: "data-pipeline",
})

httpTrigger.Start(ctx, orch)

// Trigger via HTTP:
// curl -X POST http://localhost:8080/trigger/workflow \
//   -H "Content-Type: application/json" \
//   -d '{"inputs": {"source": "api"}}'
```

### Distributed Execution with OrchestratorV2

```go
// Load configuration
cfg := orchestrator.LoadConfig()

// Create distributed orchestrator
orch, err := orchestrator.NewOrchestratorV2(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer orch.Stop()

// Register workflow with versioning
orch.RegisterWorkflow("workflow-1", "1.0.0", builderFunc, metadata)

// Execute - automatically load balanced across engines
response, err := orch.ExecuteWorkflow(ctx, "workflow-1", inputs)
```

### Sub-Workflow Coordination

```go
coordinator := orch.GetSubWorkflowCoordinator()

// Execute sub-workflow (routed through orchestrator)
executionID, err := coordinator.ExecuteSubWorkflow(
    ctx,
    "sub-workflow-id",
    "1.0.0",
    "parent-workflow-id",
    "parent-execution-id",
    inputs,
)

// Check status
status, err := coordinator.GetSubWorkflowStatus(executionID)
```

### Monitoring

```go
monitor := orchestrator.NewMonitor(ctx, orch)
monitor.Start()

// Subscribe to events
eventStream := monitor.Subscribe()
go func() {
    for event := range eventStream {
        fmt.Printf("[%s] %s - %s\n", 
            event.Severity, 
            event.EventType, 
            event.WorkflowID)
    }
}()

// Attach to engine gRPC stream
monitor.AttachToEngineGRPC("engine-1", eventStream)
```

## Configuration

Configuration is done via environment variables:

### Orchestrator Configuration

| Variable | Description | Default |
|----------|------------|---------|
| `K8S_NAMESPACE` | Kubernetes namespace | `default` |
| `K8S_SERVICE_NAME` | Engine service name | `workflow-engines` |
| `K8S_LABEL_SELECTOR` | Label selector for engines | `app=workflow-engine` |
| `IN_CLUSTER_CONFIG` | Use in-cluster K8s config | `true` |
| `TRANSPORT_TYPE` | Transport backend (`grpc`) | `grpc` |
| `LOAD_BALANCER_TYPE` | Load balancer type | `consistent-hash` |
| `GRPC_PORT` | gRPC server port | `50051` |
| `ENGINE_DISCOVERY_INTERVAL` | Discovery interval (seconds) | `30` |
| `ENGINE_HEALTH_CHECK_INTERVAL` | Health check interval (seconds) | `10` |

### Engine Configuration

| Variable | Description | Default |
|----------|------------|---------|
| `ENGINE_ID` | Engine identifier | Hostname |
| `PORT` | gRPC server port | `50051` |
| `CAPACITY` | Max concurrent workflows | `10` |

## Workflow Versioning

The system supports semantic versioning for workflows:

```go
// Register version 1.0.0
orch.RegisterWorkflow("my-workflow", "1.0.0", builder1, nil)

// Register version 1.1.0 (safe update)
orch.RegisterWorkflow("my-workflow", "1.1.0", builder2, nil)

// Register version 2.0.0 (will fail if dependents exist)
orch.RegisterWorkflow("my-workflow", "2.0.0", builder3, nil)
```

**Version Safety:**
- Workflows with dependents cannot be updated
- Versions must be newer than current
- Dependencies are automatically tracked

## Deployment

### Kubernetes Deployment

The project includes Kubernetes manifests for easy deployment:

**Engines:**
```bash
kubectl apply -f deployments/engine-deployment.yaml
```

**Orchestrator:**
```bash
kubectl apply -f deployments/orchestrator-deployment.yaml
```

### Docker Images

Build Docker images:

```bash
# Build orchestrator
docker build -t workflow-orchestrator:latest -f Dockerfile.orchestrator .

# Build engine
docker build -t workflow-engine:latest -f Dockerfile.engine .
```

### Production Considerations

- **High Availability**: Deploy multiple orchestrator replicas behind a load balancer
- **Resource Limits**: Set appropriate CPU/memory limits for engines
- **Persistent Storage**: Consider persistent storage for workflow state (future enhancement)
- **Monitoring**: Integrate with Prometheus/Grafana for metrics
- **Logging**: Use structured logging for production

## API Reference

### Orchestrator Interface

```go
type Orchestrator interface {
    RegisterWorkflow(workflowID, version string, builder WorkflowBuilder, metadata map[string]interface{}) error
    ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResponse, error)
    GetSubWorkflowCoordinator() *SubWorkflowCoordinator
    Stop()
}
```

### Workflow Builder

```go
type WorkflowBuilder func() (*WorkflowDefinition, error)
```

### Trigger Interface

```go
type Trigger interface {
    ID() string
    Start(ctx context.Context, executor WorkflowExecutor) error
    Stop() error
    IsActive() bool
    Type() string
}
```

See `orchestrator/proto/orchestrator.proto` for complete gRPC API definition.

## Development

### Project Structure

```
.
├── cmd/
│   └── engine/          # Standalone engine binary
├── dagengine/           # Core DAG execution engine
├── orchestrator/        # Orchestrator implementation
│   ├── transport/       # Transport abstraction layer
│   ├── engine/          # Engine service
│   └── proto/           # gRPC protocol definitions
├── deployments/         # Kubernetes manifests
└── main.go             # Example/demo application
```

### Generating Protobuf Code

```bash
# Install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
./generate-proto.sh
```

### Running Tests

```bash
go test ./...
```

### Code Organization

- **Polymorphic Design**: Interfaces for extensibility
- **Separation of Concerns**: Clear boundaries between components
- **Thread Safety**: All shared state protected by mutexes
- **Error Handling**: Comprehensive error propagation

## Extending the System

### Adding a New Trigger

```go
type CustomTrigger struct {
    *BaseTrigger
    // Custom fields
}

func (ct *CustomTrigger) Start(ctx context.Context, executor WorkflowExecutor) error {
    // Implement trigger logic
    executor.ExecuteWorkflow(ctx, workflowID, inputs)
    return nil
}

func (ct *CustomTrigger) Stop() error {
    // Cleanup
    return nil
}
```

### Adding a New Executor

```go
type CustomExecutor struct {
    // Executor configuration
}

func (ce *CustomExecutor) Execute(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
    // Execute custom logic
    return outputs, nil
}
```

### Adding a New Transport

Implement the `Transport` interface in `orchestrator/transport/transport.go`.

## Troubleshooting

### Common Issues

**Protobuf code not generated:**
- Run `./generate-proto.sh`
- Ensure `protoc` and plugins are installed

**Kubernetes discovery not working:**
- Verify RBAC permissions for service account
- Check label selectors match pod labels
- Ensure in-cluster config is correct

**Engines not discovered:**
- Check Kubernetes service and pod labels
- Verify network connectivity
- Check orchestrator logs for discovery errors

## Performance

- **Concurrent Execution**: Nodes execute concurrently when dependencies allow
- **Load Balancing**: Consistent hashing ensures even distribution
- **Resource Efficiency**: Engines only process assigned workflows
- **Scalability**: Horizontal scaling via Kubernetes

## Security Considerations

- **Network Security**: Use TLS for gRPC communication (future enhancement)
- **RBAC**: Configure appropriate Kubernetes RBAC
- **Input Validation**: Validate workflow inputs before execution
- **Resource Limits**: Set pod resource limits to prevent DoS

## Roadmap

- [ ] TLS support for gRPC
- [ ] Persistent workflow state storage
- [ ] Web UI for workflow management
- [ ] Workflow templates and marketplace
- [ ] Advanced scheduling (resource-aware, priority-based)
- [ ] Workflow rollback and version management UI
- [ ] Integration with external systems (webhooks, message queues)
- [ ] Workflow testing framework

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[Specify your license here]

## Support

For issues and questions:
- Open an issue on GitHub
- Check the documentation in `ORCHESTRATOR.md` and `DISTRIBUTED_ARCHITECTURE.md`

## Acknowledgments

Built with:
- [gRPC](https://grpc.io/) - High performance RPC framework
- [Kubernetes](https://kubernetes.io/) - Container orchestration
- [GopherLua](https://github.com/yuin/gopher-lua) - Lua scripting support
- [robfig/cron](https://github.com/robfig/cron) - Cron expression parsing

---

**Note**: The distributed architecture (OrchestratorV2) requires protobuf code generation before use. See the "Generating Protobuf Code" section for instructions.

