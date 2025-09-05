package graph

import (
	"context"
	"errors"
	"fmt"
)

// END is a special constant used to represent the end node in the graph.
const END = "END"

var (
	// ErrEntryPointNotSet is returned when the entry point of the graph is not set.
	ErrEntryPointNotSet = errors.New("entry point not set")

	// ErrNodeNotFound is returned when a node is not found in the graph.
	ErrNodeNotFound = errors.New("node not found")

	// ErrNoOutgoingEdge is returned when no outgoing edge is found for a node.
	ErrNoOutgoingEdge = errors.New("no outgoing edge found for node")
)

// Node represents a node in the message graph.
type Node struct {
	// Name is the unique identifier for the node.
	Name string

	// Function is the function associated with the node.
	// It takes a context and any state as input and returns the updated state and an error.
	Function func(ctx context.Context, state interface{}) (interface{}, error)
}

// Edge represents an edge in the message graph.
type Edge struct {
	// From is the name of the node from which the edge originates.
	From string

	// To is the name of the node to which the edge points.
	To string
}

// MessageGraph represents a message graph.
type MessageGraph struct {
	// nodes is a map of node names to their corresponding Node objects.
	nodes map[string]Node

	// edges is a slice of Edge objects representing the connections between nodes.
	edges []Edge

	// conditionalEdges contains a map between "From" node, while "To" node is derived based on the condition.
	conditionalEdges map[string]func(ctx context.Context, state interface{}) string

	// entryPoint is the name of the entry point node in the graph.
	entryPoint string
}

// NewMessageGraph creates a new instance of MessageGraph.
func NewMessageGraph() *MessageGraph {
	return &MessageGraph{
		nodes:            make(map[string]Node),
		conditionalEdges: make(map[string]func(ctx context.Context, state interface{}) string),
	}
}

// AddNode adds a new node to the message graph with the given name and function.
func (g *MessageGraph) AddNode(name string, fn func(ctx context.Context, state interface{}) (interface{}, error)) {
	g.nodes[name] = Node{
		Name:     name,
		Function: fn,
	}
}

// AddEdge adds a new edge to the message graph between the "from" and "to" nodes.
func (g *MessageGraph) AddEdge(from, to string) {
	g.edges = append(g.edges, Edge{
		From: from,
		To:   to,
	})
}

// AddConditionalEdge adds a conditional edge where the target node is determined at runtime.
// The condition function receives the current state and returns the name of the next node.
func (g *MessageGraph) AddConditionalEdge(from string, condition func(ctx context.Context, state interface{}) string) {
	g.conditionalEdges[from] = condition
}

// SetEntryPoint sets the entry point node name for the message graph.
func (g *MessageGraph) SetEntryPoint(name string) {
	g.entryPoint = name
}

// Runnable represents a compiled message graph that can be invoked.
type Runnable struct {
	// graph is the underlying MessageGraph object.
	graph *MessageGraph
	// tracer is the optional tracer for observability
	tracer *Tracer
}

// Compile compiles the message graph and returns a Runnable instance.
// It returns an error if the entry point is not set.
func (g *MessageGraph) Compile() (*Runnable, error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &Runnable{
		graph:  g,
		tracer: nil, // Initialize with no tracer
	}, nil
}

// SetTracer sets a tracer for observability
func (r *Runnable) SetTracer(tracer *Tracer) {
	r.tracer = tracer
}

// WithTracer returns a new Runnable with the given tracer
func (r *Runnable) WithTracer(tracer *Tracer) *Runnable {
	return &Runnable{
		graph:  r.graph,
		tracer: tracer,
	}
}

