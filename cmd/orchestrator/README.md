# Orchestrator gRPC Server

The orchestrator gRPC server exposes the ManagementService for workflow management.

## Prerequisites

1. **Generate Proto Files**: Run `./generate-proto.sh` in the project root

## Building

```bash
go build -o bin/orchestrator-server ./cmd/orchestrator
```

## Running

```bash
./bin/orchestrator-server -port 50051 -address 0.0.0.0
```

## Configuration

The orchestrator uses environment variables for configuration (see `orchestrator/config.go`).

## Services Exposed

After proto generation, the following services will be available:

- **ManagementService**: Workflow management operations
  - `RegisterWorkflow`
  - `UpdateWorkflow`
  - `DeleteWorkflow`
  - `ListWorkflows`
  - `GetWorkflow`

## Next Steps After Proto Generation

1. Uncomment proto imports in `cmd/orchestrator/main.go`
2. Uncomment `proto.RegisterManagementServiceServer` call
3. Rebuild and test

