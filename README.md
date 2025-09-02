# 🦜️🔗 LangGraphGo

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/paulnegz/langgraphgo)

> 🔀 **Forked from [tmc/langgraphgo](https://github.com/tmc/langgraphgo)** - Enhanced with performance improvements and simplified API for production use.

## Changes from Original

- **Removed LangChain dependency** - Works with any LLM client (Google AI, OpenAI, Anthropic, etc.)
- **Generic state management** - Use any type as state, not just `[]llms.MessageContent`
- **Performance optimized** - Reduced overhead for production workloads
- **Simplified API** - Cleaner interface for building graphs


## Quick Start


This is a simple example of how to use the library to create a simple chatbot that uses OpenAI to generate responses.

```go
import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/paulnegz/langgraphgo/graph"
	// Use any LLM client - OpenAI, Google AI, etc.
)

// Define your state type
type ChatState struct {
	Messages []string
	Response string
}

func main() {
	g := graph.NewMessageGraph()

	// Add nodes with generic state handling
	g.AddNode("generate", func(ctx context.Context, state interface{}) (interface{}, error) {
		chatState := state.(*ChatState)
		// Call your LLM here (OpenAI, Google AI, etc.)
		chatState.Response = "Generated response"
		return chatState, nil
	})

	g.AddEdge("oracle", graph.END)
	g.SetEntryPoint("oracle")

	runnable, err := g.Compile()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// Run with your custom state
	initialState := &ChatState{
		Messages: []string{"What is 1 + 1?"},
	}
	
	res, err := runnable.Invoke(ctx, initialState)
	if err != nil {
		panic(err)
	}

	finalState := res.(*ChatState)
	fmt.Println(finalState.Response)
}
```
