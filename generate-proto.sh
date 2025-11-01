#!/bin/bash
# Generate Go code from protobuf definitions

set -e

PROTO_DIR="orchestrator/proto"
OUT_DIR="orchestrator/proto/gen"

# Create output directory
mkdir -p "$OUT_DIR"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Install it from: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Check if Go plugins are installed
if ! go list -m github.com/golang/protobuf/protoc-gen-go > /dev/null 2>&1; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! go list -m google.golang.org/grpc/cmd/protoc-gen-go-grpc > /dev/null 2>&1; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Generate code
echo "Generating protobuf code..."

protoc \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR"/*.proto

echo "Protobuf code generated in $OUT_DIR"

