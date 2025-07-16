package wireguard

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// interfaceName retrieves the name of the WireGuard interface.
func (s *Server) interfaceName() (string, error) {
	// Opens the file containing the interface name.
	nameFile, err := os.Open(fmt.Sprintf("/var/run/wireguard/%s.name", s.name))
	if err != nil {
		return "", err
	}

	defer func() {
		_ = nameFile.Close()
	}()

	// Reads the interface name from the file.
	reader := bufio.NewReader(nameFile)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.Trim(line, "\n"), nil
}
