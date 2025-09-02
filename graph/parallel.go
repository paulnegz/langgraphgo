package graph

import (
	"context"
	"fmt"
	"sync"
)

// ParallelNode represents a set of nodes that can execute in parallel
type ParallelNode struct {
	nodes []Node
	name  string
}

// NewParallelNode creates a new parallel node
func NewParallelNode(name string, nodes ...Node) *ParallelNode {
	return &ParallelNode{
		name:  name,
		nodes: nodes,
	}
}

// Execute runs all nodes in parallel and collects results
func (pn *ParallelNode) Execute(ctx context.Context, state interface{}) (interface{}, error) {
	// Create channels for results and errors
	type result struct {
		index int
		value interface{}
		err   error
	}

	results := make(chan result, len(pn.nodes))
	var wg sync.WaitGroup

	// Execute all nodes in parallel
	for i, node := range pn.nodes {
		wg.Add(1)
		go func(idx int, n Node) {
			defer wg.Done()

			// Execute with panic recovery
			defer func() {
				if r := recover(); r != nil {
					results <- result{
						index: idx,
						err:   fmt.Errorf("panic in parallel node %s[%d]: %v", pn.name, idx, r),
					}
				}
			}()

			value, err := n.Function(ctx, state)
			results <- result{
				index: idx,
				value: value,
				err:   err,
			}
		}(i, node)
	}

	// Wait for all nodes to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	outputs := make([]interface{}, len(pn.nodes))
	var firstError error

	for res := range results {
		if res.err != nil && firstError == nil {
			firstError = res.err
		}
		outputs[res.index] = res.value
	}

	if firstError != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", firstError)
	}

	// Return collected results
	return outputs, nil
}

// AddParallelNodes adds a set of nodes that execute in parallel
func (g *MessageGraph) AddParallelNodes(groupName string, nodes map[string]func(context.Context, interface{}) (interface{}, error)) {
	// Create parallel node group
	parallelNodes := make([]Node, 0, len(nodes))
	for name, fn := range nodes {
		parallelNodes = append(parallelNodes, Node{
			Name:     name,
			Function: fn,
		})
	}

	// Add as a single parallel node
	parallelNode := NewParallelNode(groupName, parallelNodes...)
	g.AddNode(groupName, parallelNode.Execute)
}

// MapReduceNode executes nodes in parallel and reduces results
type MapReduceNode struct {
	name     string
	mapNodes []Node
	reducer  func([]interface{}) (interface{}, error)
}

// NewMapReduceNode creates a new map-reduce node
func NewMapReduceNode(name string, reducer func([]interface{}) (interface{}, error), mapNodes ...Node) *MapReduceNode {
	return &MapReduceNode{
		name:     name,
		mapNodes: mapNodes,
		reducer:  reducer,
	}
}

// Execute runs map nodes in parallel and reduces results
func (mr *MapReduceNode) Execute(ctx context.Context, state interface{}) (interface{}, error) {
	// Execute map phase in parallel
	pn := NewParallelNode(mr.name+"_map", mr.mapNodes...)
	results, err := pn.Execute(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("map phase failed: %w", err)
	}

	// Execute reduce phase
	if mr.reducer != nil {
		return mr.reducer(results.([]interface{}))
	}

	return results, nil
}

// AddMapReduceNode adds a map-reduce pattern node
func (g *MessageGraph) AddMapReduceNode(
	name string,
	mapFunctions map[string]func(context.Context, interface{}) (interface{}, error),
	reducer func([]interface{}) (interface{}, error),
) {
	// Create map nodes
	mapNodes := make([]Node, 0, len(mapFunctions))
	for nodeName, fn := range mapFunctions {
		mapNodes = append(mapNodes, Node{
			Name:     nodeName,
			Function: fn,
		})
	}

	// Create and add map-reduce node
	mrNode := NewMapReduceNode(name, reducer, mapNodes...)
	g.AddNode(name, mrNode.Execute)
}

// FanOutFanIn creates a fan-out/fan-in pattern
func (g *MessageGraph) FanOutFanIn(
	source string,
	_ []string, // workers parameter kept for API compatibility
	collector string,
	workerFuncs map[string]func(context.Context, interface{}) (interface{}, error),
	collectFunc func([]interface{}) (interface{}, error),
) {
	// Add parallel worker nodes
	g.AddParallelNodes(source+"_workers", workerFuncs)

	// Add collector node
	g.AddNode(collector, func(_ context.Context, state interface{}) (interface{}, error) {
		// State should be array of results from parallel workers
		if results, ok := state.([]interface{}); ok {
			return collectFunc(results)
		}
		return nil, fmt.Errorf("invalid state for collector: expected []interface{}")
	})

	// Connect source to workers and workers to collector
	g.AddEdge(source, source+"_workers")
	g.AddEdge(source+"_workers", collector)
}
