package graph

import (
	"context"
	"fmt"
	"time"
)

// StateGraph represents a state-based graph similar to Python's LangGraph StateGraph
type StateGraph struct {
	// nodes is a map of node names to their corresponding Node objects
	nodes map[string]Node
	
	// edges is a slice of Edge objects representing the connections between nodes
	edges []Edge
	
	// conditionalEdges contains a map between "From" node, while "To" node is derived based on the condition
	conditionalEdges map[string]func(ctx context.Context, state interface{}) string
	
	// entryPoint is the name of the entry point node in the graph
	entryPoint string
	
	// retryPolicy defines retry behavior for failed nodes
	retryPolicy *RetryPolicy
}

// RetryPolicy defines how to handle node failures
type RetryPolicy struct {
	MaxRetries      int
	BackoffStrategy BackoffStrategy
	RetryableErrors []string
}

// BackoffStrategy defines different backoff strategies
type BackoffStrategy int

const (
	FixedBackoff BackoffStrategy = iota
	ExponentialBackoff
	LinearBackoff
)

// NewStateGraph creates a new instance of StateGraph
func NewStateGraph() *StateGraph {
	return &StateGraph{
		nodes:            make(map[string]Node),
		conditionalEdges: make(map[string]func(ctx context.Context, state interface{}) string),
	}
}

// AddNode adds a new node to the state graph with the given name and function
func (g *StateGraph) AddNode(name string, fn func(ctx context.Context, state interface{}) (interface{}, error)) {
	g.nodes[name] = Node{
		Name:     name,
		Function: fn,
	}
}

// AddEdge adds a new edge to the state graph between the "from" and "to" nodes
func (g *StateGraph) AddEdge(from, to string) {
	g.edges = append(g.edges, Edge{
		From: from,
		To:   to,
	})
}

// AddConditionalEdge adds a conditional edge where the target node is determined at runtime
func (g *StateGraph) AddConditionalEdge(from string, condition func(ctx context.Context, state interface{}) string) {
	g.conditionalEdges[from] = condition
}

// SetEntryPoint sets the entry point node name for the state graph
func (g *StateGraph) SetEntryPoint(name string) {
	g.entryPoint = name
}

// SetRetryPolicy sets the retry policy for the graph
func (g *StateGraph) SetRetryPolicy(policy *RetryPolicy) {
	g.retryPolicy = policy
}

// StateRunnable represents a compiled state graph that can be invoked
type StateRunnable struct {
	graph *StateGraph
}

// Compile compiles the state graph and returns a StateRunnable instance
func (g *StateGraph) Compile() (*StateRunnable, error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &StateRunnable{
		graph: g,
	}, nil
}

// Invoke executes the compiled state graph with the given input state
func (r *StateRunnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	state := initialState
	currentNode := r.graph.entryPoint

	for {
		if currentNode == END {
			break
		}

		node, ok := r.graph.nodes[currentNode]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, currentNode)
		}

		// Execute node with retry logic
		var err error
		state, err = r.executeNodeWithRetry(ctx, node, state)
		if err != nil {
			return nil, fmt.Errorf("error in node %s: %w", currentNode, err)
		}

		// First check for conditional edges
		nextNodeFn, hasConditional := r.graph.conditionalEdges[currentNode]
		if hasConditional {
			currentNode = nextNodeFn(ctx, state)
			if currentNode == "" {
				return nil, fmt.Errorf("conditional edge returned empty next node from %s", currentNode)
			}
			continue
		}

		// Then check regular edges
		foundNext := false
		for _, edge := range r.graph.edges {
			if edge.From == currentNode {
				currentNode = edge.To
				foundNext = true
				break
			}
		}

		if !foundNext {
			return nil, fmt.Errorf("%w: %s", ErrNoOutgoingEdge, currentNode)
		}
	}

	return state, nil
}

// executeNodeWithRetry executes a node with retry logic based on the retry policy
func (r *StateRunnable) executeNodeWithRetry(ctx context.Context, node Node, state interface{}) (interface{}, error) {
	var lastErr error
	
	maxRetries := 1 // Default: no retries
	if r.graph.retryPolicy != nil {
		maxRetries = r.graph.retryPolicy.MaxRetries + 1 // +1 for initial attempt
	}
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := node.Function(ctx, state)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if r.graph.retryPolicy != nil && attempt < maxRetries-1 {
			if r.isRetryableError(err) {
				// Apply backoff strategy
				delay := r.calculateBackoffDelay(attempt)
				if delay > 0 {
					select {
					case <-time.After(delay):
						// Continue with retry after delay
					case <-ctx.Done():
						// Context cancelled, return immediately
						return nil, ctx.Err()
					}
				}
				continue
			}
		}
		
		// If not retryable or max retries reached, return error
		break
	}
	
	return nil, lastErr
}

// isRetryableError checks if an error is retryable based on the retry policy
func (r *StateRunnable) isRetryableError(err error) bool {
	if r.graph.retryPolicy == nil {
		return false
	}
	
	errorStr := err.Error()
	for _, retryablePattern := range r.graph.retryPolicy.RetryableErrors {
		if contains(errorStr, retryablePattern) {
			return true
		}
	}
	
	return false
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(substr) > 0 && len(s) > len(substr) && 
		 (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		  findSubstring(s, substr))))
}

// findSubstring finds if substr exists in s
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculateBackoffDelay calculates the delay for retry based on the backoff strategy
func (r *StateRunnable) calculateBackoffDelay(attempt int) time.Duration {
	if r.graph.retryPolicy == nil {
		return 0
	}

	baseDelay := time.Second // Default 1 second base delay

	switch r.graph.retryPolicy.BackoffStrategy {
	case FixedBackoff:
		return baseDelay
	case ExponentialBackoff:
		// Exponential backoff: 1s, 2s, 4s, 8s, ...
		return baseDelay * time.Duration(1<<attempt)
	case LinearBackoff:
		// Linear backoff: 1s, 2s, 3s, 4s, ...
		return baseDelay * time.Duration(attempt+1)
	default:
		return baseDelay
	}
}

// ListenableStateGraph extends StateGraph with listener capabilities
type ListenableStateGraph struct {
	*StateGraph
	eventEmitter *EventEmitter
}

// NewListenableStateGraph creates a state graph with listener support
func NewListenableStateGraph() *ListenableStateGraph {
	return &ListenableStateGraph{
		StateGraph:   NewStateGraph(),
		eventEmitter: NewEventEmitter(),
	}
}

// AddListener adds an event listener to the graph
func (g *ListenableStateGraph) AddListener(listener EventListener) {
	g.eventEmitter.AddListener(listener)
}

// EventEmitter handles emitting events to listeners (from listeners.go integration)
type EventEmitter struct {
	listeners []EventListener
}

// EventListener defines the interface for event listeners (matching listeners.go)
type EventListener interface {
	OnEvent(ctx context.Context, event Event) error
}

// Event represents an event (matching listeners.go pattern)
type Event struct {
	Type      string                 `json:"type"`
	NodeName  string                 `json:"node_name,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Error     error                  `json:"error,omitempty"`
	State     interface{}            `json:"state,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		listeners: make([]EventListener, 0),
	}
}

// AddListener adds an event listener
func (e *EventEmitter) AddListener(listener EventListener) {
	e.listeners = append(e.listeners, listener)
}

// EmitEvent emits an event to all listeners
func (e *EventEmitter) EmitEvent(ctx context.Context, event Event) error {
	for _, listener := range e.listeners {
		if err := listener.OnEvent(ctx, event); err != nil {
			return fmt.Errorf("event emission error: %w", err)
		}
	}
	return nil
}