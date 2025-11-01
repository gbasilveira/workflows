package spec

// WorkflowSpec represents the root YAML structure for a workflow definition
type WorkflowSpec struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   WorkflowMetadata `yaml:"metadata"`
	Spec       WorkflowSpecDef `yaml:"spec"`
}

// WorkflowMetadata contains workflow metadata
type WorkflowMetadata struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// WorkflowSpecDef contains the actual workflow specification
type WorkflowSpecDef struct {
	Nodes         []NodeSpec       `yaml:"nodes"`
	Triggers      *TriggersSpec     `yaml:"triggers,omitempty"`
	Configuration *ConfigSpec       `yaml:"configuration,omitempty"`
}

// NodeSpec defines a single node in the workflow
type NodeSpec struct {
	ID           string                 `yaml:"id"`
	Dependencies []string               `yaml:"dependencies,omitempty"`
	Executor     ExecutorSpec           `yaml:"executor"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty"`
}

// ExecutorSpec defines the executor for a node
type ExecutorSpec struct {
	Type   string                 `yaml:"type"` // "lua", "shell", etc.
	Code   string                 `yaml:"code,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// TriggersSpec defines trigger configurations (descriptive only, not executed by orchestrator)
type TriggersSpec struct {
	HTTP *HTTPTriggerSpec `yaml:"http,omitempty"`
	Cron *CronTriggerSpec  `yaml:"cron,omitempty"`
}

// HTTPTriggerSpec defines HTTP trigger configuration
type HTTPTriggerSpec struct {
	Port       int    `yaml:"port"`
	Path       string `yaml:"path"`
	Method     string `yaml:"method,omitempty"` // GET, POST, PUT, DELETE (default: POST)
	TimeoutSec int    `yaml:"timeout_seconds,omitempty"`
}

// CronTriggerSpec defines cron trigger configuration
type CronTriggerSpec struct {
	Schedule string                 `yaml:"schedule"` // Cron expression
	Inputs   map[string]interface{} `yaml:"inputs,omitempty"`
	Timezone string                 `yaml:"timezone,omitempty"`
}

// ConfigSpec defines workflow configuration
type ConfigSpec struct {
	Secrets    []SecretRef          `yaml:"secrets,omitempty"`
	Kubernetes *K8sConfig          `yaml:"kubernetes,omitempty"`
	Env        map[string]string  `yaml:"env,omitempty"`
}

// SecretRef references a Kubernetes secret
type SecretRef struct {
	Name      string            `yaml:"name"`                  // K8s secret name
	Namespace string            `yaml:"namespace,omitempty"`  // K8s namespace (default: current namespace)
	MountPath string            `yaml:"mount_path,omitempty"` // Path to mount secret (for file-based secrets)
	Keys      map[string]string `yaml:"keys,omitempty"`       // Map of secret keys to env var names
}

// K8sConfig defines Kubernetes-specific configuration
type K8sConfig struct {
	ResourceLimits *ResourceLimits `yaml:"resource_limits,omitempty"`
	Annotations    map[string]string `yaml:"annotations,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
	ServiceAccount string            `yaml:"service_account,omitempty"`
}

// ResourceLimits defines resource limits for workflow execution
type ResourceLimits struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}
