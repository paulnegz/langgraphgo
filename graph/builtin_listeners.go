package graph

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// ProgressListener provides progress tracking with customizable output
type ProgressListener struct {
	writer      io.Writer
	nodeSteps   map[string]string
	mutex       sync.RWMutex
	showTiming  bool
	showDetails bool
	prefix      string
}

// NewProgressListener creates a new progress listener
func NewProgressListener() *ProgressListener {
	return &ProgressListener{
		writer:      os.Stdout,
		nodeSteps:   make(map[string]string),
		showTiming:  true,
		showDetails: false,
		prefix:      "🔄",
	}
}

// NewProgressListenerWithWriter creates a progress listener with custom writer
func NewProgressListenerWithWriter(writer io.Writer) *ProgressListener {
	return &ProgressListener{
		writer:      writer,
		nodeSteps:   make(map[string]string),
		showTiming:  true,
		showDetails: false,
		prefix:      "🔄",
	}
}

// WithTiming enables or disables timing information
func (pl *ProgressListener) WithTiming(enabled bool) *ProgressListener {
	pl.showTiming = enabled
	return pl
}

// WithDetails enables or disables detailed output
func (pl *ProgressListener) WithDetails(enabled bool) *ProgressListener {
	pl.showDetails = enabled
	return pl
}

// WithPrefix sets a custom prefix for progress messages
func (pl *ProgressListener) WithPrefix(prefix string) *ProgressListener {
	pl.prefix = prefix
	return pl
}

// SetNodeStep sets a custom message for a specific node
func (pl *ProgressListener) SetNodeStep(nodeName, step string) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	pl.nodeSteps[nodeName] = step
}

// OnNodeEvent implements the NodeListener interface
func (pl *ProgressListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, state interface{}, err error) {
	pl.mutex.RLock()
	customStep, hasCustom := pl.nodeSteps[nodeName]
	pl.mutex.RUnlock()

	var message string

	switch event {
	case NodeEventStart:
		if hasCustom {
			message = fmt.Sprintf("%s %s", pl.prefix, customStep)
		} else {
			message = fmt.Sprintf("%s Starting %s", pl.prefix, nodeName)
		}

	case NodeEventComplete:
		emoji := "✅"
		if hasCustom {
			message = fmt.Sprintf("%s %s completed", emoji, customStep)
		} else {
			message = fmt.Sprintf("%s %s completed", emoji, nodeName)
		}

	case NodeEventError:
		emoji := "❌"
		message = fmt.Sprintf("%s %s failed: %v", emoji, nodeName, err)

	case NodeEventProgress:
		if hasCustom {
			message = fmt.Sprintf("%s %s (in progress)", pl.prefix, customStep)
		} else {
			message = fmt.Sprintf("%s %s (in progress)", pl.prefix, nodeName)
		}
	}

	if pl.showTiming {
		timestamp := time.Now().Format("15:04:05")
		message = fmt.Sprintf("[%s] %s", timestamp, message)
	}

	if pl.showDetails && state != nil {
		message = fmt.Sprintf("%s | State: %v", message, state)
	}

	fmt.Fprintln(pl.writer, message)
}

// LoggingListener provides structured logging for node events
type LoggingListener struct {
	logger       *log.Logger
	logLevel     LogLevel
	includeState bool
}

// LogLevel defines logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// NewLoggingListener creates a new logging listener
func NewLoggingListener() *LoggingListener {
	return &LoggingListener{
		logger:       log.New(os.Stdout, "[GRAPH] ", log.LstdFlags),
		logLevel:     LogLevelInfo,
		includeState: false,
	}
}

// NewLoggingListenerWithLogger creates a logging listener with custom logger
func NewLoggingListenerWithLogger(logger *log.Logger) *LoggingListener {
	return &LoggingListener{
		logger:       logger,
		logLevel:     LogLevelInfo,
		includeState: false,
	}
}

