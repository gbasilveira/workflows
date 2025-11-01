package transport

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	
	// Uncomment after running ./generate-proto.sh:
	// proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
)

// GRPCTransport implements Transport using gRPC
type GRPCTransport struct {
	dialOptions []grpc.DialOption
	connections map[string]*grpcConnection
	mu          sync.RWMutex
	timeout     time.Duration
}

// NewGRPCTransport creates a new gRPC transport
func NewGRPCTransport(timeout time.Duration) *GRPCTransport {
	return &GRPCTransport{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			}),
		},
		connections: make(map[string]*grpcConnection),
		timeout:     timeout,
	}
}

// Connect establishes a gRPC connection to an engine
func (gt *GRPCTransport) Connect(ctx context.Context, engine *EngineInfo) (Connection, error) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	// Check if connection already exists
	address := fmt.Sprintf("%s:%d", engine.Address, engine.Port)
	if conn, exists := gt.connections[engine.ID]; exists {
		return conn, nil
	}
	
	// Create new connection
	conn, err := grpc.DialContext(ctx, address, gt.dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial engine %s: %w", engine.ID, err)
	}
	
	grpcConn := &grpcConnection{
		engineID: engine.ID,
		conn:     conn,
		// TODO: Uncomment after running ./generate-proto.sh:
		// client: proto.NewEngineServiceClient(conn),
	}
	
	gt.connections[engine.ID] = grpcConn
	
	return grpcConn, nil
}

// Close closes all connections
func (gt *GRPCTransport) Close() error {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	for engineID, conn := range gt.connections {
		if err := conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection to %s: %w", engineID, err)
		}
		delete(gt.connections, engineID)
	}
	
	return nil
}

// grpcConnection implements Connection using gRPC
type grpcConnection struct {
	engineID string
	conn     *grpc.ClientConn
	// TODO: Uncomment after running ./generate-proto.sh:
	// client   proto.EngineServiceClient
	mu       sync.RWMutex
}

// ExecuteWorkflow executes a workflow via gRPC
func (gc *grpcConnection) ExecuteWorkflow(ctx context.Context, req *WorkflowRequest) (*WorkflowResponse, error) {
	// TODO: Implement after protobuf generation
	// ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	// defer cancel()
	//
	// protoReq := &proto.WorkflowRequest{
	//     WorkflowId:      req.WorkflowID,
	//     WorkflowVersion: req.WorkflowVersion,
	//     ExecutionId:     req.ExecutionID,
	//     Inputs:          req.Inputs,
	//     ParentWorkflowId: req.ParentWorkflowID,
	//     ParentExecutionId: req.ParentExecutionID,
	//     TimeoutSeconds:  req.TimeoutSeconds,
	// }
	//
	// resp, err := gc.client.ExecuteWorkflow(ctx, protoReq)
	// if err != nil {
	//     return nil, err
	// }
	//
	// return convertProtoWorkflowResponse(resp), nil
	
	// Placeholder implementation
	return &WorkflowResponse{
		ExecutionID:   req.ExecutionID,
		Success:      false,
		ErrorMessage: "protobuf code not generated yet",
	}, fmt.Errorf("protobuf code generation required")
}

// ExecuteSubWorkflow executes a sub-workflow via gRPC
func (gc *grpcConnection) ExecuteSubWorkflow(ctx context.Context, req *SubWorkflowRequest) (*SubWorkflowResponse, error) {
	// TODO: Implement after protobuf generation
	return &SubWorkflowResponse{
		ExecutionID:   req.ExecutionID,
		Success:       false,
		ErrorMessage:  "protobuf code not generated yet",
	}, fmt.Errorf("protobuf code generation required")
}

// HealthCheck checks engine health via gRPC
func (gc *grpcConnection) HealthCheck(ctx context.Context) (*HealthCheckResponse, error) {
	// TODO: Implement after protobuf generation
	// ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()
	//
	// resp, err := gc.client.HealthCheck(ctx, &proto.HealthCheckRequest{})
	// if err != nil {
	//     return nil, err
	// }
	//
	// return convertProtoHealthCheckResponse(resp), nil
	
	return &HealthCheckResponse{
		Healthy: false,
		Status:  "protobuf code not generated",
	}, fmt.Errorf("protobuf code generation required")
}

// StopWorkflow stops a workflow via gRPC
func (gc *grpcConnection) StopWorkflow(ctx context.Context, executionID string) error {
	// TODO: Implement after protobuf generation
	return fmt.Errorf("protobuf code generation required")
}

// GetEngineStatus gets engine status via gRPC
func (gc *grpcConnection) GetEngineStatus(ctx context.Context) (*EngineStatusResponse, error) {
	// TODO: Implement after protobuf generation
	return nil, fmt.Errorf("protobuf code generation required")
}

// StreamEvents streams workflow events via gRPC
func (gc *grpcConnection) StreamEvents(ctx context.Context, executionID string) (<-chan *WorkflowEvent, error) {
	// TODO: Implement after protobuf generation
	eventCh := make(chan *WorkflowEvent)
	close(eventCh)
	return eventCh, fmt.Errorf("protobuf code generation required")
}

// Close closes the gRPC connection
func (gc *grpcConnection) Close() error {
	return gc.conn.Close()
}

