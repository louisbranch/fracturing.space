package static

import "embed"

// FS exposes web2 static assets for HTTP serving.
//
//go:embed *.css *.js
var FS embed.FS
