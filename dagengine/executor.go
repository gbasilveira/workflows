package dagengine

import "context"

// Executor defines the contract for any task run by the DAG engine.
type Executor interface {
    // Execute runs the node's logic. 
    // Inputs come from upstream dependencies. Outputs are passed downstream.
    Execute(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)
}