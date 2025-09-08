package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// MergeJSON merges multiple JSON readers into one
func MergeJSON(readers []io.Reader) (io.Reader, error) {
	if len(readers) == 0 {
		return bytes.NewReader([]byte("{}")), nil
	}
	
	// Start with empty map
	merged := make(map[string]interface{})
	
	// Read and merge each JSON
	for _, r := range readers {
		var data map[string]interface{}
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode JSON: %w", err)
		}
		
		// Merge data into result
		for k, v := range data {
			merged[k] = mergeValue(merged[k], v)
		}
	}
	
	// Encode merged result
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(merged); err != nil {
		return nil, fmt.Errorf("failed to encode merged JSON: %w", err)
	}
	
	return &buf, nil
}

// mergeValue merges two values, preferring the second value
// If both are maps, they are merged recursively
func mergeValue(v1, v2 interface{}) interface{} {
	// If v1 is nil, return v2
	if v1 == nil {
		return v2
	}
	
	// If both are maps, merge them
	m1, ok1 := v1.(map[string]interface{})
	m2, ok2 := v2.(map[string]interface{})
	if ok1 && ok2 {
		result := make(map[string]interface{})
		// Copy from m1
		for k, v := range m1 {
			result[k] = v
		}
		// Merge from m2
		for k, v := range m2 {
			if existing, exists := result[k]; exists {
				result[k] = mergeValue(existing, v)
			} else {
				result[k] = v
			}
		}
		return result
	}
	
	// Otherwise, v2 overwrites v1
	return v2
}