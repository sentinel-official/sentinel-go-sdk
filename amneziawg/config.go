package amneziawg

import (
	"embed"
)

// defaultDevice is the default AmneziaWG network interface name.
const defaultDevice = "awg0"

// Embed the template files for AmneziaWG configurations.
//
//go:embed *.tmpl
var fs embed.FS
