# Distributed Orchestrator Architecture

## Overview

The orchestrator has been refactored to support distributed engine deployment using gRPC communication and Kubernetes service discovery. Engines are now independent services that can be deployed across multiple machines/containers.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│              Orchestrator (K8s Pod)                      │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Workflow Manager │ Load Balancer │ Service Disc │   │
│  └──────────────────────────────────────────────────┘   │
│                         │                                 │
│                    gRPC Transport                        │
│                         │                                 │
└─────────────────────────┼─────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
┌───────▼──────┐  ┌───────▼──────┐  ┌───────▼──────┐
│ Engine Pod 1│  │ Engine Pod 2│  │ Engine Pod 3│
│ (K8s Pod)   │  │ (K8s Pod)   │  │ (K8s Pod)   │
└─────────────┘  └─────────────┘  └─────────────┘
```

## Key Components

### 1. OrchestratorV2 (`orchestrator/orchestrator_v2.go`)
- Manages distributed engines via service discovery
- Load balances workflow execution across engines
- Handles workflow versioning and dependencies
- Coordinates sub-workflow execution

### 2. Transport Layer (`orchestrator/transport/`)
- **Transport Interface**: Abstract interface for engine communication
- **GRPCTransport**: gRPC implementation (primary)
- **KubernetesDiscovery**: Discovers engines via K8s API

### 3. Load Balancer (`orchestrator/loadbalancer.go`)
- Consistent hashing for engine selection
- Round-robin as alternative
- Tracks engine capacity and active workflows

### 4. Workflow Manager (`orchestrator/workflow_manager.go`)
- Manages workflow definitions and versions
- Tracks workflow dependencies
- Ensures safe workflow updates

### 5. Engine Service (`orchestrator/engine/engine_service.go`)
- Standalone engine service
- Executes workflows locally
- Tracks active workflows and capacity

### 6. Sub-Workflow Coordinator (`orchestrator/subworkflow.go`)
- Coordinates sub-workflow execution
- Routes through orchestrator for simplicity
- Handles parent-child relationships

### 7. Monitoring (`orchestrator/monitor.go`)
- Receives events via gRPC streams
- Supports multiple subscribers
- Event severity levels

## Setup Instructions

### 1. Generate Protobuf Code

The protobuf code must be generated before the system can run:

```bash
./generate-proto.sh
```

This requires:
- `protoc` installed
- `protoc-gen-go` plugin
- `protoc-gen-go-grpc` plugin

After generation, update the imports in:
- `orchestrator/transport/grpc_transport.go`
- `cmd/engine/main.go`

### 2. Build Engines

```bash
go build -o bin/engine ./cmd/engine
```

### 3. Deploy to Kubernetes

```bash
kubectl apply -f deployments/engine-deployment.yaml
kubectl apply -f deployments/orchestrator-deployment.yaml
```

## Configuration

The orchestrator is configured via environment variables:

- `K8S_NAMESPACE`: Kubernetes namespace (default: "default")
- `K8S_SERVICE_NAME`: Engine service name (default: "workflow-engines")
- `K8S_LABEL_SELECTOR`: Label selector for engines (default: "app=workflow-engine")
- `IN_CLUSTER_CONFIG`: Use in-cluster K8s config (default: true)
- `TRANSPORT_TYPE`: Transport backend (default: "grpc")
- `LOAD_BALANCER_TYPE`: Load balancer type (default: "consistent-hash")
- `GRPC_PORT`: gRPC server port (default: 50051)
- `ENGINE_DISCOVERY_INTERVAL`: Discovery interval in seconds (default: 30)

## Usage Example

```go
// Create orchestrator
cfg := orchestrator.LoadConfig()
orch, err := orchestrator.NewOrchestratorV2(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer orch.Stop()

// Register workflow
err = orch.RegisterWorkflow("workflow-1", "1.0.0", func() (*orchestrator.WorkflowDefinition, error) {
    // Build workflow definition
    return def, nil
}, nil)

// Execute workflow
response, err := orch.ExecuteWorkflow(ctx, "workflow-1", map[string]interface{}{
    "input": "value",
})
```

## Sub-Workflows

Sub-workflows are executed through the orchestrator for simplicity and versioning safety:

```go
coordinator := orch.GetSubWorkflowCoordinator()
executionID, err := coordinator.ExecuteSubWorkflow(
    ctx,
    "sub-workflow-1",
    "1.0.0",
    "parent-workflow-1",
    "parent-exec-123",
    inputs,
)
```

## Workflow Versioning

Workflows use semantic versioning. The system prevents unsafe updates:

- Workflows with dependents cannot be updated
- Version must be newer than current
- Dependencies are tracked automatically

## Engine Discovery

Engines are automatically discovered via Kubernetes:
- Watches for pods with label `app=workflow-engine`
- Extracts engine info from pod metadata
- Updates load balancer as engines appear/disappear

## Next Steps

1. **Generate protobuf code** (required before running)
2. **Implement gRPC service methods** in `cmd/engine/main.go`
3. **Test with local K8s cluster** (minikube, kind, etc.)
4. **Add more transport backends** (NATS, RabbitMQ) if needed
5. **Implement advanced load balancing** (latency-based, resource-based)

## Migration from Old Orchestrator

The old `Orchestrator` is still available in `orchestrator/orchestrator.go`. To migrate:

1. Replace `NewOrchestrator()` with `NewOrchestratorV2()`
2. Update workflow registration to include version
3. Triggers now use `WorkflowExecutor` interface (compatible)
4. Engines no longer need manual registration

