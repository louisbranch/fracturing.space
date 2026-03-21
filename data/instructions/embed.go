// Package instructions embeds the default instruction file tree.
package instructions

import "embed"

// V1 embeds the v1 instruction directory for use as the default instruction
// set when no override is configured.
//
//go:embed v1
var V1 embed.FS
