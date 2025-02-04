package utils

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// YAMLFromJSON converts an input (JSON-encoded bytes, string, or struct) to YAML format.
func YAMLFromJSON(i interface{}) ([]byte, error) {
	var in interface{}

	switch v := i.(type) {
	case []byte:
		// If input is a JSON byte slice, unmarshal into an interface
		if err := json.Unmarshal(v, &in); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json: %w", err)
		}
	case string:
		// If input is a JSON string, convert it to bytes and unmarshal
		if err := json.Unmarshal([]byte(v), &in); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json: %w", err)
		}
	default:
		// Marshal struct or other types to JSON
		buf, err := json.Marshal(i)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal json: %w", err)
		}
		// Unmarshal the JSON into an interface to maintain its structure
		if err := json.Unmarshal(buf, &in); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json: %w", err)
		}
	}

	// Convert the intermediate structure to YAML format and return it
	buf, err := yaml.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal yaml: %w", err)
	}

	return buf, nil
}
