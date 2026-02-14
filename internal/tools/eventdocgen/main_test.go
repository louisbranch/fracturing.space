package main

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindModuleRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := findModuleRoot(nested)
	if err != nil {
		t.Fatalf("findModuleRoot returned error: %v", err)
	}
	if got != root {
		t.Fatalf("expected root %s, got %s", root, got)
	}
}

func TestFindModuleRootMissing(t *testing.T) {
	root := t.TempDir()
	_, err := findModuleRoot(root)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}

func TestParsePackage(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir pkg: %v", err)
	}
	src := strings.Join([]string{
		"package sample",
		"",
		"import \"example.com/event\"",
		"",
		"type Type string",
		"",
		"const (",
		"\tTypeFoo Type = \"foo\"",
		"\tTypeBar event.Type = \"bar\"",
		"\tTypeIgnored string = \"ignored\"",
		")",
		"",
		"type FooPayload struct {",
		"\tID string `json:\"id\"`",
		"\tName string",
		"}",
		"",
		"type Ignored struct {",
		"\tValue string",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(pkgDir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write sample.go: %v", err)
	}

	defs, err := parsePackage(pkgDir, root, "Core")
	if err != nil {
		t.Fatalf("parsePackage returned error: %v", err)
	}
	if len(defs.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(defs.Events))
	}
	payload, ok := defs.Payloads["FooPayload"]
	if !ok {
		t.Fatal("expected FooPayload in payloads")
	}
	if payload.Owner != "Core" {
		t.Fatalf("expected payload owner Core, got %s", payload.Owner)
	}
	if len(payload.Fields) != 2 {
		t.Fatalf("expected 2 payload fields, got %d", len(payload.Fields))
	}
	if payload.Fields[0].JSONTag != "json:\"id\"" {
		t.Fatalf("expected json tag on first field, got %s", payload.Fields[0].JSONTag)
	}
}

func TestPayloadNameForEvent(t *testing.T) {
	if got := payloadNameForEvent("TypeCampaignCreated", "Core"); got != "CampaignCreatedPayload" {
		t.Fatalf("unexpected payload name: %s", got)
	}
	if got := payloadNameForEvent("EventTypeSessionStarted", "Daggerheart"); got != "SessionStartedPayload" {
		t.Fatalf("unexpected payload name: %s", got)
	}
	if got := payloadNameForEvent("EventTypeSessionStarted", "Unknown"); got != "" {
		t.Fatalf("expected empty payload name, got %s", got)
	}
}

func TestRenderCatalog(t *testing.T) {
	defs := packageDefs{
		Events: []eventDef{{
			Owner:     "Core",
			Name:      "TypeFoo",
			Value:     "foo",
			DefinedAt: "internal/foo.go:10",
		}},
		Payloads: map[string]payloadDef{
			"FooPayload": {
				Owner:     "Core",
				Name:      "FooPayload",
				DefinedAt: "internal/foo.go:20",
				Fields: []payloadField{{
					Name:    "ID",
					Type:    "string",
					JSONTag: "json:\"id\"",
				}},
			},
			"UnusedPayload": {
				Owner:     "Core",
				Name:      "UnusedPayload",
				DefinedAt: "internal/foo.go:30",
			},
		},
	}
	emitters := map[string][]string{
		"TypeFoo": {"internal/emit.go:12"},
	}
	output, err := renderCatalog([]packageDefs{defs}, emitters)
	if err != nil {
		t.Fatalf("renderCatalog returned error: %v", err)
	}
	checks := []string{
		"## Core Events",
		"### `foo` (`TypeFoo`)",
		"Payload: `FooPayload`",
		"`ID (json:\"id\")`: `string`",
		"### Unmapped Payloads",
		"`UnusedPayload`",
		"Emitters:",
		"`internal/emit.go:12`",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Fatalf("expected output to contain %q", check)
		}
	}
}

func TestScanEmitters(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "emitters")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir emitters: %v", err)
	}
	src := strings.Join([]string{
		"package sample",
		"",
		"import \"example.com/event\"",
		"",
		"func emit() {",
		"\t_ = event.Event{Type: event.TypeFoo}",
		"\t_ = event.Event{Type: TypeBar}",
		"}",
	}, "\n")
	path := filepath.Join(dir, "emit.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write emit.go: %v", err)
	}

	emitters, err := scanEmitters(root, root)
	if err != nil {
		t.Fatalf("scanEmitters returned error: %v", err)
	}
	if len(emitters["TypeFoo"]) != 1 {
		t.Fatalf("expected one emitter for TypeFoo, got %d", len(emitters["TypeFoo"]))
	}
	if len(emitters["TypeBar"]) != 1 {
		t.Fatalf("expected one emitter for TypeBar, got %d", len(emitters["TypeBar"]))
	}
	if !strings.HasPrefix(emitters["TypeFoo"][0], "emitters/emit.go:") {
		t.Fatalf("unexpected emitter path: %s", emitters["TypeFoo"][0])
	}
}

func TestFormatPosition(t *testing.T) {
	pos := formatPosition(tokenPosition("/root/pkg/file.go", 12), "/root")
	if pos != "pkg/file.go:12" {
		t.Fatalf("expected formatted position, got %s", pos)
	}
}

func tokenPosition(file string, line int) token.Position {
	return token.Position{Filename: file, Line: line}
}
