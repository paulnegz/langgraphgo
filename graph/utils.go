package graph

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

// generateRunID generates a unique run ID for callbacks
func generateRunID() string {
	return uuid.New().String()
}

// convertStateToMap converts a state to a map for callbacks
func convertStateToMap(state interface{}) map[string]interface{} {
	// Try to convert to map directly
	if m, ok := state.(map[string]interface{}); ok {
		return m
	}
	
	// Try to marshal/unmarshal through JSON
	data, err := json.Marshal(state)
	if err != nil {
		return map[string]interface{}{
			"state": fmt.Sprintf("%v", state),
		}
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]interface{}{
			"state": string(data),
		}
	}
	
	return result
}

// convertStateToString converts a state to a string for callbacks
func convertStateToString(state interface{}) string {
	// Try to marshal to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Sprintf("%v", state)
	}
	return string(data)
}