package netip

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Port struct {
	InFrom  uint16 `json:"in_from"`
	InTo    uint16 `json:"in_to"`
	OutFrom uint16 `json:"out_from"`
	OutTo   uint16 `json:"out_to"`
}

// InPort returns a string representation of the input port range.
func (p *Port) InPort() string {
	if p.InFrom == p.InTo {
		return fmt.Sprintf("%d", p.InFrom)
	}

	return fmt.Sprintf("%d-%d", p.InFrom, p.InTo)
}

// OutPort returns a string representation of the output port range.
func (p *Port) OutPort() string {
	if p.OutFrom == p.OutTo {
		return fmt.Sprintf("%d", p.OutFrom)
	}

	return fmt.Sprintf("%d-%d", p.OutFrom, p.OutTo)
}

// String provides a string representation of the Port struct.
func (p *Port) String() string {
	switch {
	case p.InFrom == p.InTo && p.OutFrom == p.OutTo && p.InFrom == p.OutFrom:
		return fmt.Sprintf("%d", p.InFrom)
	case p.InFrom == p.InTo && p.OutFrom == p.OutTo:
		return fmt.Sprintf("%d:%d", p.InFrom, p.OutFrom)
	case p.InFrom == p.InTo:
		return fmt.Sprintf("%d:%d-%d", p.InFrom, p.OutFrom, p.OutTo)
	case p.OutFrom == p.OutTo:
		return fmt.Sprintf("%d-%d:%d", p.InFrom, p.InTo, p.OutFrom)
	default:
		return fmt.Sprintf("%d-%d:%d-%d", p.InFrom, p.InTo, p.OutFrom, p.OutTo)
	}
}

// Validate checks if the Port struct values are valid.
func (p *Port) Validate() error {
	if p.InFrom < 1 || p.InTo > 65535 || p.OutFrom < 1 || p.OutTo > 65535 {
		return errors.New("numbers must be between 1 and 65535")
	}
	if p.InFrom > p.InTo {
		return errors.New("in_from cannot be greater than in_to")
	}
	if p.OutFrom > p.OutTo {
		return errors.New("out_from cannot be greater than out_to")
	}
	if (p.InTo - p.InFrom) != (p.OutTo - p.OutFrom) {
		return errors.New("in and out ranges must match in size")
	}

	return nil
}

// MarshalJSON marshals Port as a string using String()
func (p *Port) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON parses a JSON string into a Port
func (p *Port) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	port, err := NewPortFromString(s)
	if err != nil {
		return err
	}
	if port == nil {
		return errors.New("nil port")
	}

	*p = *port
	return nil
}

// NewPortFromString parses a port string and returns a Port struct if the string is valid.
func NewPortFromString(s string) (*Port, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ":")
	if len(parts) > 2 {
		return nil, errors.New("invalid format")
	}

	inRange := parts[0]

	outRange := inRange
	if len(parts) == 2 {
		outRange = parts[1]
	}

	inFrom, inTo, err := parseRange(inRange)
	if err != nil {
		return nil, fmt.Errorf("invalid in range: %w", err)
	}

	outFrom, outTo, err := parseRange(outRange)
	if err != nil {
		return nil, fmt.Errorf("invalid out range: %w", err)
	}

	port := &Port{
		InFrom:  inFrom,
		InTo:    inTo,
		OutFrom: outFrom,
		OutTo:   outTo,
	}

	if err := port.Validate(); err != nil {
		return nil, err
	}

	return port, nil
}

// parseRange parses a range string and returns the start and end as uint16.
func parseRange(s string) (uint16, uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, nil
	}

	parts := strings.Split(s, "-")
	if len(parts) > 2 {
		return 0, 0, errors.New("invalid format")
	}

	from, err := parsePort(parts[0])
	if err != nil {
		return 0, 0, err
	}

	to := from
	if len(parts) == 2 {
		to, err = parsePort(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}

	if from > to {
		return 0, 0, errors.New("from cannot be greater than to")
	}

	return from, to, nil
}

// parsePort converts a string to a uint16 port number.
func parsePort(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, errors.New("number must be between 1 and 65535")
	}

	return uint16(port), nil
}
