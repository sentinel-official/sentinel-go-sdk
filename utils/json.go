package utils

import (
	"encoding/json"
)

// MustMarshalJSON takes an interface{} as input and returns its JSON encoding.
// If the marshaling fails, it panics instead of returning an error.
func MustMarshalJSON(v interface{}) []byte {
	// Marshal the input interface into JSON format.
	buf, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	// Return the JSON encoded bytes.
	return buf
}
