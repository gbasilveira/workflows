package orchestrator

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds configuration for the orchestrator
type Config struct {
	// Transport configuration
	TransportType string // "grpc", "nats", etc.
	
	// Kubernetes configuration
	K8sNamespace      string
	K8sServiceName    string
	K8sLabelSelector  string
	InClusterConfig   bool
	
	// Load balancer configuration
	LoadBalancerType  string // "consistent-hash", "round-robin", "least-connections"
	LoadBalancerNodes int    // Number of virtual nodes for consistent hashing
	
	// gRPC configuration
	GRPCAddress     string
	GRPCPort        int
	MaxConnections  int
	ConnectionTimeout int // seconds
	
	// Engine discovery
	EngineDiscoveryInterval int // seconds
	EngineHealthCheckInterval int // seconds
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	cfg := &Config{
		TransportType:           getEnv("TRANSPORT_TYPE", "grpc"),
		K8sNamespace:            getEnv("K8S_NAMESPACE", "default"),
		K8sServiceName:          getEnv("K8S_SERVICE_NAME", "workflow-engines"),
		K8sLabelSelector:        getEnv("K8S_LABEL_SELECTOR", "app=workflow-engine"),
		InClusterConfig:        getEnvBool("IN_CLUSTER_CONFIG", true),
		LoadBalancerType:        getEnv("LOAD_BALANCER_TYPE", "consistent-hash"),
		LoadBalancerNodes:       getEnvInt("LOAD_BALANCER_NODES", 150),
		GRPCAddress:             getEnv("GRPC_ADDRESS", "0.0.0.0"),
		GRPCPort:                getEnvInt("GRPC_PORT", 50051),
		MaxConnections:          getEnvInt("MAX_CONNECTIONS", 100),
		ConnectionTimeout:       getEnvInt("CONNECTION_TIMEOUT", 30),
		EngineDiscoveryInterval: getEnvInt("ENGINE_DISCOVERY_INTERVAL", 30),
		EngineHealthCheckInterval: getEnvInt("ENGINE_HEALTH_CHECK_INTERVAL", 10),
	}
	
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Validate checks configuration validity
func (c *Config) Validate() error {
	if c.TransportType != "grpc" {
		return fmt.Errorf("unsupported transport type: %s", c.TransportType)
	}
	
	if c.LoadBalancerType != "consistent-hash" && 
	   c.LoadBalancerType != "round-robin" && 
	   c.LoadBalancerType != "least-connections" {
		return fmt.Errorf("unsupported load balancer type: %s", c.LoadBalancerType)
	}
	
	if c.GRPCPort <= 0 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPCPort)
	}
	
	return nil
}

