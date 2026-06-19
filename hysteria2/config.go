package hysteria2

import (
	"embed"
)

// Embed the template files for Hysteria2 configurations.
//
//go:embed *.tmpl
var fs embed.FS
