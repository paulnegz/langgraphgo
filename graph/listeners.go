package graph

import (
	"context"
	"sync"
	"time"
)

// NodeEvent represents different types of node events
type NodeEvent string

const (
	// NodeEventStart indicates a node has started execution
	NodeEventStart NodeEvent = "start"

	// NodeEventProgress indicates progress during node execution
	NodeEventProgress NodeEvent = "progress"

	// NodeEventComplete indicates a node has completed successfully
	NodeEventComplete NodeEvent = "complete"

	// NodeEventError indicates a node encountered an error
	NodeEventError NodeEvent = "error"
)

// NodeListener defines the interface for node event listeners
type NodeListener interface {
	// OnNodeEvent is called when a node event occurs
	OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state interface{}, err error)
}

// NodeListenerFunc is a function adapter for NodeListener
type NodeListenerFunc func(ctx context.Context, event NodeEvent, nodeName string, state interface{}, err error)

// OnNodeEvent implements the NodeListener interface
func (f NodeListenerFunc) OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state interface{}, err error) {
	f(ctx, event, nodeName, state, err)
}

// StreamEvent represents an event in the streaming execution
type StreamEvent struct {
	// Timestamp when the event occurred
	Timestamp time.Time

	// NodeName is the name of the node that generated the event
	NodeName string

	// Event is the type of event
	Event NodeEvent

	// State is the current state at the time of the event
	State interface{}

	// Error contains any error that occurred (if Event is NodeEventError)
	Error error

	// Metadata contains additional event-specific data
	Metadata map[string]interface{}

	// Duration is how long the node took (only for Complete events)
	Duration time.Duration
}

// ListenableNode extends Node with listener capabilities
type ListenableNode struct {
	Node
	listeners []NodeListener
	mutex     sync.RWMutex
}

// NewListenableNode creates a new listenable node from a regular node
func NewListenableNode(node Node) *ListenableNode {
	return &ListenableNode{
		Node:      node,
		listeners: make([]NodeListener, 0),
	}
}

// AddListener adds a listener to the node
func (ln *ListenableNode) AddListener(listener NodeListener) *ListenableNode {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	ln.listeners = append(ln.listeners, listener)
	return ln
}

// RemoveListener removes a listener from the node
func (ln *ListenableNode) RemoveListener(listener NodeListener) {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	for i, l := range ln.listeners {
		// For StreamingListener, we can compare the actual objects
		if l == listener {
			ln.listeners = append(ln.listeners[:i], ln.listeners[i+1:]...)
			break
		}
	}
}

// NotifyListeners notifies all listeners of an event
func (ln *ListenableNode) NotifyListeners(ctx context.Context, event NodeEvent, state interface{}, err error) {
	ln.mutex.RLock()
	listeners := make([]NodeListener, len(ln.listeners))
	copy(listeners, ln.listeners)
	ln.mutex.RUnlock()

	// Use WaitGroup to synchronize listener notifications
	var wg sync.WaitGroup

	// Notify listeners in separate goroutines to avoid blocking execution
	for _, listener := range listeners {
		wg.Add(1)
		go func(l NodeListener) {
			defer wg.Done()

			// Protect against panics in listeners
			defer func() {
				if r := recover(); r != nil {
					// Panic recovered, but not logged to avoid dependencies
					_ = r // Acknowledge the panic was caught
				}
			}()

			l.OnNodeEvent(ctx, event, ln.Name, state, err)
		}(listener)
	}

	// Wait for all listener notifications to complete
	wg.Wait()
}

// Execute runs the node function with listener notifications
func (ln *ListenableNode) Execute(ctx context.Context, state interface{}) (interface{}, error) {
	// Notify start
	ln.NotifyListeners(ctx, NodeEventStart, state, nil)

	// Execute the node function
	result, err := ln.Function(ctx, state)

	// Notify completion or error
	if err != nil {
		ln.NotifyListeners(ctx, NodeEventError, state, err)
	} else {
		ln.NotifyListeners(ctx, NodeEventComplete, result, nil)
	}

	return result, err
}

// GetListeners returns a copy of the current listeners
func (ln *ListenableNode) GetListeners() []NodeListener {
	ln.mutex.RLock()
	defer ln.mutex.RUnlock()

	listeners := make([]NodeListener, len(ln.listeners))
	copy(listeners, ln.listeners)
	return listeners
}

// ListenableMessageGraph extends MessageGraph with listener capabilities
type ListenableMessageGraph struct {
	*MessageGraph
	listenableNodes map[string]*ListenableNode
}

// NewListenableMessageGraph creates a new message graph with listener support
func NewListenableMessageGraph() *ListenableMessageGraph {
	return &ListenableMessageGraph{
		MessageGraph:    NewMessageGraph(),
		listenableNodes: make(map[string]*ListenableNode),
	}
}

// AddNode adds a node with listener capabilities
func (g *ListenableMessageGraph) AddNode(name string, fn func(ctx context.Context, state interface{}) (interface{}, error)) *ListenableNode {
	node := Node{
		Name:     name,
		Function: fn,
	}

	listenableNode := NewListenableNode(node)

	// Add to both the base graph and our listenable nodes map
	g.MessageGraph.AddNode(name, fn)
	g.listenableNodes[name] = listenableNode

	return listenableNode
}

// GetListenableNode returns the listenable node by name
func (g *ListenableMessageGraph) GetListenableNode(name string) *ListenableNode {
	return g.listenableNodes[name]
}

// AddGlobalListener adds a listener to all nodes in the graph
func (g *ListenableMessageGraph) AddGlobalListener(listener NodeListener) {
	for _, node := range g.listenableNodes {
		node.AddListener(listener)
	}
}

// RemoveGlobalListener removes a listener from all nodes in the graph
func (g *ListenableMessageGraph) RemoveGlobalListener(listener NodeListener) {
	for _, node := range g.listenableNodes {
		node.RemoveListener(listener)
	}
}

// ListenableRunnable wraps a Runnable with listener capabilities
type ListenableRunnable struct {
	graph           *ListenableMessageGraph
	listenableNodes map[string]*ListenableNode
}

// NewListenableRunnable creates a runnable with listener support
func (g *ListenableMessageGraph) CompileListenable() (*ListenableRunnable, error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &ListenableRunnable{
		graph:           g,
		listenableNodes: g.listenableNodes,
	}, nil
}

// Invoke executes the graph with listener notifications
func (lr *ListenableRunnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	state := initialState
	currentNode := lr.graph.entryPoint

	for {
		if currentNode == END {
			break
		}

		listenableNode, ok := lr.listenableNodes[currentNode]
		if !ok {
			return nil, ErrNodeNotFound
		}

		var err error
		state, err = listenableNode.Execute(ctx, state)
		if err != nil {
			return nil, err
		}

		// Find next node
		foundNext := false
		for _, edge := range lr.graph.edges {
			if edge.From == currentNode {
				currentNode = edge.To
				foundNext = true
				break
			}
		}

		if !foundNext {
			return nil, ErrNoOutgoingEdge
		}
	}

	return state, nil
}

// GetGraph returns a Exporter for visualization
func (lr *ListenableRunnable) GetGraph() *Exporter {
	return NewExporter(lr.graph.MessageGraph)
}