// WithLogLevel sets the minimum log level
func (ll *LoggingListener) WithLogLevel(level LogLevel) *LoggingListener {
	ll.logLevel = level
	return ll
}

// WithState enables or disables state logging
func (ll *LoggingListener) WithState(enabled bool) *LoggingListener {
	ll.includeState = enabled
	return ll
}

// OnNodeEvent implements the NodeListener interface
func (ll *LoggingListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, state interface{}, err error) {
	var level LogLevel
	var prefix string

	switch event {
	case NodeEventStart:
		level = LogLevelInfo
		prefix = "START"
	case NodeEventComplete:
		level = LogLevelInfo
		prefix = "COMPLETE"
	case NodeEventProgress:
		level = LogLevelDebug
		prefix = "PROGRESS"
	case NodeEventError:
		level = LogLevelError
		prefix = "ERROR"
	}

	if level < ll.logLevel {
		return
	}

	message := fmt.Sprintf("%s %s", prefix, nodeName)

	if err != nil {
		message = fmt.Sprintf("%s: %v", message, err)
	}

	if ll.includeState && state != nil {
		message = fmt.Sprintf("%s | State: %v", message, state)
	}

	ll.logger.Println(message)
}

// MetricsListener collects performance and execution metrics
type MetricsListener struct {
	mutex           sync.RWMutex
	nodeExecutions  map[string]int
	nodeDurations   map[string][]time.Duration
	nodeErrors      map[string]int
	totalExecutions int
	startTimes      map[string]time.Time
}

// NewMetricsListener creates a new metrics listener
func NewMetricsListener() *MetricsListener {
	return &MetricsListener{
		nodeExecutions: make(map[string]int),
		nodeDurations:  make(map[string][]time.Duration),
		nodeErrors:     make(map[string]int),
		startTimes:     make(map[string]time.Time),
	}
}

// OnNodeEvent implements the NodeListener interface
func (ml *MetricsListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, _ interface{}, _ error) {
	ml.mutex.Lock()
	defer ml.mutex.Unlock()

	switch event {
	case NodeEventStart:
		ml.startTimes[nodeName] = time.Now()
		ml.totalExecutions++

	case NodeEventComplete:
		ml.nodeExecutions[nodeName]++
		if startTime, ok := ml.startTimes[nodeName]; ok {
			duration := time.Since(startTime)
			ml.nodeDurations[nodeName] = append(ml.nodeDurations[nodeName], duration)
			delete(ml.startTimes, nodeName)
		}

	case NodeEventError:
		ml.nodeErrors[nodeName]++
		if startTime, ok := ml.startTimes[nodeName]; ok {
			duration := time.Since(startTime)
			ml.nodeDurations[nodeName] = append(ml.nodeDurations[nodeName], duration)
			delete(ml.startTimes, nodeName)
		}
	case NodeEventProgress:
		// Progress events are tracked but don't affect timing metrics
	}
}

// GetNodeExecutions returns the number of executions for each node
func (ml *MetricsListener) GetNodeExecutions() map[string]int {
	ml.mutex.RLock()
	defer ml.mutex.RUnlock()

	result := make(map[string]int)
	for k, v := range ml.nodeExecutions {
		result[k] = v
	}
	return result
}

// GetNodeErrors returns the number of errors for each node
func (ml *MetricsListener) GetNodeErrors() map[string]int {
	ml.mutex.RLock()
	defer ml.mutex.RUnlock()

	result := make(map[string]int)
	for k, v := range ml.nodeErrors {
		result[k] = v
	}
	return result
}

// GetNodeAverageDuration returns the average duration for each node
func (ml *MetricsListener) GetNodeAverageDuration() map[string]time.Duration {
	ml.mutex.RLock()
	defer ml.mutex.RUnlock()

	result := make(map[string]time.Duration)
	for nodeName, durations := range ml.nodeDurations {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			result[nodeName] = total / time.Duration(len(durations))
		}
	}
	return result
}

