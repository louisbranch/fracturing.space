package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/louisbranch/fracturing.space/internal/platform/icons"
)

func main() {
	var outPath string
	var rootFlag string
	flag.StringVar(&outPath, "out", "docs/project/icon-catalog.md", "output path for the icon catalog")
	flag.StringVar(&rootFlag, "root", "", "repo root (defaults to locating go.mod)")
	flag.Parse()

	root, err := resolveRoot(rootFlag)
	if err != nil {
		fatal(err)
	}
	output := outPath
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, outPath)
	}

	content := icons.CatalogMarkdown()
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fatal(fmt.Errorf("create output dir: %w", err))
	}
	if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
		fatal(fmt.Errorf("write catalog: %w", err))
	}
}

// resolveRoot chooses the repository root so generated docs land in the right tree.
func resolveRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		return filepath.Clean(flagRoot), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working dir: %w", err)
	}
	return findModuleRoot(wd)
}

// findModuleRoot walks upward to locate the module root for generation.
func findModuleRoot(start string) (string, error) {
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

// fatal reports a generation error and exits immediately.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
