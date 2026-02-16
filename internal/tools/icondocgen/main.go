package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/louisbranch/fracturing.space/internal/platform/icons"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fatal(err)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stdout
	var outPath string
	var rootFlag string
	flags := flag.NewFlagSet("icondocgen", flag.ContinueOnError)
	flags.StringVar(&outPath, "out", "docs/project/icon-catalog.md", "output path for the icon catalog")
	flags.StringVar(&rootFlag, "root", "", "repo root (defaults to locating go.mod)")
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}

	root, err := resolveRoot(rootFlag)
	if err != nil {
		return err
	}
	output := outPath
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, outPath)
	}

	content := fmt.Sprintf(`---
title: "Icon Catalog"
parent: "Project"
nav_order: 30
---

%s`, icons.CatalogMarkdown())
	if err := writeOutput(output, content); err != nil {
		return err
	}
	return nil
}

func writeOutput(output, content string) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
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
