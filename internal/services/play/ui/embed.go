package ui

import "embed"

// DistFS exposes built play SPA assets for embedding into the play service.
//
//go:embed all:dist
var DistFS embed.FS
