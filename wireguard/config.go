package wireguard

import (
	"embed"
)

// Embed the template files for WireGuard configurations.
//
//go:embed *.tmpl
var fs embed.FS
