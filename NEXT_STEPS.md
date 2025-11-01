# Next Steps for YAML Workflow Management System

This document outlines the remaining steps to complete the YAML workflow management system implementation.

## 1. Install Protoc and Generate Proto Files

### Install Protoc
```bash
# See INSTALL_PROTOC.md for detailed instructions
# On Fedora/RHEL:
sudo dnf install protobuf-compiler

# Verify installation
protoc --version
```

### Generate Proto Code
```bash
cd /home/gbas/trabalho/workflows
./generate-proto.sh
```

This will generate Go code in `orchestrator/proto/gen/` from the proto definitions.

## 2. Complete Proto Integration

After proto generation, update the following files:

### A. Update `orchestrator/management.go`

1. Uncomment proto import:
```go
import proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
```

2. Update ManagementService struct:
```go
type ManagementService struct {
    proto.UnimplementedManagementServiceServer
    orchestrator *OrchestratorV2
}
```

3. Update method signatures to use proto types:
```go
func (ms *ManagementService) RegisterWorkflow(ctx context.Context, req *proto.RegisterWorkflowRequest) (*proto.RegisterWorkflowResponse, error) {
    // Convert proto WorkflowDefinition to internal WorkflowDefinition
    def, err := protoToWorkflowDefinition(req.Workflow)
    // ... rest of implementation
}
```

### B. Implement Proto Conversion Functions

Update `orchestrator/management_proto.go` with actual implementations:

1. Implement `workflowDefinitionToProto` to convert internal `WorkflowDefinition` to proto
2. Implement `protoToWorkflowDefinition` to convert proto to internal `WorkflowDefinition`

See comments in the file for implementation details.

### C. Update `cmd/management/orchestrator_client.go`

1. Uncomment proto import:
```go
import proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
```

2. Update OrchestratorClient struct:
```go
type OrchestratorClient struct {
    address string
    conn    *grpc.ClientConn
    client  proto.ManagementServiceClient
}
```

3. Update `NewOrchestratorClient`:
```go
client := proto.NewManagementServiceClient(conn)
```

4. Implement actual gRPC calls in each method (see `orchestrator_client_impl.go` for guidance)

### D. Update `cmd/orchestrator/main.go`

1. Uncomment proto import:
```go
import proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
```

2. Register ManagementService:
```go
proto.RegisterManagementServiceServer(grpcServer, mgmtService)
```

## 3. Build and Test

### Build Components
```bash
# Build orchestrator server
go build -o bin/orchestrator-server ./cmd/orchestrator

# Build management service
go build -o bin/management ./cmd/management

# Build engine
go build -o bin/engine ./cmd/engine
```

### Run Tests
```bash
# Run all tests
go test ./...

# Run specific test
go test ./cmd/management -v
```

### Integration Testing

1. Start orchestrator server:
```bash
./bin/orchestrator-server -port 50051
```

2. Start management service:
```bash
./bin/management -port 8080 -orchestrator localhost:50051
```

3. Register a workflow:
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/x-yaml" \
  --data-binary @examples/sample-workflow.yaml
```

4. Verify workflow registration:
```bash
curl http://localhost:8080/api/v1/workflows/data-pipeline
```

## 4. Complete Workflow Manager Methods

Update `orchestrator/workflow_manager.go` to implement:

- `DeleteWorkflow(workflowID, version string)` - Delete workflow version
- `ListWorkflows()` - List all workflows
- Enhanced metadata handling

Then update `orchestrator/management.go` to use these methods in:
- `DeleteWorkflow`
- `ListWorkflows`

## 5. Enhance Secrets Integration

### Runtime Secret Injection

Update workflow execution to inject secrets as environment variables:

1. In `orchestrator/orchestrator_v2.go`, before executing workflow:
   - Extract secret references from workflow metadata
   - Use `SecretsManager.InjectSecretsIntoEnv()` to get env vars
   - Pass env vars to workflow execution context

2. Update engine execution to support environment variables

## 6. Documentation

- [ ] Update main README.md with YAML workflow management section
- [ ] Add API documentation for REST endpoints
- [ ] Create workflow authoring guide with YAML examples
- [ ] Document secrets configuration

## 7. Future Enhancements

- [ ] ConfigMap support for workflow configuration
- [ ] Workflow templates and marketplace
- [ ] Advanced trigger configurations (webhooks, message queues)
- [ ] Workflow rollback functionality
- [ ] Audit logging for workflow changes

## File Checklist

After proto generation, verify these files are updated:

- [x] `orchestrator/proto/orchestrator.proto` - Proto definitions
- [ ] `orchestrator/proto/gen/` - Generated proto code (after running generate-proto.sh)
- [ ] `orchestrator/management.go` - Use proto types
- [ ] `orchestrator/management_proto.go` - Implement conversion functions
- [ ] `cmd/management/orchestrator_client.go` - Use proto client
- [ ] `cmd/orchestrator/main.go` - Register management service

## Testing Checklist

- [ ] Unit tests for YAML parsing
- [ ] Unit tests for proto conversions
- [ ] Integration test: Register workflow via REST API
- [ ] Integration test: Update workflow
- [ ] Integration test: List workflows
- [ ] Integration test: Get workflow
- [ ] Integration test: Delete workflow
- [ ] End-to-end test: YAML -> REST -> gRPC -> Orchestrator -> Engine

