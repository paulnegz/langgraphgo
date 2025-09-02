package graph

import (
	"context"
	"time"
)

// TraceEvent represents different types of events in graph execution
type TraceEvent string

const (
	// TraceEventGraphStart indicates the start of graph execution
	TraceEventGraphStart TraceEvent = "graph_start"

	// TraceEventGraphEnd indicates the end of graph execution
	TraceEventGraphEnd TraceEvent = "graph_end"

	// TraceEventNodeStart indicates the start of node execution
	TraceEventNodeStart TraceEvent = "node_start"

	// TraceEventNodeEnd indicates the end of node execution
	TraceEventNodeEnd TraceEvent = "node_end"

	// TraceEventNodeError indicates an error occurred in node execution
	TraceEventNodeError TraceEvent = "node_error"

	// TraceEventEdgeTraversal indicates traversal from one node to another
	TraceEventEdgeTraversal TraceEvent = "edge_traversal"
)

// TraceSpan represents a span of execution with timing and metadata
type TraceSpan struct {
	// ID is a unique identifier for this span
	ID string

	// ParentID is the ID of the parent span (empty for root spans)
	ParentID string

	// Event indicates the type of event this span represents
	Event TraceEvent

	// NodeName is the name of the node being executed (if applicable)
	NodeName string

	// FromNode is the source node for edge traversals
	FromNode string

	// ToNode is the destination node for edge traversals
	ToNode string

	// StartTime is when this span began
	StartTime time.Time

	// EndTime is when this span completed (zero for ongoing spans)
	EndTime time.Time

	// Duration is the total time taken (calculated when span ends)
	Duration time.Duration

	// State is a snapshot of the state at this point (optional)
	State interface{}

	// Error contains any error that occurred during execution
	Error error

	// Metadata contains additional key-value pairs for observability
	Metadata map[string]interface{}
}

// TraceHook defines the interface for trace event handlers
type TraceHook interface {
	// OnEvent is called when a trace event occurs
	OnEvent(ctx context.Context, span *TraceSpan)
}

// TraceHookFunc is a function adapter for TraceHook
type TraceHookFunc func(ctx context.Context, span *TraceSpan)

// OnEvent implements the TraceHook interface
func (f TraceHookFunc) OnEvent(ctx context.Context, span *TraceSpan) {
	f(ctx, span)
}

// Tracer manages trace collection and hooks
type Tracer struct {
	hooks []TraceHook
	spans map[string]*TraceSpan
}

// NewTracer creates a new tracer instance
func NewTracer() *Tracer {
	return &Tracer{
		hooks: make([]TraceHook, 0),
		spans: make(map[string]*TraceSpan),
	}
}

// AddHook registers a new trace hook
func (t *Tracer) AddHook(hook TraceHook) {
	t.hooks = append(t.hooks, hook)
}

// StartSpan creates a new trace span
func (t *Tracer) StartSpan(ctx context.Context, event TraceEvent, nodeName string) *TraceSpan {
	span := &TraceSpan{
		ID:        generateSpanID(),
		Event:     event,
		NodeName:  nodeName,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Extract parent ID from context if available
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		span.ParentID = parentSpan.ID
	}

	t.spans[span.ID] = span

	// Notify hooks
	for _, hook := range t.hooks {
		hook.OnEvent(ctx, span)
	}

	return span
}

// EndSpan completes a trace span
func (t *Tracer) EndSpan(ctx context.Context, span *TraceSpan, state interface{}, err error) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.State = state
	span.Error = err

	// Update event type if there was an error
	if err != nil && span.Event == TraceEventNodeStart {
		span.Event = TraceEventNodeError
	} else if span.Event == TraceEventNodeStart {
		span.Event = TraceEventNodeEnd
	} else if span.Event == TraceEventGraphStart {
		span.Event = TraceEventGraphEnd
	}

	// Notify hooks
	for _, hook := range t.hooks {
		hook.OnEvent(ctx, span)
	}
}

// TraceEdgeTraversal records an edge traversal event
func (t *Tracer) TraceEdgeTraversal(ctx context.Context, fromNode, toNode string) {
	span := &TraceSpan{
		ID:        generateSpanID(),
		Event:     TraceEventEdgeTraversal,
		FromNode:  fromNode,
		ToNode:    toNode,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  0,
		Metadata:  make(map[string]interface{}),
	}

	// Extract parent ID from context if available
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		span.ParentID = parentSpan.ID
	}

	t.spans[span.ID] = span

	// Notify hooks
	for _, hook := range t.hooks {
		hook.OnEvent(ctx, span)
	}
}

// GetSpans returns all collected spans
func (t *Tracer) GetSpans() map[string]*TraceSpan {
	return t.spans
}

// Clear removes all collected spans
func (t *Tracer) Clear() {
	t.spans = make(map[string]*TraceSpan)
}

// Context keys for span storage
type contextKey string

const spanContextKey contextKey = "langgraph_span"

// ContextWithSpan returns a new context with the span stored
func ContextWithSpan(ctx context.Context, span *TraceSpan) context.Context {
	return context.WithValue(ctx, spanContextKey, span)
}

// SpanFromContext extracts a span from context
func SpanFromContext(ctx context.Context) *TraceSpan {
	if span, ok := ctx.Value(spanContextKey).(*TraceSpan); ok {
		return span
	}
	return nil
}

// generateSpanID creates a unique span identifier
func generateSpanID() string {
	return time.Now().Format("20060102150405.000000")
}

// TracedRunnable wraps a Runnable with tracing capabilities
type TracedRunnable struct {
	*Runnable
	tracer *Tracer
}

// NewTracedRunnable creates a new traced runnable
func NewTracedRunnable(runnable *Runnable, tracer *Tracer) *TracedRunnable {
	return &TracedRunnable{
		Runnable: runnable,
		tracer:   tracer,
	}
}

// Invoke executes the graph with tracing enabled
func (tr *TracedRunnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	// Start graph execution span
	graphSpan := tr.tracer.StartSpan(ctx, TraceEventGraphStart, "")
	ctx = ContextWithSpan(ctx, graphSpan)

	state := initialState
	currentNode := tr.graph.entryPoint
	var finalError error

	for {
		if currentNode == END {
			break
		}

		node, ok := tr.graph.nodes[currentNode]
		if !ok {
			finalError = ErrNodeNotFound
			tr.tracer.EndSpan(ctx, graphSpan, state, finalError)
			return nil, finalError
		}

		// Start node execution span
		nodeSpan := tr.tracer.StartSpan(ctx, TraceEventNodeStart, currentNode)
		nodeCtx := ContextWithSpan(ctx, nodeSpan)

		var err error
		state, err = node.Function(nodeCtx, state)

		// End node execution span
		tr.tracer.EndSpan(nodeCtx, nodeSpan, state, err)

		if err != nil {
			finalError = err
			tr.tracer.EndSpan(ctx, graphSpan, state, finalError)
			return nil, finalError
		}

		// Find next node
		foundNext := false
		for _, edge := range tr.graph.edges {
			if edge.From == currentNode {
				tr.tracer.TraceEdgeTraversal(ctx, currentNode, edge.To)
				currentNode = edge.To
				foundNext = true
				break
			}
		}

		if !foundNext {
			finalError = ErrNoOutgoingEdge
			tr.tracer.EndSpan(ctx, graphSpan, state, finalError)
			return nil, finalError
		}
	}

	tr.tracer.EndSpan(ctx, graphSpan, state, nil)
	return state, nil
}

// GetTracer returns the tracer instance
func (tr *TracedRunnable) GetTracer() *Tracer {
	return tr.tracer
}
