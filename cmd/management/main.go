package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		port            = flag.Int("port", 8080, "HTTP server port")
		orchestratorAddr = flag.String("orchestrator", "localhost:50051", "Orchestrator gRPC address")
	)
	flag.Parse()

	// Create HTTP server
	server, err := NewHTTPServer(*port, *orchestratorAddr)
	if err != nil {
		log.Fatalf("Failed to create HTTP server: %v", err)
	}
	defer server.client.Close()
	defer server.Stop()

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Printf("Management service started on port %d", *port)
	log.Printf("Orchestrator address: %s", *orchestratorAddr)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down management service...")
}

