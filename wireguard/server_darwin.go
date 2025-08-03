package wireguard

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// deviceName reads and returns the WireGuard interface name from its file.
func (s *Server) deviceName() (string, error) {
	// Build the full path to the file containing the interface name
	deviceFile := fmt.Sprintf("/var/run/wireguard/%s.name", s.name)

	// Verify if the file exists before attempting to read it
	exists, err := utils.IsFileExists(deviceFile)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}

	// Open the file for reading
	f, err := os.Open(deviceFile)
	if err != nil {
		return "", err
	}

	// Ensure the file is closed when the function ends
	defer func() { _ = f.Close() }()

	// Create a buffered reader and read the first line (device name)
	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove any trailing newline character and return the name
	return strings.Trim(line, "\n"), nil
}
