package utils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"stringsJoin":  strings.Join,
	"sum":          func(x, y int) int { return x + y },
	"filepathJoin": filepath.Join,
}

// ExecTemplateToFile generates content from a template and writes it to a file.
func ExecTemplateToFile(text string, data interface{}, filename string) error {
	// Parse the template with custom functions
	tmpl, err := template.New("config").Funcs(funcMap).Parse(text)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Execute the template and capture the output
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Write the generated content to the specified file
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing to file %q: %w", filename, err)
	}

	return nil
}
