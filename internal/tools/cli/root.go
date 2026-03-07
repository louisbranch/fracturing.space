package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveRoot resolves the repository root for tooling commands.
func ResolveRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		return filepath.Clean(flagRoot), nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working dir: %w", err)
	}
	return FindModuleRoot(wd)
}

// FindModuleRoot walks upward from start until it finds go.mod.
func FindModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found above %s", start)
}

// ResolvePath joins path to root unless path is already absolute.
func ResolvePath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
