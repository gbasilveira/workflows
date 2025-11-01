package orchestrator

import (
	"fmt"
	"sync"
	
	"github.com/lafikl/consistent"
)

// LoadBalancer interface for selecting engines
type LoadBalancer interface {
	// AddEngine adds an engine to the load balancer
	AddEngine(engineID string, capacity int) error
	
	// RemoveEngine removes an engine from the load balancer
	RemoveEngine(engineID string) error
	
	// SelectEngine selects an engine for a given key (e.g., workflow ID)
	SelectEngine(key string) (string, error)
	
	// GetEngineCapacity returns the capacity of an engine
	GetEngineCapacity(engineID string) (int, error)
	
	// UpdateEngineCapacity updates the capacity of an engine
	UpdateEngineCapacity(engineID string, capacity int) error
	
	// ListEngines returns all engine IDs
	ListEngines() []string
}

// ConsistentHashLoadBalancer uses consistent hashing for engine selection
type ConsistentHashLoadBalancer struct {
	hashRing  *consistent.Consistent
	capacities map[string]int
	activeWorkflows map[string]int // engineID -> active count
	mu         sync.RWMutex
}

// NewConsistentHashLoadBalancer creates a new consistent hash load balancer
func NewConsistentHashLoadBalancer(virtualNodes int) *ConsistentHashLoadBalancer {
	return &ConsistentHashLoadBalancer{
		hashRing:        consistent.New(),
		capacities:       make(map[string]int),
		activeWorkflows: make(map[string]int),
	}
}

// AddEngine adds an engine to the load balancer
func (lb *ConsistentHashLoadBalancer) AddEngine(engineID string, capacity int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	lb.hashRing.Add(engineID)
	lb.capacities[engineID] = capacity
	lb.activeWorkflows[engineID] = 0
	
	return nil
}

// RemoveEngine removes an engine from the load balancer
func (lb *ConsistentHashLoadBalancer) RemoveEngine(engineID string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	lb.hashRing.Remove(engineID)
	delete(lb.capacities, engineID)
	delete(lb.activeWorkflows, engineID)
	
	return nil
}

// SelectEngine selects an engine using consistent hashing
func (lb *ConsistentHashLoadBalancer) SelectEngine(key string) (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	if lb.hashRing.Size() == 0 {
		return "", fmt.Errorf("no engines available")
	}
	
	return lb.hashRing.Get(key)
}

// GetEngineCapacity returns the capacity of an engine
func (lb *ConsistentHashLoadBalancer) GetEngineCapacity(engineID string) (int, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	capacity, exists := lb.capacities[engineID]
	if !exists {
		return 0, fmt.Errorf("engine %s not found", engineID)
	}
	
	return capacity, nil
}

// UpdateEngineCapacity updates the capacity of an engine
func (lb *ConsistentHashLoadBalancer) UpdateEngineCapacity(engineID string, capacity int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	if _, exists := lb.capacities[engineID]; !exists {
		return fmt.Errorf("engine %s not found", engineID)
	}
	
	lb.capacities[engineID] = capacity
	return nil
}

// IncrementActiveWorkflows increments the active workflow count for an engine
func (lb *ConsistentHashLoadBalancer) IncrementActiveWorkflows(engineID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	lb.activeWorkflows[engineID]++
}

// DecrementActiveWorkflows decrements the active workflow count for an engine
func (lb *ConsistentHashLoadBalancer) DecrementActiveWorkflows(engineID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	if count := lb.activeWorkflows[engineID]; count > 0 {
		lb.activeWorkflows[engineID]--
	}
}

// GetActiveWorkflows returns the active workflow count for an engine
func (lb *ConsistentHashLoadBalancer) GetActiveWorkflows(engineID string) int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	return lb.activeWorkflows[engineID]
}

// ListEngines returns all engine IDs
func (lb *ConsistentHashLoadBalancer) ListEngines() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	engines := make([]string, 0, len(lb.capacities))
	for engineID := range lb.capacities {
		engines = append(engines, engineID)
	}
	
	return engines
}

// RoundRobinLoadBalancer uses round-robin selection
type RoundRobinLoadBalancer struct {
	engines   []string
	current   int
	capacities map[string]int
	activeWorkflows map[string]int
	mu         sync.RWMutex
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		engines:         make([]string, 0),
		capacities:      make(map[string]int),
		activeWorkflows: make(map[string]int),
	}
}

// AddEngine adds an engine to the load balancer
func (lb *RoundRobinLoadBalancer) AddEngine(engineID string, capacity int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	// Check if already exists
	for _, id := range lb.engines {
		if id == engineID {
			return fmt.Errorf("engine %s already exists", engineID)
		}
	}
	
	lb.engines = append(lb.engines, engineID)
	lb.capacities[engineID] = capacity
	lb.activeWorkflows[engineID] = 0
	
	return nil
}

// RemoveEngine removes an engine from the load balancer
func (lb *RoundRobinLoadBalancer) RemoveEngine(engineID string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	for i, id := range lb.engines {
		if id == engineID {
			lb.engines = append(lb.engines[:i], lb.engines[i+1:]...)
			delete(lb.capacities, engineID)
			delete(lb.activeWorkflows, engineID)
			return nil
		}
	}
	
	return fmt.Errorf("engine %s not found", engineID)
}

// SelectEngine selects an engine using round-robin
func (lb *RoundRobinLoadBalancer) SelectEngine(key string) (string, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	if len(lb.engines) == 0 {
		return "", fmt.Errorf("no engines available")
	}
	
	// Find next available engine (has capacity)
	start := lb.current
	for i := 0; i < len(lb.engines); i++ {
		engineID := lb.engines[lb.current]
		lb.current = (lb.current + 1) % len(lb.engines)
		
		capacity := lb.capacities[engineID]
		active := lb.activeWorkflows[engineID]
		
		if active < capacity {
			return engineID, nil
		}
		
		if lb.current == start {
			// All engines at capacity, return first one anyway
			return engineID, nil
		}
	}
	
	return lb.engines[0], nil
}

// GetEngineCapacity returns the capacity of an engine
func (lb *RoundRobinLoadBalancer) GetEngineCapacity(engineID string) (int, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	capacity, exists := lb.capacities[engineID]
	if !exists {
		return 0, fmt.Errorf("engine %s not found", engineID)
	}
	
	return capacity, nil
}

// UpdateEngineCapacity updates the capacity of an engine
func (lb *RoundRobinLoadBalancer) UpdateEngineCapacity(engineID string, capacity int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	if _, exists := lb.capacities[engineID]; !exists {
		return fmt.Errorf("engine %s not found", engineID)
	}
	
	lb.capacities[engineID] = capacity
	return nil
}

// ListEngines returns all engine IDs
func (lb *RoundRobinLoadBalancer) ListEngines() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	engines := make([]string, len(lb.engines))
	copy(engines, lb.engines)
	return engines
}

