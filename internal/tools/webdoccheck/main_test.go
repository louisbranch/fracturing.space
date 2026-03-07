package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunUnsupportedMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-mode", "invalid"}, &stdout, &stderr)
	if got := exitCode(err); got != 2 {
		t.Fatalf("exitCode(run) = %d, want 2 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `webdoccheck: unsupported mode "invalid"`) {
		t.Fatalf("stderr = %q, want unsupported mode message", stderr.String())
	}
}

func TestRunUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-nope"}, &stdout, &stderr)
	if got := exitCode(err); got != 2 {
		t.Fatalf("exitCode(run) = %d, want 2 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q, want flag parse error", stderr.String())
	}
}

func TestRunDeclarationsWriteBaselineSorted(t *testing.T) {
	prev := scanMissingDeclarationComments
	scanMissingDeclarationComments = func() ([]declarationEntry, error) {
		return []declarationEntry{
			{Path: "b.go", Line: 10, Kind: "func", Name: "B"},
			{Path: "a.go", Line: 2, Kind: "type", Name: "A"},
		}, nil
	}
	t.Cleanup(func() {
		scanMissingDeclarationComments = prev
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-mode", "declarations", "-write-baseline"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned err: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	want := "a.go:2 type A\nb.go:10 func B\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunDeclarationsViolationExitCodeOne(t *testing.T) {
	prev := scanMissingDeclarationComments
	scanMissingDeclarationComments = func() ([]declarationEntry, error) {
		return []declarationEntry{{Path: "a.go", Line: 3, Kind: "func", Name: "Alpha"}}, nil
	}
	t.Cleanup(func() {
		scanMissingDeclarationComments = prev
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-mode", "declarations"}, &stdout, &stderr)
	if got := exitCode(err); got != 1 {
		t.Fatalf("exitCode(run) = %d, want 1 (err=%v)", got, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "webdoccheck: missing declaration comments") {
		t.Fatalf("stdout = %q, want declaration violation header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "a.go:3 func Alpha") {
		t.Fatalf("stdout = %q, want declaration violation line", stdout.String())
	}
}

func TestRunPackagesRejectsBaseline(t *testing.T) {
	prev := scanMissingPackageComments
	scanMissingPackageComments = func() ([]packageEntry, error) {
		return nil, nil
	}
	t.Cleanup(func() {
		scanMissingPackageComments = prev
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-mode", "packages", "-baseline", "baseline.txt"}, &stdout, &stderr)
	if got := exitCode(err); got != 2 {
		t.Fatalf("exitCode(run) = %d, want 2 (err=%v)", got, err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "webdoccheck: -baseline is not supported in packages mode") {
		t.Fatalf("stderr = %q, want baseline unsupported message", stderr.String())
	}
}
