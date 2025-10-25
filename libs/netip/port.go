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

// NewPortFromString parses a port string and returns a Port struct if the string is valid.
func NewPortFromString(s string) (*Port, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ":")
	if len(parts) > 2 {
		return nil, fmt.Errorf("too many colons in port %q", s)
	}

	inRange := parts[0]

	outRange := inRange
	if len(parts) == 2 {
		outRange = parts[1]
	}

	inFrom, inTo, err := parseRange(inRange)
	if err != nil {
		return nil, fmt.Errorf("parsing in range %q: %w", inRange, err)
	}

	outFrom, outTo, err := parseRange(outRange)
	if err != nil {
		return nil, fmt.Errorf("parsing out range %q: %w", outRange, err)
	}

	port := &Port{
		InFrom:  inFrom,
		InTo:    inTo,
		OutFrom: outFrom,
		OutTo:   outTo,
	}

	if err := port.Validate(); err != nil {
		return nil, fmt.Errorf("validating port %q: %w", port, err)
	}

	return port, nil
}

// InPort returns a string representation of the input port range.
func (p *Port) InPort() string {
	if p.InFrom == p.InTo {
		return strconv.FormatUint(uint64(p.InFrom), 10)
	}

	return fmt.Sprintf("%d-%d", p.InFrom, p.InTo)
}

// OutPort returns a string representation of the output port range.
func (p *Port) OutPort() string {
	if p.OutFrom == p.OutTo {
		return strconv.FormatUint(uint64(p.OutFrom), 10)
	}

	return fmt.Sprintf("%d-%d", p.OutFrom, p.OutTo)
}

// String provides a string representation of the Port struct.
func (p *Port) String() string {
	switch {
	case p.InFrom == p.InTo && p.OutFrom == p.OutTo && p.InFrom == p.OutFrom:
		return strconv.FormatUint(uint64(p.InFrom), 10)
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
	if p.InFrom < 1 || p.OutFrom < 1 {
		return fmt.Errorf("port numbers are out of range 1-65535 (got: %d-%d:%d-%d)",
			p.InFrom, p.InTo, p.OutFrom, p.OutTo)
	}

	if p.InFrom > p.InTo {
		return fmt.Errorf("in_from %d is greater than in_to %d", p.InFrom, p.InTo)
	}

	if p.OutFrom > p.OutTo {
		return fmt.Errorf("out_from %d is greater than out_to %d", p.OutFrom, p.OutTo)
	}

	if (p.InTo - p.InFrom) != (p.OutTo - p.OutFrom) {
		return fmt.Errorf("in range size %d does not match out range size %d", p.InTo-p.InFrom, p.OutTo-p.OutFrom)
	}

	return nil
}

// MarshalJSON marshals Port as a string using String().
func (p *Port) MarshalJSON() ([]byte, error) {
	buf, err := json.Marshal(p.String())
	if err != nil {
		return nil, fmt.Errorf("marshaling port string: %w", err)
	}

	return buf, nil
}

// UnmarshalJSON parses a JSON string into a Port.
func (p *Port) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("unmarshaling port from data: %w", err)
	}

	port, err := NewPortFromString(s)
	if err != nil {
		return fmt.Errorf("parsing port: %w", err)
	}

	if port == nil {
		return errors.New("nil port")
	}

	*p = *port

	return nil
}

// parseRange parses a range string and returns the start and end as uint16.
func parseRange(s string) (uint16, uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, nil
	}

	parts := strings.Split(s, "-")
	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("too many dashes in port %q", s)
	}

	from, err := parsePort(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing from port %q: %w", parts[0], err)
	}

	to := from
	if len(parts) == 2 {
		to, err = parsePort(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("parsing to port %q: %w", parts[1], err)
		}
	}

	if from > to {
		return 0, 0, fmt.Errorf("from port %d is greater than to port %d", from, to)
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
		return 0, fmt.Errorf("parsing port %q: %w", s, err)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port %d is out of range 1-65535", port)
	}

	return uint16(port), nil
}
