package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/gbasilveira/dag-engine/orchestrator/transport"
)

// OrchestratorV2 manages distributed engines using service discovery and load balancing
type OrchestratorV2 struct {
	config          *Config
	workflowManager *WorkflowManager
	loadBalancer    LoadBalancer
	transport       transport.Transport
	discovery       transport.ServiceDiscovery
	engines         map[string]*transport.EngineInfo // engineID -> EngineInfo
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	subWorkflowCoord *SubWorkflowCoordinator
	executionCounter int64
	executionCounterMu sync.Mutex
}

// NewOrchestratorV2 creates a new distributed orchestrator
func NewOrchestratorV2(ctx context.Context, cfg *Config) (*OrchestratorV2, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	orchCtx, cancel := context.WithCancel(ctx)
	
	// Create workflow manager
	wfManager := NewWorkflowManager()
	
	// Create load balancer
	var lb LoadBalancer
	switch cfg.LoadBalancerType {
	case "consistent-hash":
		lb = NewConsistentHashLoadBalancer(cfg.LoadBalancerNodes)
	case "round-robin":
		lb = NewRoundRobinLoadBalancer()
	default:
		return nil, fmt.Errorf("unsupported load balancer type: %s", cfg.LoadBalancerType)
	}
	
	// Create transport
	var trans transport.Transport
	switch cfg.TransportType {
	case "grpc":
		trans = transport.NewGRPCTransport(time.Duration(cfg.ConnectionTimeout) * time.Second)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", cfg.TransportType)
	}
	
	// Create service discovery
	var disc transport.ServiceDiscovery
	var err error
	// For now, K8s is the only implementation
	disc, err = transport.NewKubernetesDiscovery(
		cfg.K8sNamespace,
		cfg.K8sServiceName,
		cfg.K8sLabelSelector,
		cfg.InClusterConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery: %w", err)
	}
	
	orch := &OrchestratorV2{
		config:           cfg,
		workflowManager:  wfManager,
		loadBalancer:     lb,
		transport:        trans,
		discovery:        disc,
		engines:          make(map[string]*transport.EngineInfo),
		ctx:              orchCtx,
		cancel:           cancel,
		subWorkflowCoord: nil, // Will be set after orchestrator is created
	}
	
	// Create sub-workflow coordinator (will be set below)
	subCoord := NewSubWorkflowCoordinator(nil)
	orch.subWorkflowCoord = subCoord
	
	// Update the coordinator's orchestrator reference after creation
	subCoord.orchestrator = orch
	
	// Start engine discovery
	if err := orch.startDiscovery(); err != nil {
		return nil, fmt.Errorf("failed to start discovery: %w", err)
	}
	
	return orch, nil
}

// RegisterWorkflow registers a workflow with versioning
func (o *OrchestratorV2) RegisterWorkflow(workflowID, version string, builder WorkflowBuilder, metadata map[string]interface{}) error {
	return o.workflowManager.RegisterWorkflow(workflowID, version, builder, metadata)
}

