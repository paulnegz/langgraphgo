# Conditional Edges Example

This example demonstrates how to use conditional edges in LangGraphGo for dynamic workflow routing based on runtime conditions.

## Overview

This example shows three different patterns for conditional routing:
1. **Intent-Based Routing** - Route user requests to appropriate handlers
2. **Multi-Step Workflow** - Branch execution based on validation results
3. **Dynamic Tool Selection** - Choose tools based on task requirements

## Features Demonstrated

### Conditional Edge Patterns
- **Simple Conditions** - Basic if/else routing logic
- **Multi-Way Branching** - Route to multiple possible destinations
- **Content Analysis** - Make routing decisions based on state content
- **Default Fallbacks** - Handle unexpected conditions gracefully

### State-Based Routing
- **Message Analysis** - Route based on user input content
- **Data Validation** - Branch on validation success/failure
- **Threshold Decisions** - Route based on numeric comparisons
- **Keyword Detection** - Route based on text content analysis

## Running the Example

**Note**: This example has build constraints and may require additional setup.

```bash
cd examples/conditional_edges_example
go run -tags="" main.go
```

## Code Structure

### 1. Intent-Based Routing
```go
// Conditional routing based on user intent
g.AddConditionalEdge("analyze_intent", func(ctx context.Context, state interface{}) string {
    messages := state.([]llms.MessageContent)
    text := strings.ToLower(messages[0].Parts[0].(llms.TextContent).Text)
    
    // Route based on keywords
    if strings.Contains(text, "?") || strings.Contains(text, "what") {
        return "handle_question"
    }
    if strings.Contains(text, "please") || strings.Contains(text, "run") {
        return "handle_command"
    }
    return "handle_feedback" // Default
})
```

### 2. Multi-Step Workflow
```go
// Validation-based routing
g.AddConditionalEdge("validate", func(ctx context.Context, state interface{}) string {
    data := state.(map[string]interface{})
    if valid, ok := data["valid"].(bool); ok && valid {
        return "process"  // Continue processing
    }
    return "handle_error"  // Handle validation failure
})
```

### 3. Dynamic Tool Selection
```go
// Task-based tool selection
g.AddConditionalEdge("analyze_task", func(ctx context.Context, state interface{}) string {
    task := strings.ToLower(state.(string))
    
    if strings.Contains(task, "calculate") {
        return "calculator"
    }
    if strings.Contains(task, "search") {
        return "web_search"
    }
    return "web_search" // Default tool
})
```

## Expected Output

### Intent Routing
```
📝 Input: What is the weather today?
   → Routing to Question Handler
   ❓ Question Handler: I'll help answer your question about that.

📝 Input: Please run the diagnostic tool
   → Routing to Command Handler
   ⚡ Command Handler: Executing your command...
```

### Multi-Step Workflow
```
Test 1: Input = map[value:60]
   → Data is valid, proceeding to process
   → Large result, storing...
   Final State: map[result:120 status:processed valid:true value:60]

Test 2: Input = map[value:-5]
   → Data is invalid, handling error
   Final State: map[error:Invalid input value status:error valid:false value:-5]
```

### Dynamic Tool Selection
```
📋 Task: Calculate the compound interest
   → Selecting Calculator
   🧮 Using Calculator Tool

📋 Task: Search for best practices in Go
   → Selecting Web Search
   🔍 Using Web Search Tool
```

## Key Concepts

### Conditional Edge Functions
- **Context Access**: Full access to current execution context
- **State Inspection**: Examine current state to make routing decisions
- **String Return**: Return next node name as string
- **Runtime Evaluation**: Decisions made during execution, not compilation

### Routing Patterns
- **Content-Based**: Route based on input content analysis
- **Validation-Based**: Route based on data validation results
- **Threshold-Based**: Route based on numeric or boolean conditions
- **Classification-Based**: Route based on categorization logic

### Best Practices
- **Default Routes**: Always provide fallback routing
- **Error Handling**: Handle invalid state conditions gracefully
- **Performance**: Keep conditional logic lightweight
- **Maintainability**: Use clear, readable condition logic

## Use Cases

- **Chatbot Intent Routing**: Direct user messages to appropriate handlers
- **Data Processing Pipelines**: Route based on data quality or type
- **Multi-Modal Workflows**: Choose processing based on input format
- **Error Recovery**: Route to different recovery strategies based on error type