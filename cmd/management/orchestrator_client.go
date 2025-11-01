package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gbasilveira/dag-engine/orchestrator"
)

// OrchestratorClient is a client for communicating with the orchestrator via gRPC
type OrchestratorClient struct {
	address string
	conn    *grpc.ClientConn
	// NOTE: Will use proto client after running generate-proto.sh
	// client  proto.ManagementServiceClient
}

// NewOrchestratorClient creates a new orchestrator client
func NewOrchestratorClient(address string) (*OrchestratorClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to orchestrator: %w", err)
	}

	return &OrchestratorClient{
		address: address,
		conn:    conn,
	}, nil
}

// Close closes the gRPC connection
func (c *OrchestratorClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RegisterWorkflow registers a workflow with the orchestrator
func (c *OrchestratorClient) RegisterWorkflow(ctx context.Context, def *orchestrator.WorkflowDefinition) error {
	// TODO: Use proto client after running generate-proto.sh
	// For now, this is a placeholder that shows the interface
	// The actual implementation will call:
	// req := &proto.RegisterWorkflowRequest{Workflow: convertToProto(def)}
	// resp, err := c.client.RegisterWorkflow(ctx, req)
	_ = ctx
	_ = def
	return fmt.Errorf("gRPC client not yet implemented - need to generate proto files first")
}

// UpdateWorkflow updates a workflow
func (c *OrchestratorClient) UpdateWorkflow(ctx context.Context, def *orchestrator.WorkflowDefinition, force bool) error {
	_ = ctx
	_ = def
	_ = force
	return fmt.Errorf("gRPC client not yet implemented - need to generate proto files first")
}

// DeleteWorkflow deletes a workflow
func (c *OrchestratorClient) DeleteWorkflow(ctx context.Context, workflowID, version string, force bool) error {
	_ = ctx
	_ = workflowID
	_ = version
	_ = force
	return fmt.Errorf("gRPC client not yet implemented - need to generate proto files first")
}

// ListWorkflows lists all workflows
// Returns workflow information (using internal WorkflowInfo type from management.go)
func (c *OrchestratorClient) ListWorkflows(ctx context.Context, filter string) ([]interface{}, error) {
	_ = ctx
	_ = filter
	return nil, fmt.Errorf("gRPC client not yet implemented - need to generate proto files first")
}

// GetWorkflow retrieves a workflow
func (c *OrchestratorClient) GetWorkflow(ctx context.Context, workflowID, version string) (*orchestrator.WorkflowDefinition, error) {
	_ = ctx
	_ = workflowID
	_ = version
	return nil, fmt.Errorf("gRPC client not yet implemented - need to generate proto files first")
}

