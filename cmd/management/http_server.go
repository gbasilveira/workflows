package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gbasilveira/dag-engine/orchestrator"
	"github.com/gbasilveira/dag-engine/spec"
)

// HTTPServer provides REST API endpoints for workflow management
type HTTPServer struct {
	client   *OrchestratorClient
	port     int
	mux      *http.ServeMux
	server   *http.Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port int, orchestratorAddr string) (*HTTPServer, error) {
	client, err := NewOrchestratorClient(orchestratorAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator client: %w", err)
	}

	mux := http.NewServeMux()
	srv := &HTTPServer{
		client: client,
		port:   port,
		mux:    mux,
	}

	srv.setupRoutes()

	return srv, nil
}

// setupRoutes sets up HTTP routes
func (s *HTTPServer) setupRoutes() {
	// Workflow CRUD operations
	s.mux.HandleFunc("POST /api/v1/workflows", s.handleCreateWorkflow)
	s.mux.HandleFunc("PUT /api/v1/workflows/{id}", s.handleUpdateWorkflow)
	s.mux.HandleFunc("DELETE /api/v1/workflows/{id}", s.handleDeleteWorkflow)
	s.mux.HandleFunc("GET /api/v1/workflows", s.handleListWorkflows)
	s.mux.HandleFunc("GET /api/v1/workflows/{id}", s.handleGetWorkflow)

	// Health check
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.mux,
	}

	log.Printf("Management service HTTP server listening on :%d", s.port)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleCreateWorkflow handles POST /api/v1/workflows
func (s *HTTPServer) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse YAML from request body
	yamlSpec, err := ParseYAML(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse YAML", err)
		return
	}

	// Convert YAML to WorkflowDefinition
	def, err := ConvertYAMLToWorkflowDefinition(yamlSpec)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to convert workflow definition", err)
		return
	}

	// Register workflow via gRPC
	err = s.client.RegisterWorkflow(r.Context(), def)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to register workflow", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":    true,
		"workflow_id": def.WorkflowID,
		"version":     def.Version,
		"message":     "Workflow registered successfully",
	})
}

// handleUpdateWorkflow handles PUT /api/v1/workflows/{id}
func (s *HTTPServer) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workflowID := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")

	// Parse YAML from request body
	yamlSpec, err := ParseYAML(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse YAML", err)
		return
	}

	// Ensure the workflow ID matches
	if yamlSpec.Metadata.ID != workflowID {
		respondError(w, http.StatusBadRequest, "Workflow ID mismatch", 
			fmt.Errorf("URL workflow ID %s does not match YAML workflow ID %s", workflowID, yamlSpec.Metadata.ID))
		return
	}

	// Convert YAML to WorkflowDefinition
	def, err := ConvertYAMLToWorkflowDefinition(yamlSpec)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to convert workflow definition", err)
		return
	}

	// Update workflow via gRPC
	force := r.URL.Query().Get("force") == "true"
	err = s.client.UpdateWorkflow(r.Context(), def, force)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update workflow", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"workflow_id": def.WorkflowID,
		"version":     def.Version,
		"message":     "Workflow updated successfully",
	})
}

// handleDeleteWorkflow handles DELETE /api/v1/workflows/{id}
func (s *HTTPServer) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workflowID := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
	version := r.URL.Query().Get("version")
	force := r.URL.Query().Get("force") == "true"

	err := s.client.DeleteWorkflow(r.Context(), workflowID, version, force)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete workflow", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Workflow deleted successfully",
	})
}

// handleListWorkflows handles GET /api/v1/workflows
func (s *HTTPServer) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter := r.URL.Query().Get("filter")
	workflows, err := s.client.ListWorkflows(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list workflows", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"workflows": workflows,
	})
}

// handleGetWorkflow handles GET /api/v1/workflows/{id}
func (s *HTTPServer) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workflowID := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
	version := r.URL.Query().Get("version")

	workflow, err := s.client.GetWorkflow(r.Context(), workflowID, version)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get workflow", err)
		return
	}

	if workflow == nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, workflow)
}

// handleHealth handles GET /health
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	respondJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   errorMsg,
	})
}

