package assets

import "embed"

//go:embed templates
//go:embed static
var FS embed.FS