// ExecuteWorkflow executes a workflow on a selected engine
func (o *OrchestratorV2) ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResponse, error) {
	// Check if workflow exists
	if !o.workflowManager.HasWorkflow(workflowID) {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	
	// Get latest version
	version, err := o.workflowManager.GetLatestVersion(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow version: %w", err)
	}
	
	// Generate execution ID
	executionID := o.nextExecutionID()
	
	// Select engine using load balancer
	engineID, err := o.loadBalancer.SelectEngine(workflowID) // Use workflowID as key for consistent hashing
	if err != nil {
		return nil, fmt.Errorf("failed to select engine: %w", err)
	}
	
	// Get engine info
	o.mu.RLock()
	engineInfo, exists := o.engines[engineID]
	o.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("engine %s not found", engineID)
	}
	
	// Build workflow definition
	def, err := o.workflowManager.BuildWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}
	
	// Convert inputs to map[string]string for transport
	inputMap := make(map[string]string)
	for k, v := range inputs {
		inputMap[k] = fmt.Sprintf("%v", v)
	}
	
	// Create transport connection
	conn, err := o.transport.Connect(ctx, engineInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to engine: %w", err)
	}
	defer conn.Close()
	
	// Create workflow request
	req := &transport.WorkflowRequest{
		WorkflowID:      workflowID,
		WorkflowVersion: version,
		ExecutionID:     executionID,
		Inputs:          inputMap,
		TimeoutSeconds:  600, // 10 minutes default
	}
	
	// Update load balancer active workflows
	o.loadBalancer.(*ConsistentHashLoadBalancer).IncrementActiveWorkflows(engineID)
	defer func() {
		o.loadBalancer.(*ConsistentHashLoadBalancer).DecrementActiveWorkflows(engineID)
	}()
	
	// Execute workflow
	startTime := time.Now()
	resp, err := conn.ExecuteWorkflow(ctx, req)
	duration := time.Since(startTime)
	
	if err != nil {
		return &WorkflowResponse{
			WorkflowID: workflowID,
			Success:    false,
			Duration:   duration.Nanoseconds(),
		}, err
	}
	
	// Convert outputs back
	outputs := make(map[string]interface{})
	for k, v := range resp.Outputs {
		outputs[k] = v
	}
	
	return &WorkflowResponse{
		WorkflowID: workflowID,
		Success:    resp.Success,
		Outputs:    outputs,
		Duration:   duration.Nanoseconds(),
		Metadata: map[string]interface{}{
			"engine_id":     engineID,
			"execution_id":  executionID,
			"version":       version,
		},
	}, nil
}

// startDiscovery starts the engine discovery process
func (o *OrchestratorV2) startDiscovery() error {
	// Use Watch for real-time updates
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		
		onChange := func(engines []*transport.EngineInfo) {
			o.mu.Lock()
			defer o.mu.Unlock()
			
			// Update engines map
			engineMap := make(map[string]bool)
			for _, engine := range engines {
				engineMap[engine.ID] = true
				existing, exists := o.engines[engine.ID]
				
				if !exists {
					// New engine
					o.engines[engine.ID] = engine
					o.loadBalancer.AddEngine(engine.ID, engine.Capacity)
					fmt.Printf("Discovered new engine: %s at %s:%d\n", engine.ID, engine.Address, engine.Port)
				} else {
					// Update existing engine
					existing.Address = engine.Address
					existing.Port = engine.Port
					existing.Capacity = engine.Capacity
					existing.LastSeen = engine.LastSeen
					o.loadBalancer.UpdateEngineCapacity(engine.ID, engine.Capacity)
				}
			}
			
			// Remove engines that are no longer present
			for engineID := range o.engines {
				if !engineMap[engineID] {
					delete(o.engines, engineID)
					o.loadBalancer.RemoveEngine(engineID)
					fmt.Printf("Removed engine: %s\n", engineID)
				}
			}
		}
		
		if err := o.discovery.Watch(o.ctx, onChange); err != nil {
			fmt.Printf("Discovery watch error: %v\n", err)
		}
	}()
	
	return nil
}

// Stop gracefully stops the orchestrator
func (o *OrchestratorV2) Stop() {
	o.cancel()
	o.wg.Wait()
	
	if o.transport != nil {
		o.transport.Close()
	}
	
	if o.discovery != nil {
		o.discovery.Close()
	}
}

// nextExecutionID generates a unique execution ID
func (o *OrchestratorV2) nextExecutionID() string {
	o.executionCounterMu.Lock()
	defer o.executionCounterMu.Unlock()
	o.executionCounter++
	return fmt.Sprintf("exec-%d-%d", time.Now().UnixNano(), o.executionCounter)
}

// GetSubWorkflowCoordinator returns the sub-workflow coordinator
func (o *OrchestratorV2) GetSubWorkflowCoordinator() *SubWorkflowCoordinator {
	return o.subWorkflowCoord
}

