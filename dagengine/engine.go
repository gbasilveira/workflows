package dagengine

import (
    "context"
    "fmt"
    "sync"
)

// DAGEngine manages the graph structure and handles execution.
type DAGEngine struct {
    Nodes map[string]*Node 
    mu    sync.Mutex
    wg    sync.WaitGroup // Use a WaitGroup to wait for all nodes to finish
}

func NewDAGEngine() *DAGEngine {
    return &DAGEngine{
        Nodes: make(map[string]*Node),
    }
}

// Run starts the concurrent execution of the entire DAG.
func (e *DAGEngine) Run(ctx context.Context) error {
    // Basic dependency verification and cycle check (critical but omitted for brevity)
    
    // Identify and start all root nodes (those with no dependencies)
    for _, node := range e.Nodes {
        if len(node.Dependencies) == 0 {
            e.wg.Add(1) // Increment counter for each root node started
            go e.executeNode(ctx, node)
        }
    }
    
    // Block until all nodes added to the WaitGroup are finished.
    e.wg.Wait()

    // Check final status for overall success/failure
    // ...
    return nil
}

// executeNode is the concurrent worker function for a single node.
func (e *DAGEngine) executeNode(ctx context.Context, n *Node) {
    defer e.wg.Done() // Signal completion when the goroutine exits
    
    // 1. Gather Inputs (Simplified: actual logic needs careful data mapping)
    // ... logic to read n.Dependencies[i].Result into the inputs map ...

    // 2. Execute Task
    n.mu.Lock()
    n.Status = "RUNNING"
    n.mu.Unlock()
    
    result, err := n.Task.Execute(ctx, nil) // passing nil for simplified inputs
    
    // 3. Update Status and Trigger Dependents
    n.mu.Lock()
    if err != nil {
        n.Status = "FAILED"
        fmt.Printf("Node %s FAILED: %v\n", n.ID, err)
        // Add logic to cascade failure (fail-fast)
        return
    }
    
    n.Status = "COMPLETED"
    n.Result = result
    fmt.Printf("Node %s COMPLETED. Result: %v\n", n.ID, result)
    n.mu.Unlock()

    // 4. Trigger Children
	e.triggerChildren(ctx, n)
}

// triggerChildren iterates over the pre-calculated direct children.
func (e *DAGEngine) triggerChildren(ctx context.Context, parentNode *Node) {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    // Iterate over the cached list of direct children
    for _, childID := range parentNode.Children {
        childNode, exists := e.Nodes[childID]
        if !exists {
            // Should not happen if FinalizeDAG ran correctly, but good for safety
            continue
        }
        
        childNode.mu.Lock()
        
        // This is the only logic needed for concurrency control:
        childNode.ReadyCounter-- 

        fmt.Printf("Node %s dependency fulfilled by %s. ReadyCounter: %d\n", 
            childNode.ID, parentNode.ID, childNode.ReadyCounter)

        if childNode.ReadyCounter == 0 {
            if childNode.Status == "PENDING" {
                e.wg.Add(1)
                go e.executeNode(ctx, childNode)
            }
        }
        childNode.mu.Unlock()
    }
}

// AddNode registers a new Node into the graph.
// It uses a mutex to ensure thread-safe map access.
func (e *DAGEngine) AddNode(node *Node) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    // 1. Check for duplicate ID
    if _, exists := e.Nodes[node.ID]; exists {
        return fmt.Errorf("node with ID '%s' already exists", node.ID)
    }

    // 2. Basic Dependency Check
    // In a full engine, you'd check that all IDs in node.Dependencies 
    // actually exist in e.Nodes. We'll skip that for the boilerplate.

    // 3. Add the Node to the map
    e.Nodes[node.ID] = node

    return nil
}

// dagengine/engine.go

// PreprocessDAG processes all nodes to build the 'Children' list
// and runs necessary validation before execution.
func (e *DAGEngine) PreprocessDAG() error {
    e.mu.Lock()
    defer e.mu.Unlock()

    // 1. Clear existing children lists (in case of re-finalization)
    for _, node := range e.Nodes {
        node.Children = nil
    }

    // 2. Build the Children (Adjacency) list
    for childID, childNode := range e.Nodes {
        for _, parentID := range childNode.Dependencies {
            parentNode, exists := e.Nodes[parentID]
            if !exists {
                return fmt.Errorf("dependency error: node '%s' depends on non-existent node '%s'", childID, parentID)
            }
            // Append the child's ID to the parent's Children list
            parentNode.Children = append(parentNode.Children, childID)
        }
    }
    
    // 3. (Optional but recommended): Run cycle detection here.
    
    return nil
}