package utils

import (
	"encoding/json"
	"fmt"
	"io"
)

// writeJSON formats the output as JSON and writes it to the provided writer.
func writeJSON(w io.Writer, v interface{}) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	_, _ = fmt.Fprintf(w, "%s", buf)
	return nil
}

// writeText formats the output as YAML and writes it to the provided writer.
func writeText(w io.Writer, v interface{}) error {
	buf, err := YAMLFromJSON(v)
	if err != nil {
		return fmt.Errorf("converting JSON to YAML: %w", err)
	}

	_, _ = fmt.Fprintf(w, "%s", buf)
	return nil
}

// Write formats the output according to the specified format and writes it to the provided writer.
func Write(w io.Writer, v interface{}, format string) error {
	switch format {
	case "json":
		return writeJSON(w, v)
	case "text":
		return writeText(w, v)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

// Writeln writes the formatted output to the writer and adds a newline.
func Writeln(w io.Writer, v interface{}, format string) error {
	if err := Write(w, v, format); err != nil {
		return fmt.Errorf("writing %s output: %w", format, err)
	}

	_, _ = fmt.Fprintln(w)
	return nil
}
