package web

import "embed"

//go:embed static/*.css
var assetsFS embed.FS
