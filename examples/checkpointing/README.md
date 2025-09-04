# Checkpointing Example

This example demonstrates how to use checkpointing in LangGraphGo for state persistence and recovery in long-running workflows.

## Overview

This example shows:
- Creating checkpointable graphs for state persistence
- Configuring automatic checkpointing with intervals and limits
- Resuming execution from saved checkpoints
- Managing checkpoint storage and retrieval

## Features Demonstrated

### Automatic Checkpointing
- **Memory Store**: In-memory checkpoint storage
- **Auto Save**: Automatic checkpoint creation during execution
- **Save Interval**: Time-based checkpoint frequency
- **Max Checkpoints**: Limit number of stored checkpoints

### State Management
- Custom state structures with multiple fields
- Step-by-step state evolution
- History tracking throughout execution

### Recovery Operations
- Listing all available checkpoints
- Resuming from specific checkpoint points
- State reconstruction from checkpoint data

## Running the Example

```bash
cd examples/checkpointing
go run main.go
```

## Code Structure

```go
// Define custom state structure
type ProcessState struct {
    Step    int
    Data    string
    History []string
}

// Configure checkpointing
config := graph.CheckpointConfig{
    Store:          graph.NewMemoryCheckpointStore(),
    AutoSave:       true,
    SaveInterval:   2 * time.Second,
    MaxCheckpoints: 5,
}

// Create checkpointable graph
g := graph.NewCheckpointableMessageGraph()
g.SetCheckpointConfig(config)

// Compile and execute with automatic checkpointing
runnable, _ := g.CompileCheckpointable()
result, _ := runnable.Invoke(ctx, initialState)
```

## Expected Output

```
=== Starting execution with checkpointing ===
Executing Step 1...
Executing Step 2...
Executing Step 3...

=== Execution completed ===
Final Step: 3
Final Data: Start → Step1 → Step2 → Step3
History: [Initialized Completed Step 1 Completed Step 2 Completed Step 3]

=== Created 3 checkpoints ===
Checkpoint 1: ID=checkpoint_1, Time=2024-01-01 10:00:00
Checkpoint 2: ID=checkpoint_2, Time=2024-01-01 10:00:02
Checkpoint 3: ID=checkpoint_3, Time=2024-01-01 10:00:04

=== Resuming from checkpoint checkpoint_2 ===
Resumed at Step: 2
Resumed Data: Start → Step1 → Step2
```

## Key Concepts

### Checkpoint Configuration
- **Store Types**: Memory, file-based, or database storage
- **Auto Save**: Automatic vs manual checkpoint creation
- **Intervals**: Time-based or step-based checkpointing
- **Limits**: Managing storage space and retention

### State Persistence
- **Serialization**: How state is stored and retrieved
- **Evolution**: State changes across execution steps
- **Consistency**: Ensuring checkpoint integrity

### Recovery Patterns
- **Resume Points**: Selecting appropriate checkpoints for recovery
- **State Validation**: Ensuring resumed state is valid
- **Error Handling**: Managing recovery failures

## Use Cases

- **Long-Running Workflows**: Multi-hour processing pipelines
- **Fault Tolerance**: Recovery from system failures
- **Debugging**: Examining intermediate states
- **Branching**: Testing different execution paths from same checkpoint