// Invoke executes the compiled message graph with the given input state.
// It returns the resulting state and an error if any occurs during the execution.
func (r *Runnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	return r.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the compiled message graph with the given input state and config.
// It returns the resulting state and an error if any occurs during the execution.
func (r *Runnable) InvokeWithConfig(ctx context.Context, initialState interface{}, config *Config) (interface{}, error) {
	state := initialState
	currentNode := r.graph.entryPoint

	// Generate run ID for callbacks
	runID := generateRunID()

	// Notify callbacks of graph start
	if config != nil && len(config.Callbacks) > 0 {
		serialized := map[string]interface{}{
			"name": "graph",
			"type": "chain",
		}
		inputs := convertStateToMap(initialState)

		for _, cb := range config.Callbacks {
			cb.OnChainStart(ctx, serialized, inputs, runID, nil, config.Tags, config.Metadata)
		}
	}

	// Start graph tracing if tracer is set
	var graphSpan *TraceSpan
	if r.tracer != nil {
		graphSpan = r.tracer.StartSpan(ctx, TraceEventGraphStart, "graph")
		graphSpan.State = initialState
	}

	for {
		if currentNode == END {
			break
		}

		node, ok := r.graph.nodes[currentNode]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, currentNode)
		}

		// Start node tracing
		var nodeSpan *TraceSpan
		if r.tracer != nil {
			nodeSpan = r.tracer.StartSpan(ctx, TraceEventNodeStart, currentNode)
			nodeSpan.State = state
		}

		var err error
		state, err = node.Function(ctx, state)

		// End node tracing
		if r.tracer != nil && nodeSpan != nil {
			if err != nil {
				r.tracer.EndSpan(ctx, nodeSpan, state, err)
				// Also emit error event
				errorSpan := r.tracer.StartSpan(ctx, TraceEventNodeError, currentNode)
				errorSpan.Error = err
				errorSpan.State = state
				r.tracer.EndSpan(ctx, errorSpan, state, err)
			} else {
				r.tracer.EndSpan(ctx, nodeSpan, state, nil)
			}
		}

		if err != nil {
			// Notify callbacks of error
			if config != nil && len(config.Callbacks) > 0 {
				for _, cb := range config.Callbacks {
					cb.OnChainError(ctx, err, runID)
				}
			}
			return nil, fmt.Errorf("error in node %s: %w", currentNode, err)
		}

		// Notify callbacks of node execution (as tool)
		if config != nil && len(config.Callbacks) > 0 {
			nodeRunID := generateRunID()
			serialized := map[string]interface{}{
				"name": currentNode,
				"type": "tool",
			}
			for _, cb := range config.Callbacks {
				cb.OnToolStart(ctx, serialized, convertStateToString(state), nodeRunID, &runID, config.Tags, config.Metadata)
				cb.OnToolEnd(ctx, convertStateToString(state), nodeRunID)
			}
		}

		// Determine next node
		var nextNode string

		// First check for conditional edges
		nextNodeFn, hasConditional := r.graph.conditionalEdges[currentNode]
		if hasConditional {
			nextNode = nextNodeFn(ctx, state)
			if nextNode == "" {
				return nil, fmt.Errorf("conditional edge returned empty next node from %s", currentNode)
			}
		} else {
			// Then check regular edges
			foundNext := false
			for _, edge := range r.graph.edges {
				if edge.From == currentNode {
					nextNode = edge.To
					foundNext = true
					break
				}
			}

			if !foundNext {
				return nil, fmt.Errorf("%w: %s", ErrNoOutgoingEdge, currentNode)
			}
		}

		// Trace edge traversal
		if r.tracer != nil && nextNode != "" && nextNode != END {
			edgeSpan := r.tracer.StartSpan(ctx, TraceEventEdgeTraversal, fmt.Sprintf("%s->%s", currentNode, nextNode))
			edgeSpan.FromNode = currentNode
			edgeSpan.ToNode = nextNode
			r.tracer.EndSpan(ctx, edgeSpan, state, nil)
		}

		currentNode = nextNode
	}

	// End graph tracing
	if r.tracer != nil && graphSpan != nil {
		r.tracer.EndSpan(ctx, graphSpan, state, nil)
	}

	// Notify callbacks of graph end
	if config != nil && len(config.Callbacks) > 0 {
		outputs := convertStateToMap(state)
		for _, cb := range config.Callbacks {
			cb.OnChainEnd(ctx, outputs, runID)
		}
	}

	return state, nil
}
