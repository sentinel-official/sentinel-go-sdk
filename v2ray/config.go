package v2ray

import (
	"embed"
)

// Embed the template files for V2Ray configurations.
//
//go:embed *.tmpl
var fs embed.FS
