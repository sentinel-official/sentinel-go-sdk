package utils

import "regexp"

var interfaceNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{1,15}$`)

// IsValidInterfaceName reports whether name is a valid Linux network interface
// name: non-empty, at most 15 characters, and free of shell metacharacters.
func IsValidInterfaceName(name string) bool {
	return interfaceNameRegex.MatchString(name)
}

// HasJSONUnsafeChars reports whether s contains a character that could break out
// of a JSON string literal: a double quote, a backslash, or a control character.
func HasJSONUnsafeChars(s string) bool {
	for _, r := range s {
		if r == '"' || r == '\\' || r < 0x20 {
			return true
		}
	}

	return false
}
