// Package main generates the icon catalog from the shared icon registry.
//
// It is a docs-oriented tooling boundary: source metadata drives generated UX
// documentation, and no runtime behavior depends on this tool.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/louisbranch/fracturing.space/internal/platform/icons"
	"github.com/louisbranch/fracturing.space/internal/tools/cli"
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
	flags.StringVar(&outPath, "out", "docs/reference/icon-catalog.md", "output path for the icon catalog")
	flags.StringVar(&rootFlag, "root", "", "repo root (defaults to locating go.mod)")
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}

	root, err := cli.ResolveRoot(rootFlag)
	if err != nil {
		return err
	}
	output := cli.ResolvePath(root, outPath)

	content := fmt.Sprintf(`---
title: "Icon Catalog"
parent: "Reference"
nav_order: 11
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

// fatal reports a generation error and exits immediately.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
