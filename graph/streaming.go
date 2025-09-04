package graph

import (
	"context"
	"sync"
	"time"
)

// StreamConfig configures streaming behavior
type StreamConfig struct {
	// BufferSize is the size of the event channel buffer
	BufferSize int

	// EnableBackpressure determines if backpressure handling is enabled
	EnableBackpressure bool

	// MaxDroppedEvents is the maximum number of events to drop before logging
	MaxDroppedEvents int
}

// DefaultStreamConfig returns the default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		BufferSize:         1000,
		EnableBackpressure: true,
		MaxDroppedEvents:   100,
	}
}

// StreamResult contains the channels returned by streaming execution
type StreamResult struct {
	// Events channel receives StreamEvent objects in real-time
	Events <-chan StreamEvent

	// Result channel receives the final result when execution completes
	Result <-chan interface{}

	// Errors channel receives any errors that occur during execution
	Errors <-chan error

	// Done channel is closed when streaming is complete
	Done <-chan struct{}

	// Cancel function can be called to stop streaming
	Cancel context.CancelFunc
}

// StreamingListener implements NodeListener for streaming events
type StreamingListener struct {
	eventChan chan<- StreamEvent
	config    StreamConfig
	mutex     sync.RWMutex

	droppedEvents int
	closed        bool
}

// NewStreamingListener creates a new streaming listener
func NewStreamingListener(eventChan chan<- StreamEvent, config StreamConfig) *StreamingListener {
	return &StreamingListener{
		eventChan: eventChan,
		config:    config,
	}
}

// OnNodeEvent implements the NodeListener interface
func (sl *StreamingListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, state interface{}, err error) {
	// Check if listener is closed
	sl.mutex.RLock()
	if sl.closed {
		sl.mutex.RUnlock()
		return
	}
	sl.mutex.RUnlock()

	streamEvent := StreamEvent{
		Timestamp: time.Now(),
		NodeName:  nodeName,
		Event:     event,
		State:     state,
		Error:     err,
		Metadata:  make(map[string]interface{}),
	}

	// Try to send event without blocking
	select {
	case sl.eventChan <- streamEvent:
		// Event sent successfully
	default:
		// Channel is full
		if sl.config.EnableBackpressure {
			sl.handleBackpressure()
		}
		// Drop the event if backpressure handling is disabled or channel is still full
	}
}

// Close marks the listener as closed to prevent sending to closed channels
func (sl *StreamingListener) Close() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.closed = true
}

// handleBackpressure manages channel backpressure
func (sl *StreamingListener) handleBackpressure() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	sl.droppedEvents++

	// Could implement more sophisticated backpressure strategies here
	// For now, we just track dropped events
}

// GetDroppedEventsCount returns the number of dropped events
func (sl *StreamingListener) GetDroppedEventsCount() int {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	return sl.droppedEvents
}

// StreamingRunnable wraps a ListenableRunnable with streaming capabilities
type StreamingRunnable struct {
	runnable *ListenableRunnable
	config   StreamConfig
}

// NewStreamingRunnable creates a new streaming runnable
func NewStreamingRunnable(runnable *ListenableRunnable, config StreamConfig) *StreamingRunnable {
	return &StreamingRunnable{
		runnable: runnable,
		config:   config,
	}
}

// NewStreamingRunnableWithDefaults creates a streaming runnable with default config
func NewStreamingRunnableWithDefaults(runnable *ListenableRunnable) *StreamingRunnable {
	return NewStreamingRunnable(runnable, DefaultStreamConfig())
}

