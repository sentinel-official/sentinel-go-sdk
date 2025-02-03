package input

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/mattn/go-isatty"
)

// isTTY checks if the standard input is a terminal.
func isTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// readLineFromBuf reads a single line from the given buffered reader.
func readLineFromBuf(buf *bufio.Reader) (string, error) {
	line, err := buf.ReadString('\n')

	switch {
	case errors.Is(err, io.EOF):
		// If there's an EOF but we have some data, return it
		if len(line) > 0 {
			break
		}
		return "", err
	case err != nil:
		return "", err
	}

	return strings.TrimSpace(line), nil
}

// GetConfirmation prompts the user with a message and returns true if the response starts with 'y' or 'Y'.
func GetConfirmation(prompt string, buf *bufio.Reader) (bool, error) {
	line, err := GetString(prompt, buf)
	if err != nil {
		return false, err
	}

	line = strings.ToLower(line)
	if len(line) > 0 && line[0] == 'y' {
		return true, nil
	}

	return false, nil
}

// GetPassword prompts the user for a password. If the standard input is a terminal, use a secure prompt.
func GetPassword(prompt string, buf *bufio.Reader) (string, error) {
	if prompt != "" && isTTY() {
		return speakeasy.FAsk(os.Stderr, prompt)
	}

	return readLineFromBuf(buf)
}

// GetString prompts the user with a message and reads a line from the buffer or terminal.
func GetString(prompt string, buf *bufio.Reader) (string, error) {
	if prompt != "" && isTTY() {
		_, _ = fmt.Fprint(os.Stderr, prompt)
	}

	return readLineFromBuf(buf)
}
