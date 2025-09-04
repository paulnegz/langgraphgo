# Langfuse Tracing Example

This example demonstrates how to integrate Langfuse observability and tracing with LangGraphGo workflows.

## Overview

This example shows:
- Setting up Langfuse tracing for LangGraphGo workflows
- Automatic trace generation for graph execution
- Node-level span creation and tracking
- Edge traversal event logging

## Prerequisites

1. **Langfuse Account**: Sign up at [Langfuse](https://langfuse.com)
2. **API Credentials**: Get your public and secret keys from the Langfuse dashboard
3. **Environment Variables**: Set your credentials

```bash
export LANGFUSE_PUBLIC_KEY="your-public-key"
export LANGFUSE_SECRET_KEY="your-secret-key"
export LANGFUSE_HOST="https://cloud.langfuse.com"  # optional
```

## Features Demonstrated

### Automatic Tracing
- **Graph-Level Traces**: Entire workflow execution tracking
- **Node-Level Spans**: Individual node execution spans
- **Edge Events**: Transition tracking between nodes
- **Error Capture**: Automatic error logging and tracing

### Metadata Enrichment
- **User Context**: User ID and session tracking
- **Custom Metadata**: Additional workflow context
- **Timing Information**: Execution duration tracking
- **State Snapshots**: Input/output state capture

## Running the Example

```bash
cd examples/langfuse_tracing
export LANGFUSE_PUBLIC_KEY="your-public-key"
export LANGFUSE_SECRET_KEY="your-secret-key"
go run langfuse_tracing.go
```

## Code Structure

```go
// Create graph with tracing
g := graph.NewMessageGraph()

// Add nodes (will be automatically traced)
g.AddNode("start", startFunction)
g.AddNode("process", processFunction)
g.AddNode("finish", finishFunction)

// Compile graph
runnable, _ := g.Compile()

// Create tracer with Langfuse hook
tracer := graph.NewTracer()
langfuseHook := graph.NewLangfuseHook()
tracer.AddHook(langfuseHook)

// Create traced runnable
tracedRunnable := graph.NewTracedRunnable(runnable, tracer)

// Execute with automatic tracing
result, _ := tracedRunnable.Invoke(ctx, initialState)
```

## Expected Output

```
2024/01/01 10:00:00 Starting workflow
2024/01/01 10:00:01 Processing data
2024/01/01 10:00:02 Finishing workflow
Final result: map[result:success step:completed]
```

## Langfuse Dashboard

After running the example, check your Langfuse dashboard for:
- **Traces**: Complete workflow execution traces
- **Spans**: Individual node execution details
- **Events**: Edge traversal and state transitions
- **Metadata**: Execution context and timing information

## Trace Structure

```
graph_execution (Trace)
├── node_start (Span)
│   ├── Input: {"user_id": "user123", "session_id": "session456"}
│   └── Output: {"step": "started", ...}
├── edge_start_to_process (Event)
├── node_process (Span)
│   ├── Input: {"step": "started", ...}
│   └── Output: {"step": "processed", ...}
├── edge_process_to_finish (Event)
└── node_finish (Span)
    ├── Input: {"step": "processed", ...}
    └── Output: {"step": "completed", ...}
```

## Key Concepts

### Automatic Integration
- **Zero-Code Tracing**: No manual instrumentation required
- **Hook Pattern**: Clean separation of concerns
- **Async Processing**: Non-blocking trace submission
- **Error Resilience**: Tracing failures don't affect workflow

### Observability Features
- **Performance Monitoring**: Node and graph execution times
- **State Evolution**: Track how state changes through workflow
- **Error Tracking**: Capture and categorize failures
- **User Journey**: Track user sessions across workflows

### Configuration Options
- **Environment Variables**: Easy credential management
- **Conditional Tracing**: Enable/disable based on environment
- **Custom Metadata**: Add workflow-specific context
- **Sampling**: Control trace volume for high-traffic workflows

## Use Cases

- **Production Monitoring**: Track workflow performance and errors
- **Debugging**: Analyze workflow execution and state changes
- **Analytics**: Understand user behavior and workflow usage
- **Optimization**: Identify bottlenecks and improvement opportunities