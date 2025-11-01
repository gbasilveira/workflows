# Installing Protocol Buffers Compiler (protoc)

The protobuf code generator requires `protoc` to be installed. Here are installation instructions for different platforms:

## Linux (Fedora/RHEL)

```bash
# Install protoc
sudo dnf install protobuf-compiler protobuf-devel

# Verify installation
protoc --version
```

## Linux (Ubuntu/Debian)

```bash
# Install protoc
sudo apt-get update
sudo apt-get install protobuf-compiler

# Verify installation
protoc --version
```

## macOS

```bash
# Using Homebrew
brew install protobuf

# Verify installation
protoc --version
```

## Manual Installation (All Platforms)

1. Download the latest release from: https://github.com/protocolbuffers/protobuf/releases
2. Extract the archive
3. Add the `bin` directory to your PATH

For example:
```bash
wget https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-linux-x86_64.zip
unzip protoc-25.1-linux-x86_64.zip -d protoc
export PATH=$PATH:$(pwd)/protoc/bin
```

## Install Go Plugins

After installing protoc, install the Go plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Generate Code

Once protoc is installed, run:

```bash
./generate-proto.sh
```

This will generate Go code in `orchestrator/proto/gen/`.