// GetTotalExecutions returns the total number of node executions
func (ml *MetricsListener) GetTotalExecutions() int {
	ml.mutex.RLock()
	defer ml.mutex.RUnlock()
	return ml.totalExecutions
}

// PrintSummary prints a summary of collected metrics
func (ml *MetricsListener) PrintSummary(writer io.Writer) {
	ml.mutex.RLock()
	defer ml.mutex.RUnlock()

	fmt.Fprintln(writer, "\n=== Node Execution Metrics ===")
	fmt.Fprintf(writer, "Total Executions: %d\n", ml.totalExecutions)
	fmt.Fprintln(writer)

	fmt.Fprintln(writer, "Node Executions:")
	for nodeName, count := range ml.nodeExecutions {
		fmt.Fprintf(writer, "  %s: %d\n", nodeName, count)
	}
	fmt.Fprintln(writer)

	fmt.Fprintln(writer, "Average Durations:")
	for nodeName, durations := range ml.nodeDurations {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))
			fmt.Fprintf(writer, "  %s: %v (from %d samples)\n", nodeName, avg, len(durations))
		}
	}

	if len(ml.nodeErrors) > 0 {
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "Errors:")
		for nodeName, count := range ml.nodeErrors {
			fmt.Fprintf(writer, "  %s: %d errors\n", nodeName, count)
		}
	}
}

// Reset clears all collected metrics
func (ml *MetricsListener) Reset() {
	ml.mutex.Lock()
	defer ml.mutex.Unlock()

	ml.nodeExecutions = make(map[string]int)
	ml.nodeDurations = make(map[string][]time.Duration)
	ml.nodeErrors = make(map[string]int)
	ml.startTimes = make(map[string]time.Time)
	ml.totalExecutions = 0
}

// ChatListener provides real-time chat-style updates
type ChatListener struct {
	writer       io.Writer
	nodeMessages map[string]string
	mutex        sync.RWMutex
	showTime     bool
}

// NewChatListener creates a new chat-style listener
func NewChatListener() *ChatListener {
	return &ChatListener{
		writer:       os.Stdout,
		nodeMessages: make(map[string]string),
		showTime:     true,
	}
}

// NewChatListenerWithWriter creates a chat listener with custom writer
func NewChatListenerWithWriter(writer io.Writer) *ChatListener {
	return &ChatListener{
		writer:       writer,
		nodeMessages: make(map[string]string),
		showTime:     true,
	}
}

// WithTime enables or disables timestamps
func (cl *ChatListener) WithTime(enabled bool) *ChatListener {
	cl.showTime = enabled
	return cl
}

// SetNodeMessage sets a custom message for a specific node
func (cl *ChatListener) SetNodeMessage(nodeName, message string) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	cl.nodeMessages[nodeName] = message
}

// OnNodeEvent implements the NodeListener interface
func (cl *ChatListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, _ interface{}, err error) {
	cl.mutex.RLock()
	customMessage, hasCustom := cl.nodeMessages[nodeName]
	cl.mutex.RUnlock()

	var message string

	switch event {
	case NodeEventStart:
		if hasCustom {
			message = customMessage
		} else {
			message = fmt.Sprintf("🤖 Starting %s...", nodeName)
		}

	case NodeEventComplete:
		if hasCustom {
			message = fmt.Sprintf("✅ %s completed", customMessage)
		} else {
			message = fmt.Sprintf("✅ %s finished", nodeName)
		}

	case NodeEventError:
		message = fmt.Sprintf("❌ Error in %s: %v", nodeName, err)

	case NodeEventProgress:
		if hasCustom {
			message = fmt.Sprintf("⏳ %s...", customMessage)
		} else {
			message = fmt.Sprintf("⏳ %s in progress...", nodeName)
		}
	}

	if cl.showTime {
		timestamp := time.Now().Format("15:04:05")
		fmt.Fprintf(cl.writer, "[%s] %s\n", timestamp, message)
	} else {
		fmt.Fprintf(cl.writer, "%s\n", message)
	}
}
