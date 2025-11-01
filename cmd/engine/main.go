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
	
	"github.com/gbasilveira/dag-engine/orchestrator/engine"
	
	// Uncomment after running ./generate-proto.sh:
	// proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
)

var (
	engineID    = flag.String("engine-id", "", "Engine ID (required)")
	port        = flag.Int("port", 50051, "gRPC server port")
	capacity    = flag.Int("capacity", 10, "Maximum concurrent workflows")
	address     = flag.String("address", "0.0.0.0", "Server address")
)

func main() {
	flag.Parse()
	
	if *engineID == "" {
		// Try to get from environment or pod name
		*engineID = os.Getenv("ENGINE_ID")
		if *engineID == "" {
			// Use hostname as fallback
			if hostname, err := os.Hostname(); err == nil {
				*engineID = hostname
			} else {
				log.Fatal("engine-id is required (use -engine-id flag or ENGINE_ID env var)")
			}
		}
	}
	
	log.Printf("Starting workflow engine: %s", *engineID)
	log.Printf("Listening on %s:%d", *address, *port)
	log.Printf("Capacity: %d concurrent workflows", *capacity)
	
	// Create engine service
	engineService := engine.NewEngineService(*engineID, *capacity)
	
	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *address, *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	grpcServer := grpc.NewServer()
	
	// TODO: Register gRPC service after protobuf generation
	// proto.RegisterEngineServiceServer(grpcServer, &engineGRPCServer{
	//     engineService: engineService,
	// })
	
	log.Printf("gRPC server ready (protobuf code generation required for full functionality)")
	
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
	log.Println("Shutting down engine...")
	
	// Graceful shutdown
	grpcServer.GracefulStop()
	log.Println("Engine stopped")
}

// Placeholder for gRPC server implementation
// This will be implemented after protobuf code generation
type engineGRPCServer struct {
	// proto.UnimplementedEngineServiceServer
	engineService *engine.EngineService
}

// TODO: Implement gRPC service methods:
// - ExecuteWorkflow
// - ExecuteSubWorkflow  
// - HealthCheck
// - StopWorkflow
// - GetEngineStatus
// - StreamWorkflowEvents

