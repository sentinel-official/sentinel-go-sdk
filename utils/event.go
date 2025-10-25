package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/gogoproto/proto"
)

// getEventName determines the event name by checking if name is a string or a proto.Message.
func getEventName(t interface{}) (string, error) {
	switch v := t.(type) {
	case string:
		return v, nil
	case proto.Message:
		return proto.MessageName(v), nil
	default:
		return "", fmt.Errorf("unsupported event type %T", t)
	}
}

// EventFromEvents searches through a list of events and returns the first event that matches the given name.
func EventFromEvents(items []types.Event, t interface{}) (*types.Event, error) {
	// Retrieve the event name
	name, err := getEventName(t)
	if err != nil {
		return nil, fmt.Errorf("getting event name for %T: %w", t, err)
	}

	for _, item := range items {
		if item.GetType() == name {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("event %T does not exist", t)
}

// AttributeValueFromEvent searches for an attribute within an event by its key name.
func AttributeValueFromEvent(item *types.Event, key string) (string, error) {
	for _, attribute := range item.GetAttributes() {
		if attribute.GetKey() == key {
			return attribute.GetValue(), nil
		}
	}

	return "", fmt.Errorf("attribute %q does not exist", key)
}

// AttributeValueFromEvents retrieves an attribute's value from a list of events.
func AttributeValueFromEvents(items []types.Event, t interface{}, key string) (string, error) {
	// Find the event with the given type
	event, err := EventFromEvents(items, t)
	if err != nil {
		return "", fmt.Errorf("getting event %T from events: %w", t, err)
	}

	// Retrieve the attribute value from the event
	value, err := AttributeValueFromEvent(event, key)
	if err != nil {
		return "", fmt.Errorf("getting attribute %q from event %T: %w", key, event, err)
	}

	return value, nil
}

// SessionIDFromEvents extracts the "session_id" attribute from an event of the given type in a list of events.
func SessionIDFromEvents(items []types.Event, t interface{}) (uint64, error) {
	key := "session_id"

	// Retrieve the "id" attribute from the specified event type
	value, err := AttributeValueFromEvents(items, t, key)
	if err != nil {
		return 0, fmt.Errorf("getting attribute %q from events: %w", key, err)
	}

	value = strings.Trim(value, `"`)

	// Convert the ID string to uint64
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing value %q: %w", value, err)
	}

	return id, nil
}
