# Basic LLM Example

This example demonstrates how to integrate Large Language Models (LLMs) with LangGraphGo using LangChain Go.

## Overview

This example shows how to:
- Integrate OpenAI's LLM with LangGraphGo workflows
- Use LangChain Go for LLM interactions
- Handle message conversations in graph nodes
- Process AI-generated responses

## Prerequisites

1. **OpenAI API Key**: Set your OpenAI API key as an environment variable:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

2. **Dependencies**: This example uses LangChain Go:
   ```bash
   go mod tidy
   ```

## Features Demonstrated

### LLM Integration
- Creating LangChain OpenAI client
- Configuring LLM parameters (temperature, etc.)
- Handling LLM responses within graph nodes

### Message Handling
- Working with LangChain message formats
- Managing conversation context
- Appending AI responses to conversation history

### Graph Workflow
- Single-node LLM processing
- State management with message arrays
- Error handling for LLM operations

## Running the Example

```bash
cd examples/basic_llm
export OPENAI_API_KEY="your-api-key-here"
go run main.go
```

## Code Structure

```go
// Create LLM client
model, err := openai.New()

// Graph node with LLM processing
g.AddNode("generate", func(ctx context.Context, state interface{}) (interface{}, error) {
    messages := state.([]llms.MessageContent)
    
    // Generate response with LangChain
    response, err := model.GenerateContent(ctx, messages,
        llms.WithTemperature(0.7),
    )
    
    // Return updated conversation
    return append(messages, llms.TextParts("ai", response.Choices[0].Content)), nil
})
```

## Expected Output

```
AI Response: 1 + 1 equals 2.
```

## Key Concepts

- **LangChain Integration**: Using LangChain Go for LLM operations
- **Message Management**: Handling conversation context in graphs
- **State Evolution**: How conversation state grows through processing
- **Error Handling**: Managing LLM API errors in workflows

## Extensions

You can extend this example by:
- Adding multiple LLM nodes for complex conversations
- Implementing different LLM providers
- Adding conversation memory and context management
- Integrating with streaming responses