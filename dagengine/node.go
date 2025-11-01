package dagengine

import (
    "sync"
)

// Node represents a single step in the DAG.
type Node struct {
    ID           string
    Dependencies []string            // IDs of prerequisite nodes
    Task         Executor            // The actual logic runner (e.g., LuaExecutor)
    Children     []string            // IDs of nodes that depend on this one
    // Internal state for the scheduler
    Result       map[string]interface{}
    Status       string              // "PENDING", "RUNNING", "COMPLETED", "FAILED"
    ReadyCounter int                 // Tracks unfulfilled dependencies
    mu           sync.RWMutex        // Lock for thread-safe state updates
}

// NewNode is a constructor for creating a Node instance.
func NewNode(id string, deps []string, task Executor) *Node {
    return &Node{
        ID: id,
        Dependencies: deps,
        Task: task,
        Status: "PENDING",
        ReadyCounter: len(deps), // Initialize counter based on dependencies
        Result: make(map[string]interface{}),
    }
}