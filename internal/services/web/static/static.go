package static

import "embed"

// FS exposes web static assets for HTTP serving.
//
//go:embed *.css *.js *.svg
var FS embed.FS
