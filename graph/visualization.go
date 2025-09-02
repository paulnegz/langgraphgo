package graph

import (
	"fmt"
	"sort"
	"strings"
)

// Exporter provides methods to export graphs in different formats
type Exporter struct {
	graph *MessageGraph
}

// NewExporter creates a new graph exporter for the given graph
func NewExporter(graph *MessageGraph) *Exporter {
	return &Exporter{graph: graph}
}

// DrawMermaid generates a Mermaid diagram representation of the graph
func (ge *Exporter) DrawMermaid() string {
	var sb strings.Builder

	// Start Mermaid flowchart
	sb.WriteString("flowchart TD\n")

	// Add entry point styling
	if ge.graph.entryPoint != "" {
		sb.WriteString(fmt.Sprintf("    %s[[\"%s\"]]\n", ge.graph.entryPoint, ge.graph.entryPoint))
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", "START", ge.graph.entryPoint))
		sb.WriteString("    START([\"START\"])\n")
		sb.WriteString("    style START fill:#90EE90\n")
	}

	// Get sorted node names for consistent output
	nodeNames := make([]string, 0, len(ge.graph.nodes))
	for name := range ge.graph.nodes {
		if name != ge.graph.entryPoint && name != END {
			nodeNames = append(nodeNames, name)
		}
	}
	sort.Strings(nodeNames)

	// Add regular nodes
	for _, name := range nodeNames {
		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", name, name))
	}

	// Add END node if referenced
	hasEnd := false
	for _, edge := range ge.graph.edges {
		if edge.To == END {
			hasEnd = true
			break
		}
	}

	if hasEnd {
		sb.WriteString("    END([\"END\"])\n")
		sb.WriteString("    style END fill:#FFB6C1\n")
	}

	// Add edges
	for _, edge := range ge.graph.edges {
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", edge.From, edge.To))
	}

	// Style entry point
	if ge.graph.entryPoint != "" {
		sb.WriteString(fmt.Sprintf("    style %s fill:#87CEEB\n", ge.graph.entryPoint))
	}

	return sb.String()
}

// DrawDOT generates a DOT (Graphviz) representation of the graph
func (ge *Exporter) DrawDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph G {\n")
	sb.WriteString("    rankdir=TD;\n")
	sb.WriteString("    node [shape=box];\n")

	// Add START node if there's an entry point
	if ge.graph.entryPoint != "" {
		sb.WriteString("    START [label=\"START\", shape=ellipse, style=filled, fillcolor=lightgreen];\n")
		sb.WriteString(fmt.Sprintf("    START -> %s;\n", ge.graph.entryPoint))
	}

	// Add entry point styling
	if ge.graph.entryPoint != "" {
		sb.WriteString(fmt.Sprintf("    %s [style=filled, fillcolor=lightblue];\n", ge.graph.entryPoint))
	}

	// Add END node styling if referenced
	hasEnd := false
	for _, edge := range ge.graph.edges {
		if edge.To == END {
			hasEnd = true
			break
		}
	}

	if hasEnd {
		sb.WriteString("    END [label=\"END\", shape=ellipse, style=filled, fillcolor=lightpink];\n")
	}

	// Add edges
	for _, edge := range ge.graph.edges {
		sb.WriteString(fmt.Sprintf("    %s -> %s;\n", edge.From, edge.To))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// DrawASCII generates an ASCII tree representation of the graph
func (ge *Exporter) DrawASCII() string {
	if ge.graph.entryPoint == "" {
		return "No entry point set\n"
	}

	var sb strings.Builder
	visited := make(map[string]bool)

	sb.WriteString("Graph Execution Flow:\n")
	sb.WriteString("├── START\n")

	ge.drawASCIINode(ge.graph.entryPoint, "│   ", true, visited, &sb)

	return sb.String()
}

// drawASCIINode recursively draws ASCII representation of nodes
func (ge *Exporter) drawASCIINode(nodeName string, prefix string, isLast bool, visited map[string]bool, sb *strings.Builder) {
	if visited[nodeName] {
		// Handle cycles
		connector := "├──"
		if isLast {
			connector = "└──"
		}
		sb.WriteString(fmt.Sprintf("%s%s %s (cycle)\n", prefix, connector, nodeName))
		return
	}

	visited[nodeName] = true

	connector := "├──"
	nextPrefix := prefix + "│   "
	if isLast {
		connector = "└──"
		nextPrefix = prefix + "    "
	}

	sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, connector, nodeName))

	if nodeName == END {
		return
	}

	// Find outgoing edges
	outgoingEdges := make([]string, 0)
	for _, edge := range ge.graph.edges {
		if edge.From == nodeName {
			outgoingEdges = append(outgoingEdges, edge.To)
		}
	}

	// Sort for consistent output
	sort.Strings(outgoingEdges)

	// Draw child nodes
	for i, target := range outgoingEdges {
		isLastChild := i == len(outgoingEdges)-1
		ge.drawASCIINode(target, nextPrefix, isLastChild, visited, sb)
	}
}

// GetGraph returns a Exporter for the compiled graph's visualization
func (r *Runnable) GetGraph() *Exporter {
	return NewExporter(r.graph)
}
