package web

import "embed"

//go:embed static/*.css static/*.js
var assetsFS embed.FS
