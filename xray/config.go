package xray

import (
	"embed"
)

// Embed the template files for Xray configurations.
//
//go:embed *.tmpl
var fs embed.FS
