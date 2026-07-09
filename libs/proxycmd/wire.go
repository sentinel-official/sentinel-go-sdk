package proxycmd

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// appendString appends a string field with the given field number.
func appendString(b []byte, num protowire.Number, v string) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendString(b, v)

	return b
}

// appendBytes appends a bytes field with the given field number.
func appendBytes(b []byte, num protowire.Number, v []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendBytes(b, v)

	return b
}

// appendBool appends a bool field with the given field number.
func appendBool(b []byte, num protowire.Number, v bool) []byte {
	b = protowire.AppendTag(b, num, protowire.VarintType)

	if v {
		b = protowire.AppendVarint(b, 1)
	} else {
		b = protowire.AppendVarint(b, 0)
	}

	return b
}

// appendMessage appends a nested-message (bytes) field with the given field number.
func appendMessage(b []byte, num protowire.Number, v []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendBytes(b, v)

	return b
}

// typedMessage encodes a {string=1; bytes=2} wrapper message, which is the
// wire-identical layout shared by xray serial.TypedMessage and google.protobuf.Any.
func typedMessage(typeURL string, value []byte) []byte {
	var b []byte

	if typeURL != "" {
		b = appendString(b, 1, typeURL)
	}

	if len(value) > 0 {
		b = appendBytes(b, 2, value)
	}

	return b
}

// parseQueryStatsResponse decodes a QueryStatsResponse wire message and returns
// the slice of Stat entries found in repeated field 1.
func parseQueryStatsResponse(data []byte) ([]Stat, error) {
	var stats []Stat

	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("parsing QueryStatsResponse tag: %w", protowire.ParseError(n))
		}

		data = data[n:]

		if num == 1 && typ == protowire.BytesType {
			sub, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, fmt.Errorf("parsing QueryStatsResponse stat bytes: %w", protowire.ParseError(n))
			}

			data = data[n:]

			stat, err := parseStat(sub)
			if err != nil {
				return nil, fmt.Errorf("parsing stat entry: %w", err)
			}

			stats = append(stats, stat)

			continue
		}

		// Skip unknown fields.
		n = protowire.ConsumeFieldValue(num, typ, data)
		if n < 0 {
			return nil, fmt.Errorf("skipping QueryStatsResponse field %d: %w", num, protowire.ParseError(n))
		}

		data = data[n:]
	}

	return stats, nil
}

// parseStat decodes a single Stat submessage: name (field 1, string) and value (field 2, int64/varint).
func parseStat(data []byte) (Stat, error) {
	var s Stat

	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return Stat{}, fmt.Errorf("parsing stat tag: %w", protowire.ParseError(n))
		}

		data = data[n:]

		switch {
		case num == 1 && typ == protowire.BytesType:
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return Stat{}, fmt.Errorf("parsing stat name: %w", protowire.ParseError(n))
			}

			data = data[n:]
			s.Name = string(v)
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return Stat{}, fmt.Errorf("parsing stat value: %w", protowire.ParseError(n))
			}

			data = data[n:]
			s.Value = int64(v)
		default:
			n = protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return Stat{}, fmt.Errorf("skipping stat field %d: %w", num, protowire.ParseError(n))
			}

			data = data[n:]
		}
	}

	return s, nil
}
