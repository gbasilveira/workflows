package orchestrator

import (
	"context"
	"fmt"

	"github.com/gbasilveira/dag-engine/dagengine"
	// Uncomment after running ./generate-proto.sh:
	// proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
)

// ManagementService implements the gRPC ManagementService interface
// NOTE: Embed proto.UnimplementedManagementServiceServer after running generate-proto.sh
type ManagementService struct {
	// proto.UnimplementedManagementServiceServer
	orchestrator *OrchestratorV2
}

// ManagementServiceRequest/Response types - these will be replaced by proto types
// after running generate-proto.sh
type RegisterWorkflowRequest struct {
	Workflow *WorkflowDefinition
}

type RegisterWorkflowResponse struct {
	Success    bool
	Message    string
	WorkflowID string
	Version    string
}

type UpdateWorkflowRequest struct {
	Workflow *WorkflowDefinition
	Force    bool
}

type UpdateWorkflowResponse struct {
	Success    bool
	Message    string
	WorkflowID string
	Version    string
}

type DeleteWorkflowRequest struct {
	WorkflowID string
	Version    string
	Force      bool
}

type DeleteWorkflowResponse struct {
	Success bool
	Message string
}

type ListWorkflowsRequest struct {
	Filter string
}

type WorkflowInfo struct {
	WorkflowID  string
	Name        string
	Version     string
	Description string
	CreatedAt   int64
	UpdatedAt   int64
	Metadata    map[string]string
}

type ListWorkflowsResponse struct {
	Workflows []*WorkflowInfo
}

type GetWorkflowRequest struct {
	WorkflowID string
	Version    string
}

type GetWorkflowResponse struct {
	Found    bool
	Workflow *WorkflowDefinition
}

// NewManagementService creates a new management service
func NewManagementService(orch *OrchestratorV2) *ManagementService {
	return &ManagementService{
		orchestrator: orch,
	}
}

// RegisterWorkflow registers a workflow from a WorkflowDefinition
func (ms *ManagementService) RegisterWorkflow(ctx context.Context, req *RegisterWorkflowRequest) (*RegisterWorkflowResponse, error) {
	if req.Workflow == nil {
		return &RegisterWorkflowResponse{
			Success: false,
			Message: "workflow definition is required",
		}, nil
	}

	// Create a builder function from the definition
	def := req.Workflow
	builder := func() (*WorkflowDefinition, error) {
		return def, nil
	}

	// Extract metadata
	metadata := def.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Register workflow
	err := ms.orchestrator.RegisterWorkflow(
		def.WorkflowID,
		def.Version,
		builder,
		metadata,
	)

	if err != nil {
		return &RegisterWorkflowResponse{
			Success: false,
			Message: fmt.Sprintf("failed to register workflow: %v", err),
		}, nil
	}

	return &RegisterWorkflowResponse{
		Success:    true,
		Message:    "workflow registered successfully",
		WorkflowID: def.WorkflowID,
		Version:    def.Version,
	}, nil
}

// UpdateWorkflow updates an existing workflow
func (ms *ManagementService) UpdateWorkflow(ctx context.Context, req *UpdateWorkflowRequest) (*UpdateWorkflowResponse, error) {
	if req.Workflow == nil {
		return &UpdateWorkflowResponse{
			Success: false,
			Message: "workflow definition is required",
		}, nil
	}

	// Create a builder function from the definition
	def := req.Workflow
	builder := func() (*WorkflowDefinition, error) {
		return def, nil
	}

	// Extract metadata
	metadata := def.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// For updates, we use RegisterWorkflow which handles version checking
	// If force is true, we might need to handle dependents differently
	// For now, just register - the WorkflowManager will handle version safety
	err := ms.orchestrator.RegisterWorkflow(
		def.WorkflowID,
		def.Version,
		builder,
		metadata,
	)

	if err != nil {
		return &UpdateWorkflowResponse{
			Success: false,
			Message: fmt.Sprintf("failed to update workflow: %v", err),
		}, nil
	}

	return &UpdateWorkflowResponse{
		Success:    true,
		Message:    "workflow updated successfully",
		WorkflowID: def.WorkflowID,
		Version:    def.Version,
	}, nil
}

// DeleteWorkflow deletes a workflow
func (ms *ManagementService) DeleteWorkflow(ctx context.Context, req *DeleteWorkflowRequest) (*DeleteWorkflowResponse, error) {
	// TODO: Implement workflow deletion in WorkflowManager
	// For now, return not implemented
	return &DeleteWorkflowResponse{
		Success: false,
		Message: "workflow deletion not yet implemented",
	}, nil
}

// ListWorkflows lists all registered workflows
func (ms *ManagementService) ListWorkflows(ctx context.Context, req *ListWorkflowsRequest) (*ListWorkflowsResponse, error) {
	// TODO: Implement workflow listing in WorkflowManager
	// For now, return empty list
	return &ListWorkflowsResponse{
		Workflows: []*WorkflowInfo{},
	}, nil
}

// GetWorkflow retrieves a specific workflow
func (ms *ManagementService) GetWorkflow(ctx context.Context, req *GetWorkflowRequest) (*GetWorkflowResponse, error) {
	version := req.Version
	if version == "" {
		var err error
		version, err = ms.orchestrator.workflowManager.GetLatestVersion(req.WorkflowID)
		if err != nil {
			return &GetWorkflowResponse{
				Found: false,
			}, nil
		}
	}

	// Get workflow definition
	def, err := ms.orchestrator.workflowManager.GetWorkflowDefinition(req.WorkflowID, version)
	if err != nil {
		return &GetWorkflowResponse{
			Found: false,
		}, nil
	}

	return &GetWorkflowResponse{
		Found:    true,
		Workflow: def,
	}, nil
}

// protoToWorkflowDefinition and workflowDefinitionToProto will be implemented
// once proto files are generated. For now, we work directly with WorkflowDefinition.

// buildDAGEngineFromDefinition builds a DAGEngine from a WorkflowDefinition
func buildDAGEngineFromDefinition(def *WorkflowDefinition) (*dagengine.DAGEngine, error) {
	engine := dagengine.NewDAGEngine()

	for _, nodeDef := range def.Nodes {
		var executor dagengine.Executor

		switch nodeDef.ExecutorType {
		case "lua":
			executor = &dagengine.LuaExecutor{
				Code: nodeDef.ExecutorCode,
			}
		case "shell":
			// TODO: Implement shell executor
			return nil, fmt.Errorf("shell executor not yet implemented")
		default:
			return nil, fmt.Errorf("unsupported executor type: %s", nodeDef.ExecutorType)
		}

		node := dagengine.NewNode(nodeDef.NodeID, nodeDef.Dependencies, executor)
		engine.AddNode(node)
	}

	if err := engine.PreprocessDAG(); err != nil {
		return nil, fmt.Errorf("failed to preprocess DAG: %w", err)
	}

	return engine, nil
}

