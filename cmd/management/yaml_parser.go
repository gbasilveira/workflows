package main

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
	"github.com/gbasilveira/dag-engine/spec"
)

// ParseYAML parses a YAML workflow specification from a reader
func ParseYAML(reader io.Reader) (*spec.WorkflowSpec, error) {
	var workflowSpec spec.WorkflowSpec
	
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&workflowSpec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Validate the parsed spec
	if err := workflowSpec.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	return &workflowSpec, nil
}

// ParseYAMLFromFile parses a YAML workflow specification from a file
func ParseYAMLFromFile(filename string) (*spec.WorkflowSpec, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	return ParseYAML(file)
}

// ParseYAMLFromBytes parses a YAML workflow specification from a byte slice
func ParseYAMLFromBytes(data []byte) (*spec.WorkflowSpec, error) {
	var workflowSpec spec.WorkflowSpec
	
	if err := yaml.Unmarshal(data, &workflowSpec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Validate the parsed spec
	if err := workflowSpec.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	return &workflowSpec, nil
}