// Stream executes the graph with real-time event streaming
func (sr *StreamingRunnable) Stream(ctx context.Context, initialState interface{}) *StreamResult {
	// Create channels
	eventChan := make(chan StreamEvent, sr.config.BufferSize)
	resultChan := make(chan interface{}, 1)
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Create cancellable context
	streamCtx, cancel := context.WithCancel(ctx)

	// Create streaming listener
	streamingListener := NewStreamingListener(eventChan, sr.config)

	// Add the streaming listener to all nodes
	for _, node := range sr.runnable.listenableNodes {
		node.AddListener(streamingListener)
	}

	// Execute in goroutine
	go func() {
		defer func() {
			// First, close the streaming listener to prevent new events
			streamingListener.Close()

			// Clean up: remove streaming listener from all nodes
			for _, node := range sr.runnable.listenableNodes {
				node.RemoveListener(streamingListener)
			}

			// Give a small delay for any in-flight listener calls to complete
			time.Sleep(10 * time.Millisecond)

			// Now safe to close channels
			close(eventChan)
			close(resultChan)
			close(errorChan)
			close(doneChan)
		}()

		// Execute the runnable
		result, err := sr.runnable.Invoke(streamCtx, initialState)

		// Send result or error
		if err != nil {
			select {
			case errorChan <- err:
			case <-streamCtx.Done():
			}
		} else {
			select {
			case resultChan <- result:
			case <-streamCtx.Done():
			}
		}
	}()

	return &StreamResult{
		Events: eventChan,
		Result: resultChan,
		Errors: errorChan,
		Done:   doneChan,
		Cancel: cancel,
	}
}

// StreamingMessageGraph extends ListenableMessageGraph with streaming capabilities
type StreamingMessageGraph struct {
	*ListenableMessageGraph
	config StreamConfig
}

// NewStreamingMessageGraph creates a new streaming message graph
func NewStreamingMessageGraph() *StreamingMessageGraph {
	return &StreamingMessageGraph{
		ListenableMessageGraph: NewListenableMessageGraph(),
		config:                 DefaultStreamConfig(),
	}
}

// NewStreamingMessageGraphWithConfig creates a streaming graph with custom config
func NewStreamingMessageGraphWithConfig(config StreamConfig) *StreamingMessageGraph {
	return &StreamingMessageGraph{
		ListenableMessageGraph: NewListenableMessageGraph(),
		config:                 config,
	}
}

// CompileStreaming compiles the graph into a streaming runnable
func (g *StreamingMessageGraph) CompileStreaming() (*StreamingRunnable, error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewStreamingRunnable(listenableRunnable, g.config), nil
}

// SetStreamConfig updates the streaming configuration
func (g *StreamingMessageGraph) SetStreamConfig(config StreamConfig) {
	g.config = config
}

// GetStreamConfig returns the current streaming configuration
func (g *StreamingMessageGraph) GetStreamConfig() StreamConfig {
	return g.config
}

// StreamingExecutor provides a high-level interface for streaming execution
type StreamingExecutor struct {
	runnable *StreamingRunnable
}

// NewStreamingExecutor creates a new streaming executor
func NewStreamingExecutor(runnable *StreamingRunnable) *StreamingExecutor {
	return &StreamingExecutor{
		runnable: runnable,
	}
}

// ExecuteWithCallback executes the graph and calls the callback for each event
//
//nolint:cyclop // Complex streaming logic requires multiple conditional paths
func (se *StreamingExecutor) ExecuteWithCallback(
	ctx context.Context,
	initialState interface{},
	eventCallback func(event StreamEvent),
	resultCallback func(result interface{}, err error),
) error {

	streamResult := se.runnable.Stream(ctx, initialState)
	defer streamResult.Cancel()

	var finalResult interface{}
	var finalError error
	resultReceived := false

	for {
		select {
		case event, ok := <-streamResult.Events:
			if !ok {
				// Events channel closed
				if resultReceived && resultCallback != nil {
					resultCallback(finalResult, finalError)
				}
				return finalError
			}
			if eventCallback != nil {
				eventCallback(event)
			}

		case result := <-streamResult.Result:
			finalResult = result
			resultReceived = true
			// Don't return immediately, wait for events channel to close

		case err := <-streamResult.Errors:
			finalError = err
			resultReceived = true
			// Don't return immediately, wait for events channel to close

		case <-streamResult.Done:
			if resultReceived && resultCallback != nil {
				resultCallback(finalResult, finalError)
			}
			return finalError

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ExecuteAsync executes the graph asynchronously and returns immediately
func (se *StreamingExecutor) ExecuteAsync(ctx context.Context, initialState interface{}) *StreamResult {
	return se.runnable.Stream(ctx, initialState)
}

// GetGraph returns a Exporter for the streaming runnable
func (sr *StreamingRunnable) GetGraph() *Exporter {
	return sr.runnable.GetGraph()
}
