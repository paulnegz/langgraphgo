package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/paulnegz/langgraphgo/graph"
)

func main() {
	// Set Langfuse credentials (or use environment variables)
	os.Setenv("LANGFUSE_PUBLIC_KEY", "your-public-key")
	os.Setenv("LANGFUSE_SECRET_KEY", "your-secret-key")
	os.Setenv("LANGFUSE_HOST", "https://cloud.langfuse.com") // optional

	// Create a simple graph
	g := graph.NewMessageGraph()

	// Add nodes
	g.AddNode("start", func(ctx context.Context, state interface{}) (interface{}, error) {
		log.Println("Starting workflow")
		stateMap := state.(map[string]interface{})
		stateMap["step"] = "started"
		return stateMap, nil
	})

	g.AddNode("process", func(ctx context.Context, state interface{}) (interface{}, error) {
		log.Println("Processing data")
		stateMap := state.(map[string]interface{})
		stateMap["step"] = "processed"
		stateMap["result"] = "success"
		return stateMap, nil
	})

	g.AddNode("finish", func(ctx context.Context, state interface{}) (interface{}, error) {
		log.Println("Finishing workflow")
		stateMap := state.(map[string]interface{})
		stateMap["step"] = "completed"
		return stateMap, nil
	})

	// Add edges
	g.SetEntryPoint("start")
	g.AddEdge("start", "process")
	g.AddEdge("process", "finish")
	g.AddEdge("finish", graph.END)

	// Compile the graph
	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Create a tracer with Langfuse hook
	tracer := graph.NewTracer()
	langfuseHook := graph.NewLangfuseHook()
	tracer.AddHook(langfuseHook)

	// Create traced runnable
	tracedRunnable := graph.NewTracedRunnable(runnable, tracer)

	// Execute with initial state
	ctx := context.Background()
	initialState := map[string]interface{}{
		"user_id":    "user123",
		"session_id": "session456",
	}

	// The Langfuse hook will automatically:
	// 1. Create a trace when the graph starts
	// 2. Create spans for each node execution
	// 3. Track edge traversals as events
	// 4. Send all data to Langfuse

	result, err := tracedRunnable.Invoke(ctx, initialState)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final result: %v\n", result)

	// Flush any pending traces
	langfuseHook.Flush()
}
