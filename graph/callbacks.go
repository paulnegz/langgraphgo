package graph

import (
	"context"
	"time"
)

// CallbackHandler defines the interface for handling graph execution callbacks
// This matches Python's LangChain callback pattern
type CallbackHandler interface {
	// Chain callbacks (for graph/workflow execution)
	OnChainStart(ctx context.Context, serialized map[string]interface{}, inputs map[string]interface{}, runID string, parentRunID *string, tags []string, metadata map[string]interface{})
	OnChainEnd(ctx context.Context, outputs map[string]interface{}, runID string)
	OnChainError(ctx context.Context, err error, runID string)

	// LLM callbacks (for AI model calls)
	OnLLMStart(ctx context.Context, serialized map[string]interface{}, prompts []string, runID string, parentRunID *string, tags []string, metadata map[string]interface{})
	OnLLMEnd(ctx context.Context, response interface{}, runID string)
	OnLLMError(ctx context.Context, err error, runID string)

	// Tool callbacks (for tool/function calls)
	OnToolStart(ctx context.Context, serialized map[string]interface{}, inputStr string, runID string, parentRunID *string, tags []string, metadata map[string]interface{})
	OnToolEnd(ctx context.Context, output string, runID string)
	OnToolError(ctx context.Context, err error, runID string)

	// Retriever callbacks (for data retrieval operations)
	OnRetrieverStart(ctx context.Context, serialized map[string]interface{}, query string, runID string, parentRunID *string, tags []string, metadata map[string]interface{})
	OnRetrieverEnd(ctx context.Context, documents []interface{}, runID string)
	OnRetrieverError(ctx context.Context, err error, runID string)
}

// Config represents configuration for graph invocation
// This matches Python's config dict pattern
type Config struct {
	// Callbacks to be invoked during execution
	Callbacks []CallbackHandler `json:"callbacks"`

	// Metadata to attach to the execution
	Metadata map[string]interface{} `json:"metadata"`

	// Tags to categorize the execution
	Tags []string `json:"tags"`

	// Configurable parameters for the execution
	Configurable map[string]interface{} `json:"configurable"`

	// RunName for this execution
	RunName string `json:"run_name"`

	// Timeout for the execution
	Timeout *time.Duration `json:"timeout"`
}

// NoOpCallbackHandler provides a no-op implementation of CallbackHandler
type NoOpCallbackHandler struct{}

func (n *NoOpCallbackHandler) OnChainStart(ctx context.Context, serialized map[string]interface{}, inputs map[string]interface{}, runID string, parentRunID *string, tags []string, metadata map[string]interface{}) {
}
func (n *NoOpCallbackHandler) OnChainEnd(ctx context.Context, outputs map[string]interface{}, runID string) {
}
func (n *NoOpCallbackHandler) OnChainError(ctx context.Context, err error, runID string) {}
func (n *NoOpCallbackHandler) OnLLMStart(ctx context.Context, serialized map[string]interface{}, prompts []string, runID string, parentRunID *string, tags []string, metadata map[string]interface{}) {
}
func (n *NoOpCallbackHandler) OnLLMEnd(ctx context.Context, response interface{}, runID string) {}
func (n *NoOpCallbackHandler) OnLLMError(ctx context.Context, err error, runID string)          {}
func (n *NoOpCallbackHandler) OnToolStart(ctx context.Context, serialized map[string]interface{}, inputStr string, runID string, parentRunID *string, tags []string, metadata map[string]interface{}) {
}
func (n *NoOpCallbackHandler) OnToolEnd(ctx context.Context, output string, runID string) {}
func (n *NoOpCallbackHandler) OnToolError(ctx context.Context, err error, runID string)   {}
func (n *NoOpCallbackHandler) OnRetrieverStart(ctx context.Context, serialized map[string]interface{}, query string, runID string, parentRunID *string, tags []string, metadata map[string]interface{}) {
}
func (n *NoOpCallbackHandler) OnRetrieverEnd(ctx context.Context, documents []interface{}, runID string) {
}
func (n *NoOpCallbackHandler) OnRetrieverError(ctx context.Context, err error, runID string) {}
