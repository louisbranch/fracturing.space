package web

import "embed"

// assetsFS stores compiled web assets so handlers can serve CSS/JS from a single
// deployable binary without runtime filesystem dependencies.
//
//go:embed static/*.css static/*.js
var assetsFS embed.FS
