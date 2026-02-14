package config

import (
	"fmt"
	"os"
)

// Exitf writes a formatted error message to stderr and exits with code 1.
// It provides a consistent fatal-exit pattern for CLI entry points.
func Exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
