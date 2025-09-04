package graph

import (
	"context"
	"fmt"
)

// Subgraph represents a nested graph that can be used as a node
type Subgraph struct {
	name     string
	graph    *MessageGraph
	runnable *Runnable
}

// NewSubgraph creates a new subgraph
func NewSubgraph(name string, graph *MessageGraph) (*Subgraph, error) {
	runnable, err := graph.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile subgraph %s: %w", name, err)
	}

	return &Subgraph{
		name:     name,
		graph:    graph,
		runnable: runnable,
	}, nil
}

// Execute runs the subgraph as a node
func (s *Subgraph) Execute(ctx context.Context, state interface{}) (interface{}, error) {
	result, err := s.runnable.Invoke(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("subgraph %s execution failed: %w", s.name, err)
	}
	return result, nil
}

// AddSubgraph adds a subgraph as a node in the parent graph
func (g *MessageGraph) AddSubgraph(name string, subgraph *MessageGraph) error {
	sg, err := NewSubgraph(name, subgraph)
	if err != nil {
		return err
	}

	g.AddNode(name, sg.Execute)
	return nil
}

// CreateSubgraph creates and adds a subgraph using a builder function
func (g *MessageGraph) CreateSubgraph(name string, builder func(*MessageGraph)) error {
	subgraph := NewMessageGraph()
	builder(subgraph)
	return g.AddSubgraph(name, subgraph)
}

// CompositeGraph allows composing multiple graphs together
type CompositeGraph struct {
	graphs map[string]*MessageGraph
	main   *MessageGraph
}

// NewCompositeGraph creates a new composite graph
func NewCompositeGraph() *CompositeGraph {
	return &CompositeGraph{
		graphs: make(map[string]*MessageGraph),
		main:   NewMessageGraph(),
	}
}

// AddGraph adds a named graph to the composite
func (cg *CompositeGraph) AddGraph(name string, graph *MessageGraph) {
	cg.graphs[name] = graph
}

// Connect connects two graphs with a transformation function
func (cg *CompositeGraph) Connect(
	fromGraph string,
	fromNode string,
	toGraph string,
	toNode string,
	transform func(interface{}) interface{},
) error {
	// Create a bridge node that transforms state between graphs
	bridgeName := fmt.Sprintf("%s_%s_to_%s_%s", fromGraph, fromNode, toGraph, toNode)

	cg.main.AddNode(bridgeName, func(_ context.Context, state interface{}) (interface{}, error) {
		if transform != nil {
			return transform(state), nil
		}
		return state, nil
	})

	return nil
}

// Compile compiles the composite graph into a single runnable
func (cg *CompositeGraph) Compile() (*Runnable, error) {
	// Add all subgraphs to the main graph
	for name, graph := range cg.graphs {
		if err := cg.main.AddSubgraph(name, graph); err != nil {
			return nil, fmt.Errorf("failed to add subgraph %s: %w", name, err)
		}
	}

	return cg.main.Compile()
}

// RecursiveSubgraph allows a subgraph to call itself recursively
type RecursiveSubgraph struct {
	name      string
	graph     *MessageGraph
	maxDepth  int
	condition func(interface{}, int) bool // Should continue recursion?
}

// NewRecursiveSubgraph creates a new recursive subgraph
func NewRecursiveSubgraph(
	name string,
	maxDepth int,
	condition func(interface{}, int) bool,
) *RecursiveSubgraph {
	return &RecursiveSubgraph{
		name:      name,
		graph:     NewMessageGraph(),
		maxDepth:  maxDepth,
		condition: condition,
	}
}

// Execute runs the recursive subgraph
func (rs *RecursiveSubgraph) Execute(ctx context.Context, state interface{}) (interface{}, error) {
	return rs.executeRecursive(ctx, state, 0)
}

func (rs *RecursiveSubgraph) executeRecursive(ctx context.Context, state interface{}, depth int) (interface{}, error) {
	// Check max depth
	if depth >= rs.maxDepth {
		return state, nil
	}

	// Check condition
	if !rs.condition(state, depth) {
		return state, nil
	}

	// Compile and execute the graph
	runnable, err := rs.graph.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile recursive subgraph at depth %d: %w", depth, err)
	}

	result, err := runnable.Invoke(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("recursive execution failed at depth %d: %w", depth, err)
	}

	// Recurse with the result
	return rs.executeRecursive(ctx, result, depth+1)
}

// AddRecursiveSubgraph adds a recursive subgraph to the parent graph
func (g *MessageGraph) AddRecursiveSubgraph(
	name string,
	maxDepth int,
	condition func(interface{}, int) bool,
	builder func(*MessageGraph),
) {
	rs := NewRecursiveSubgraph(name, maxDepth, condition)
	builder(rs.graph)
	g.AddNode(name, rs.Execute)
}

// NestedConditionalSubgraph creates a subgraph with its own conditional routing
func (g *MessageGraph) AddNestedConditionalSubgraph(
	name string,
	router func(interface{}) string,
	subgraphs map[string]*MessageGraph,
) error {
	// Create a wrapper node that routes to different subgraphs
	g.AddNode(name, func(ctx context.Context, state interface{}) (interface{}, error) {
		// Determine which subgraph to use
		subgraphName := router(state)

		subgraph, exists := subgraphs[subgraphName]
		if !exists {
			return nil, fmt.Errorf("subgraph %s not found", subgraphName)
		}

		// Compile and execute the selected subgraph
		runnable, err := subgraph.Compile()
		if err != nil {
			return nil, fmt.Errorf("failed to compile subgraph %s: %w", subgraphName, err)
		}

		return runnable.Invoke(ctx, state)
	})

	return nil
}
