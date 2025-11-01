package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HTTPTrigger executes workflows in response to HTTP requests.
type HTTPTrigger struct {
	*BaseTrigger
	port         string
	path         string
	workflowID   string
	server       *http.Server
	mu           sync.Mutex
	mux          *http.ServeMux
}

// HTTPTriggerConfig configures an HTTP trigger.
type HTTPTriggerConfig struct {
	ID         string
	Port       string // e.g., ":8080"
	Path       string // e.g., "/trigger/workflow"
	WorkflowID string
}

// NewHTTPTrigger creates a new HTTP trigger.
func NewHTTPTrigger(config HTTPTriggerConfig) *HTTPTrigger {
	mux := http.NewServeMux()
	
	return &HTTPTrigger{
		BaseTrigger: NewBaseTrigger(config.ID, "http"),
		port:        config.Port,
		path:        config.Path,
		workflowID:  config.WorkflowID,
		mux:         mux,
	}
}

// Start begins the HTTP trigger server.
func (ht *HTTPTrigger) Start(ctx context.Context, executor WorkflowExecutor) error {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	
	if ht.active {
		return fmt.Errorf("HTTP trigger %s is already active", ht.id)
	}
	
	// Setup handler
	ht.mux.HandleFunc(ht.path, func(w http.ResponseWriter, r *http.Request) {
		ht.handleRequest(w, r, executor)
	})
	
	ht.server = &http.Server{
		Addr:    ht.port,
		Handler: ht.mux,
	}
	
	ht.setActive(true)
	
	// Start server in goroutine
	go func() {
		fmt.Printf("HTTP trigger %s listening on %s%s\n", ht.id, ht.port, ht.path)
		if err := ht.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP trigger %s error: %v\n", ht.id, err)
			ht.setActive(false)
		}
	}()
	
	// Stop when context is done
	go func() {
		<-ctx.Done()
		ht.Stop()
	}()
	
	return nil
}

// Stop stops the HTTP trigger server.
func (ht *HTTPTrigger) Stop() error {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	
	if !ht.active {
		return nil
	}
	
	if ht.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ht.server.Shutdown(ctx)
	}
	
	ht.setActive(false)
	return nil
}

// handleRequest processes an incoming HTTP request.
func (ht *HTTPTrigger) handleRequest(w http.ResponseWriter, r *http.Request, executor WorkflowExecutor) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse request body
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	
	// Extract inputs (can be customized)
	inputs, ok := requestBody["inputs"].(map[string]interface{})
	if !ok {
		inputs = make(map[string]interface{})
	}
	
	// Add trigger metadata
	inputs["_trigger_type"] = "http"
	inputs["_trigger_id"] = ht.id
	inputs["_trigger_time"] = time.Now().Unix()
	inputs["_http_method"] = r.Method
	inputs["_http_path"] = r.URL.Path
	if r.RemoteAddr != "" {
		inputs["_http_remote_addr"] = r.RemoteAddr
	}
	
	// Execute workflow with timeout
	workflowCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	response, err := executor.ExecuteWorkflow(workflowCtx, ht.workflowID, inputs)
	
	// Prepare HTTP response
	resp := map[string]interface{}{
		"trigger_id":  ht.id,
		"workflow_id": ht.workflowID,
	}
	
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		resp["success"] = false
		resp["error"] = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp["success"] = response.Success
	resp["duration_ns"] = response.Duration
	resp["outputs"] = response.Outputs
	json.NewEncoder(w).Encode(resp)
}

