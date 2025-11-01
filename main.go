package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gbasilveira/dag-engine/dagengine"
	"github.com/gbasilveira/dag-engine/orchestrator"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(ctx)
	defer orch.Stop()

	// Register engines
	for i := 1; i <= 3; i++ {
		engineID := fmt.Sprintf("engine-%d", i)
		engine := dagengine.NewDAGEngine()
		if err := orch.RegisterEngine(engineID, engine); err != nil {
			log.Fatalf("Failed to register engine %s: %v", engineID, err)
		}
		fmt.Printf("Registered engine: %s\n", engineID)
	}

	// Register workflows
	workflow1 := &orchestrator.Workflow{
		ID:          "workflow-1",
		Name:        "Sample Workflow A->B->C->D",
		Description: "A simple workflow with dependencies",
		Builder: func() (*dagengine.DAGEngine, error) {
			engine := dagengine.NewDAGEngine()

			// Define Node A (Root Node)
			nodeA := dagengine.NewNode("A", nil, &dagengine.LuaExecutor{
				Code: `
					print("Running Node A...")
					print("Node A finished.")
				`,
			})
			engine.AddNode(nodeA)

			// Define Node B (Depends on A)
			nodeB := dagengine.NewNode("B", []string{"A"}, &dagengine.LuaExecutor{
				Code: `print("Running Node B after A.")`,
			})
			engine.AddNode(nodeB)

			// Define Node C (Depends on A)
			nodeC := dagengine.NewNode("C", []string{"A"}, &dagengine.LuaExecutor{
				Code: `print("Running Node C after A.")`,
			})
			engine.AddNode(nodeC)

			// Define Node D (Depends on B and C)
			nodeD := dagengine.NewNode("D", []string{"B", "C"}, &dagengine.LuaExecutor{
				Code: `print("Running Node D after B and C.")`,
			})
			engine.AddNode(nodeD)

			return engine, nil
		},
	}

	if err := orch.RegisterWorkflow(workflow1); err != nil {
		log.Fatalf("Failed to register workflow: %v", err)
	}
	fmt.Printf("Registered workflow: %s\n", workflow1.ID)

	// Create and start monitoring system
	monitor := orchestrator.NewMonitor(ctx, orch)
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	// Attach monitoring to all engines
	states := orch.GetAllEngineStates()
	for engineID := range states {
		if err := monitor.AttachToEngine(engineID); err != nil {
			log.Printf("Warning: failed to attach monitor to engine %s: %v", engineID, err)
		}
	}

	// Start monitor event subscriber
	monitorSub := monitor.Subscribe()
	go func() {
		for event := range monitorSub {
			fmt.Printf("[MONITOR] [%s] %s - Engine: %s, Workflow: %s\n",
				event.Severity, event.EventType, event.EngineID, event.WorkflowID)
			if event.Severity == orchestrator.SeverityError || event.Severity == orchestrator.SeverityCritical {
				fmt.Printf("  Error details: %+v\n", event.Data)
			}
		}
	}()

	// Create and start cron trigger (runs every 30 seconds)
	cronTrigger, err := orchestrator.NewCronTrigger(orchestrator.CronTriggerConfig{
		ID:         "cron-trigger-1",
		Schedule:   "*/30 * * * * *", // Every 30 seconds
		WorkflowID: "workflow-1",
		InputsBuilder: func() map[string]interface{} {
			return map[string]interface{}{
				"triggered_at": time.Now().Unix(),
				"source":       "cron",
			}
		},
	})
	if err != nil {
		log.Fatalf("Failed to create cron trigger: %v", err)
	}

	if err := cronTrigger.Start(ctx, orch); err != nil {
		log.Fatalf("Failed to start cron trigger: %v", err)
	}
	defer cronTrigger.Stop()
	fmt.Printf("Started cron trigger: %s (every 30 seconds)\n", cronTrigger.ID())

	// Create and start HTTP trigger
	httpTrigger := orchestrator.NewHTTPTrigger(orchestrator.HTTPTriggerConfig{
		ID:         "http-trigger-1",
		Port:       ":8080",
		Path:       "/trigger/workflow",
		WorkflowID: "workflow-1",
	})

	if err := httpTrigger.Start(ctx, orch); err != nil {
		log.Fatalf("Failed to start HTTP trigger: %v", err)
	}
	defer httpTrigger.Stop()
	fmt.Printf("Started HTTP trigger: %s at http://localhost:8080/trigger/workflow\n", httpTrigger.ID())

	// Manual execution example (runs immediately)
	fmt.Println("\n--- Manual Workflow Execution ---")
	response, err := orch.ExecuteWorkflow(ctx, "workflow-1", map[string]interface{}{
		"manual": true,
		"test":   "data",
	})
	if err != nil {
		log.Printf("Manual execution failed: %v", err)
	} else {
		fmt.Printf("Manual execution completed: Success=%v, Duration=%v\n",
			response.Success, time.Duration(response.Duration))
	}

	// Wait for interrupt signal
	fmt.Println("\nOrchestrator running. Press Ctrl+C to stop.")
	fmt.Println("You can trigger workflows via:")
	fmt.Println("  1. Cron trigger (automatically every 30 seconds)")
	fmt.Println("  2. HTTP POST to http://localhost:8080/trigger/workflow")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
}
