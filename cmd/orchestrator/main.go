package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/gbasilveira/dag-engine/orchestrator"
	// Uncomment after running ./generate-proto.sh:
	// proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
)

var (
	port = flag.Int("port", 50051, "gRPC server port")
	address = flag.String("address", "0.0.0.0", "Server address")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	// Load configuration
	cfg := orchestrator.LoadConfig()

	// Create orchestrator
	orch, err := orchestrator.NewOrchestratorV2(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Stop()

	// Create management service
	mgmtService := orchestrator.NewManagementService(orch)

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *address, *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register ManagementService
	// Uncomment after running ./generate-proto.sh:
	// proto.RegisterManagementServiceServer(grpcServer, mgmtService)

	log.Printf("Orchestrator gRPC server listening on %s:%d", *address, *port)
	log.Printf("Management service registered (protobuf code generation required for full functionality)")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt
	<-sigChan
	log.Println("Shutting down orchestrator...")

	// Graceful shutdown
	grpcServer.GracefulStop()
	log.Println("Orchestrator stopped")
}

