# Basic Example

This example demonstrates the core features of LangGraphGo in a comprehensive overview.

## Overview

The basic example showcases four main features:
1. **Basic Graph Execution** - Simple linear workflow
2. **Streaming with Listeners** - Real-time progress tracking
3. **Checkpointing** - State persistence and recovery
4. **Visualization** - Graph representation in multiple formats

## Features Demonstrated

### 1. Basic Graph Execution
- Creating a simple MessageGraph
- Adding nodes with processing functions
- Setting up edges and entry points
- Compiling and invoking the graph

### 2. Streaming with Listeners
- Using ListenableMessageGraph for real-time updates
- Adding progress listeners to nodes
- Configuring custom progress messages
- Monitoring execution with timing information

### 3. Checkpointing
- Creating checkpointable graphs for state persistence
- Configuring automatic checkpointing
- Setting maximum checkpoint limits
- Listing and managing saved checkpoints

### 4. Visualization
- Exporting graph structure
- Generating Mermaid diagrams
- Creating ASCII tree representations
- Visual debugging and documentation

## Running the Example

```bash
cd examples/basic_example
go run main.go
```

## Expected Output

The example will demonstrate each feature with clear output showing:
- Processing results from basic execution
- Real-time streaming updates
- Checkpoint creation and management
- Visual representations of the graph structure

## Code Structure

```go
// Basic execution
g := graph.NewMessageGraph()
g.AddNode("process", processingFunction)
g.AddEdge("process", graph.END)

// Streaming with listeners  
g := graph.NewListenableMessageGraph()
progressListener := graph.NewProgressListener()
node.AddListener(progressListener)

// Checkpointing
g := graph.NewCheckpointableMessageGraph()
config := graph.CheckpointConfig{
    Store: graph.NewMemoryCheckpointStore(),
    AutoSave: true,
}

// Visualization
exporter := runnable.GetGraph()
mermaidDiagram := exporter.DrawMermaid()
asciiTree := exporter.DrawASCII()
```

## Key Concepts

- **MessageGraph**: Core workflow orchestration
- **Listeners**: Real-time event handling and progress tracking
- **Checkpointing**: State persistence for long-running workflows
- **Visualization**: Graph structure representation and debugging