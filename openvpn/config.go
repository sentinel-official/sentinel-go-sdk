package openvpn

import (
	"embed"
)

// Embed the template files for OpenVPN configurations.
//
//go:embed *.tmpl
var fs embed.FS
