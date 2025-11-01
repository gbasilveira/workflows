# Setup Instructions

This document provides step-by-step instructions to complete the setup and make the distributed orchestrator fully functional.

## Step 1: Install Protocol Buffers Compiler

The protobuf code generator requires `protoc`. See `INSTALL_PROTOC.md` for detailed installation instructions.

**Quick install (Fedora/RHEL):**
```bash
sudo dnf install protobuf-compiler protobuf-devel
```

**Verify installation:**
```bash
protoc --version
```

## Step 2: Install Go Protobuf Plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Verify plugins are in PATH:**
```bash
which protoc-gen-go
which protoc-gen-go-grpc
```

If not found, add Go bin directory to PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## Step 3: Generate Protobuf Code

```bash
./generate-proto.sh
```

This will create Go code in `orchestrator/proto/gen/`.

**Expected output:**
```
Generating protobuf code...
Protobuf code generated in orchestrator/proto/gen
```

## Step 4: Fix Dependencies

If you encounter dependency issues:

```bash
# Clean module cache
go clean -modcache

# Update dependencies
go mod tidy

# Verify build
go build ./...
```

## Step 5: Complete gRPC Implementation

After protobuf code generation, you need to:

### A. Update `orchestrator/transport/grpc_transport.go`

1. Uncomment the import for generated proto code:
```go
import (
    proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
)
```

2. Uncomment the client initialization in `grpcConnection`:
```go
client: proto.NewEngineServiceClient(conn),
```

3. Replace placeholder implementations with real ones from `grpc_transport_impl.go`:
   - Copy the implementations from the comments in `grpc_transport_impl.go`
   - Replace the TODO sections in `grpc_transport.go`

### B. Update `cmd/engine/main.go`

1. Uncomment the import for generated proto code:
```go
import (
    proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
)
```

2. Uncomment and implement the gRPC server:
   - Copy the `engineGRPCServer` implementation from `cmd/engine/grpc_server_impl.go`
   - Register the server:
```go
proto.RegisterEngineServiceServer(grpcServer, newEngineGRPCServer(engineService))
```

3. Remove the TODO comments and placeholder struct

### C. Add Helper Functions

Create `orchestrator/transport/proto_converters.go` with conversion functions:
- Copy the converter functions from comments in `grpc_transport_impl.go`

## Step 6: Verify Build

```bash
# Build orchestrator
go build -o bin/orchestrator .

# Build engine
go build -o bin/engine ./cmd/engine

# Run tests
go test ./...
```

## Step 7: Test Locally

### Test Engine (Standalone)

```bash
./bin/engine -engine-id test-engine -port 50051 -capacity 10
```

### Test Orchestrator (with local engines)

You'll need to either:
1. Use the old orchestrator (`Orchestrator`) for local testing
2. Deploy engines in Kubernetes and use `OrchestratorV2`

## Troubleshooting

### protoc not found
- Install protoc (see `INSTALL_PROTOC.md`)
- Verify it's in PATH: `which protoc`

### protoc-gen-go not found
- Install: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
- Add Go bin to PATH: `export PATH=$PATH:$(go env GOPATH)/bin`

### Import errors after protobuf generation
- Ensure protobuf code is in `orchestrator/proto/gen/`
- Check that `go_package` option in `.proto` files matches your module path
- Run `go mod tidy`

### gRPC connection errors
- Verify engines are running and accessible
- Check firewall/network settings
- Verify port numbers match configuration

## Next Steps After Setup

1. **Deploy to Kubernetes**: See `deployments/` directory
2. **Configure monitoring**: Set up event subscribers
3. **Register workflows**: Use workflow versioning
4. **Set up triggers**: Configure cron and HTTP triggers

## Development Workflow

1. **Modify `.proto` files** → Regenerate code with `./generate-proto.sh`
2. **Update implementations** → Follow patterns in `*_impl.go` files
3. **Test locally** → Use local orchestrator for development
4. **Deploy to K8s** → Use distributed orchestrator for production

