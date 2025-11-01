package main

import (
	"strings"
	"testing"

	"github.com/gbasilveira/dag-engine/spec"
)

func TestParseYAML(t *testing.T) {
	yamlContent := `
apiVersion: workflows/v1
kind: Workflow
metadata:
  id: "test-workflow"
  name: "Test Workflow"
  version: "1.0.0"
spec:
  nodes:
    - id: "node1"
      executor:
        type: "lua"
        code: |
          print("Hello")
      dependencies: []
`

	spec, err := ParseYAMLFromBytes([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if spec.Metadata.ID != "test-workflow" {
		t.Errorf("Expected workflow ID 'test-workflow', got '%s'", spec.Metadata.ID)
	}

	if len(spec.Spec.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(spec.Spec.Nodes))
	}

	if spec.Spec.Nodes[0].ID != "node1" {
		t.Errorf("Expected node ID 'node1', got '%s'", spec.Spec.Nodes[0].ID)
	}
}

func TestValidateYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml     string
		wantErr bool
	}{
		{
			name: "valid workflow",
			yaml: `
apiVersion: workflows/v1
kind: Workflow
metadata:
  id: "test"
  name: "Test"
  version: "1.0.0"
spec:
  nodes:
    - id: "node1"
      executor:
        type: "lua"
        code: "print('test')"
`,
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			yaml: `
kind: Workflow
metadata:
  id: "test"
  name: "Test"
  version: "1.0.0"
spec:
  nodes: []
`,
			wantErr: true,
		},
		{
			name: "missing nodes",
			yaml: `
apiVersion: workflows/v1
kind: Workflow
metadata:
  id: "test"
  name: "Test"
  version: "1.0.0"
spec:
  nodes: []
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := ParseYAMLFromBytes([]byte(tt.yaml))
			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if err := spec.Validate(); err != nil {
				if !tt.wantErr {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

func TestParseYAMLFromReader(t *testing.T) {
	yamlContent := `
apiVersion: workflows/v1
kind: Workflow
metadata:
  id: "test"
  name: "Test"
  version: "1.0.0"
spec:
  nodes:
    - id: "node1"
      executor:
        type: "lua"
        code: "print('test')"
`

	reader := strings.NewReader(yamlContent)
	spec, err := ParseYAML(reader)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if spec.Metadata.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", spec.Metadata.ID)
	}
}

