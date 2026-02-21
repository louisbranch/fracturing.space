package main

import (
	"bytes"
	"flag"
	"go/ast"
	"go/parser"
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
	if got := payloadNameForEvent("EventTypeSessionStarted", "Unknown"); got != "SessionStartedPayload" {
		t.Fatalf("unexpected payload name for unknown owner: %s", got)
	}
}

func TestRenderCatalog(t *testing.T) {
	defs := packageDefs{
		Events: []eventDef{
			{
				Owner:     "Core",
				Name:      "TypeFoo",
				Value:     "action.foo",
				DefinedAt: "internal/foo.go:10",
			},
			{
				Owner:     "Core",
				Name:      "TypeBar",
				Value:     "session.bar",
				DefinedAt: "internal/bar.go:11",
			},
		},
		Payloads: map[string]payloadDef{
			"FooPayload": {
				Owner:     "Core",
				Name:      "FooPayload",
				DefinedAt: "internal/foo.go:20",
				Fields: []payloadField{
					{
						Name:    "ID",
						Type:    "string",
						JSONTag: "json:\"id\"",
					},
					{
						Name:    "Note",
						Type:    "string",
						JSONTag: "json:\"note,omitempty\"",
					},
				},
			},
			"BarPayload": {
				Owner:     "Core",
				Name:      "BarPayload",
				DefinedAt: "internal/bar.go:21",
			},
			"UnusedPayload": {
				Owner:     "Core",
				Name:      "UnusedPayload",
				DefinedAt: "internal/foo.go:30",
			},
		},
	}
	emitters := map[string][]string{
		"action.foo": {"internal/emit.go:12"},
	}
	output, err := renderCatalog([]packageDefs{defs}, emitters)
	if err != nil {
		t.Fatalf("renderCatalog returned error: %v", err)
	}
	checks := []string{
		"## Core Events",
		"### Summary",
		"| Event | Namespace | Name | Constant | Payload | Emitters |",
		"| `action.foo` | `action` | `foo` | `TypeFoo` | `FooPayload` | 1 |",
		"| `session.bar` | `session` | `bar` | `TypeBar` | `BarPayload` | 0 |",
		"### Namespace `action`",
		"#### `action.foo`",
		"- Constant: `TypeFoo`",
		"- Payload: `FooPayload` (`internal/foo.go:20`)",
		"| `ID` | `id` | `string` | yes |",
		"| `Note` | `note` | `string` | no |",
		"### Namespace `session`",
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

func TestRenderCatalog_UsesEventValueEmitterAndPayloadMapping(t *testing.T) {
	packages := []packageDefs{
		{
			Events: []eventDef{
				{
					Owner:     "Core",
					Name:      "eventTypeCreated",
					Value:     "campaign.created",
					DefinedAt: "internal/services/game/domain/campaign/decider.go:20",
					Payload:   "CreatePayload",
				},
			},
			Payloads: map[string]payloadDef{
				"CreatePayload": {
					Owner:     "Core",
					Name:      "CreatePayload",
					DefinedAt: "internal/services/game/domain/campaign/payload.go:4",
					Fields: []payloadField{
						{Name: "Name", Type: "string", JSONTag: "json:\"name\""},
					},
				},
			},
		},
		{
			Events: []eventDef{
				{
					Owner:     "Core",
					Name:      "eventTypeCreated",
					Value:     "invite.created",
					DefinedAt: "internal/services/game/domain/invite/decider.go:17",
					Payload:   "CreatePayload",
				},
			},
			Payloads: map[string]payloadDef{
				"CreatePayload": {
					Owner:     "Core",
					Name:      "CreatePayload",
					DefinedAt: "internal/services/game/domain/invite/payload.go:4",
					Fields: []payloadField{
						{Name: "InviteID", Type: "string", JSONTag: "json:\"invite_id\""},
					},
				},
			},
		},
	}

	emitters := map[string][]string{
		"campaign.created": {"internal/services/game/domain/campaign/decider.go:100"},
		"invite.created":   {"internal/services/game/domain/invite/decider.go:69"},
	}

	output, err := renderCatalog(packages, emitters)
	if err != nil {
		t.Fatalf("renderCatalog returned error: %v", err)
	}

	checks := []string{
		"| `campaign.created` | `campaign` | `created` | `eventTypeCreated` | `CreatePayload` | 1 |",
		"| `invite.created` | `invite` | `created` | `eventTypeCreated` | `CreatePayload` | 1 |",
		"- Payload: `CreatePayload` (`internal/services/game/domain/campaign/payload.go:4`)",
		"- Payload: `CreatePayload` (`internal/services/game/domain/invite/payload.go:4`)",
		"| `Name` | `name` | `string` | yes |",
		"| `InviteID` | `invite_id` | `string` | yes |",
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

func TestResolveRoot(t *testing.T) {
	t.Run("explicit flag", func(t *testing.T) {
		got, err := resolveRoot("/some/path")
		if err != nil {
			t.Fatal(err)
		}
		if got != "/some/path" {
			t.Errorf("got %q, want /some/path", got)
		}
	})

	t.Run("empty flag uses cwd", func(t *testing.T) {
		// From the project root, findModuleRoot should succeed.
		got, err := resolveRoot("")
		if err != nil {
			t.Fatal(err)
		}
		if got == "" {
			t.Error("expected non-empty root")
		}
	})
}

func TestSelectValueExpr(t *testing.T) {
	a := &ast.BasicLit{Kind: token.STRING, Value: `"a"`}
	b := &ast.BasicLit{Kind: token.STRING, Value: `"b"`}

	t.Run("empty list", func(t *testing.T) {
		if got := selectValueExpr(nil, 0); got != nil {
			t.Error("expected nil for empty list")
		}
	})

	t.Run("single value any index", func(t *testing.T) {
		if got := selectValueExpr([]ast.Expr{a}, 5); got != a {
			t.Error("expected single value to always be returned")
		}
	})

	t.Run("multi value in range", func(t *testing.T) {
		if got := selectValueExpr([]ast.Expr{a, b}, 1); got != b {
			t.Error("expected second element")
		}
	})

	t.Run("multi value out of range", func(t *testing.T) {
		if got := selectValueExpr([]ast.Expr{a, b}, 5); got != nil {
			t.Error("expected nil for out-of-range index")
		}
	})
}

func TestEventNameFromExpr(t *testing.T) {
	t.Run("selector expr", func(t *testing.T) {
		e := &ast.SelectorExpr{
			X:   &ast.Ident{Name: "event"},
			Sel: &ast.Ident{Name: "TypeFoo"},
		}
		if got := eventNameFromExpr(e); got != "TypeFoo" {
			t.Errorf("got %q, want TypeFoo", got)
		}
	})

	t.Run("ident expr", func(t *testing.T) {
		e := &ast.Ident{Name: "TypeBar"}
		if got := eventNameFromExpr(e); got != "TypeBar" {
			t.Errorf("got %q, want TypeBar", got)
		}
	})

	t.Run("other expr", func(t *testing.T) {
		e := &ast.BasicLit{Kind: token.STRING, Value: `"literal"`}
		if got := eventNameFromExpr(e); got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestUnmappedPayloads(t *testing.T) {
	t.Run("nil payloads", func(t *testing.T) {
		result := unmappedPayloads(nil, nil)
		if result != nil {
			t.Error("expected nil for nil payloads")
		}
	})

	t.Run("all used", func(t *testing.T) {
		payloads := map[string]payloadDef{"A": {Name: "A"}}
		used := map[string]struct{}{"A": {}}
		result := unmappedPayloads(payloads, used)
		if len(result) != 0 {
			t.Errorf("expected 0 unmapped, got %d", len(result))
		}
	})

	t.Run("some unmapped", func(t *testing.T) {
		payloads := map[string]payloadDef{
			"A": {Name: "A"},
			"B": {Name: "B"},
			"C": {Name: "C"},
		}
		used := map[string]struct{}{"A": {}}
		result := unmappedPayloads(payloads, used)
		if len(result) != 2 {
			t.Fatalf("expected 2 unmapped, got %d", len(result))
		}
		// Should be sorted by name.
		if result[0].Name != "B" || result[1].Name != "C" {
			t.Errorf("expected [B, C], got [%s, %s]", result[0].Name, result[1].Name)
		}
	})
}

func TestExprString(t *testing.T) {
	fset := token.NewFileSet()
	e := &ast.Ident{Name: "MyType"}
	got := exprString(fset, e)
	if got != "MyType" {
		t.Errorf("got %q, want MyType", got)
	}
}

func TestParsePayloadFields(t *testing.T) {
	t.Run("nil fields", func(t *testing.T) {
		result := parsePayloadFields(nil, token.NewFileSet())
		if result != nil {
			t.Error("expected nil for nil fields")
		}
	})

	t.Run("embedded field skipped", func(t *testing.T) {
		// Parse a struct with an embedded field (no names).
		src := `package x; type S struct { Embedded; Name string }` //nolint
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		var structType *ast.StructType
		ast.Inspect(file, func(n ast.Node) bool {
			if st, ok := n.(*ast.StructType); ok {
				structType = st
				return false
			}
			return true
		})
		if structType == nil {
			t.Fatal("struct not found")
		}
		fields := parsePayloadFields(structType.Fields, fset)
		// Only "Name" should be returned (embedded "Embedded" is skipped).
		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].Name != "Name" {
			t.Errorf("got field %q, want Name", fields[0].Name)
		}
	})
}

func TestRenderCatalog_EmptyPackage(t *testing.T) {
	// A package with no events should produce no section.
	defs := packageDefs{Events: nil, Payloads: map[string]payloadDef{}}
	output, err := renderCatalog([]packageDefs{defs}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "## ") {
		t.Error("expected no section header for empty package")
	}
}

func TestRenderCatalog_NoPayload(t *testing.T) {
	defs := packageDefs{
		Events: []eventDef{{
			Owner:     "Core",
			Name:      "TypeOrphan",
			Value:     "orphan",
			DefinedAt: "foo.go:1",
		}},
		Payloads: map[string]payloadDef{},
	}
	output, err := renderCatalog([]packageDefs{defs}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Payload: not found") {
		t.Error("expected 'Payload: not found' for event without matching payload")
	}
}

func TestSplitEventValue(t *testing.T) {
	t.Run("with namespace", func(t *testing.T) {
		namespace, name := splitEventValue("session.started")
		if namespace != "session" {
			t.Fatalf("namespace = %q, want session", namespace)
		}
		if name != "started" {
			t.Fatalf("name = %q, want started", name)
		}
	})

	t.Run("without namespace", func(t *testing.T) {
		namespace, name := splitEventValue("orphan")
		if namespace != "(none)" {
			t.Fatalf("namespace = %q, want (none)", namespace)
		}
		if name != "orphan" {
			t.Fatalf("name = %q, want orphan", name)
		}
	})

	t.Run("empty", func(t *testing.T) {
		namespace, name := splitEventValue("")
		if namespace != "(none)" {
			t.Fatalf("namespace = %q, want (none)", namespace)
		}
		if name != "(none)" {
			t.Fatalf("name = %q, want (none)", name)
		}
	})
}

func TestParseJSONTag(t *testing.T) {
	t.Run("required tagged field", func(t *testing.T) {
		name, required := parseJSONTag(`json:"id"`, "ID")
		if name != "id" {
			t.Fatalf("name = %q, want id", name)
		}
		if !required {
			t.Fatal("expected required field")
		}
	})

	t.Run("omitempty tagged field", func(t *testing.T) {
		name, required := parseJSONTag(`json:"note,omitempty"`, "Note")
		if name != "note" {
			t.Fatalf("name = %q, want note", name)
		}
		if required {
			t.Fatal("expected optional field")
		}
	})

	t.Run("missing tag", func(t *testing.T) {
		name, required := parseJSONTag("", "DisplayName")
		if name != "DisplayName" {
			t.Fatalf("name = %q, want DisplayName", name)
		}
		if !required {
			t.Fatal("expected required field for untagged field")
		}
	})

	t.Run("unexpected tag format", func(t *testing.T) {
		name, required := parseJSONTag("bad", "DisplayName")
		if name != "bad" {
			t.Fatalf("name = %q, want bad", name)
		}
		if !required {
			t.Fatal("expected required field for unknown tag format")
		}
	})
}

func TestWriteOutput(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "docs", "events", "out.md")
	if err := writeOutput(out, "content", "catalog"); err != nil {
		t.Fatalf("writeOutput returned error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "content" {
		t.Fatalf("unexpected output content: %q", string(data))
	}
}

func TestMainGeneratesCatalogAndUsageMap(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/louisbranch/fracturing.space\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	coreDir := filepath.Join(root, "internal", "services", "game", "domain", "campaign")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("mkdir core dir: %v", err)
	}
	coreEvent := strings.Join([]string{
		"package campaign",
		"",
		`import event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"`,
		"",
		"const (",
		"\tTypeCampaignCreated event.Type = \"campaign.created\"",
		")",
	}, "\n")
	if err := os.WriteFile(filepath.Join(coreDir, "event.go"), []byte(coreEvent), 0o644); err != nil {
		t.Fatalf("write core event file: %v", err)
	}
	corePayload := strings.Join([]string{
		"package campaign",
		"",
		"type CampaignCreatedPayload struct {",
		"\tName string `json:\"name\"`",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(coreDir, "payload.go"), []byte(corePayload), 0o644); err != nil {
		t.Fatalf("write core payload file: %v", err)
	}

	for _, dir := range []string{
		filepath.Join(root, "internal", "services", "game", "domain", "character"),
		filepath.Join(root, "internal", "services", "game", "domain", "invite"),
		filepath.Join(root, "internal", "services", "game", "domain", "participant"),
		filepath.Join(root, "internal", "services", "game", "domain", "session"),
		filepath.Join(root, "internal", "services", "game", "domain", "action"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir package dir %s: %v", dir, err)
		}
	}

	daggerheartDir := filepath.Join(root, "internal", "services", "game", "domain", "systems", "daggerheart")
	if err := os.MkdirAll(daggerheartDir, 0o755); err != nil {
		t.Fatalf("mkdir daggerheart dir: %v", err)
	}
	daggerheartEvents := strings.Join([]string{
		"package daggerheart",
		"",
		"import event \"github.com/louisbranch/fracturing.space/internal/services/game/domain/event\"",
		"",
		"const (",
		"\tEventTypeAlphaDone event.Type = \"action.alpha_done\"",
		")",
	}, "\n")
	if err := os.WriteFile(filepath.Join(daggerheartDir, "event_types.go"), []byte(daggerheartEvents), 0o644); err != nil {
		t.Fatalf("write daggerheart event file: %v", err)
	}
	daggerheartPayload := strings.Join([]string{
		"package daggerheart",
		"",
		"type AlphaDonePayload struct {",
		"\tID string `json:\"id\"`",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(daggerheartDir, "payload.go"), []byte(daggerheartPayload), 0o644); err != nil {
		t.Fatalf("write daggerheart payload file: %v", err)
	}

	eventDir := filepath.Join(root, "internal", "services", "game", "domain", "event")
	if err := os.MkdirAll(eventDir, 0o755); err != nil {
		t.Fatalf("mkdir event dir: %v", err)
	}
	eventPkg := strings.Join([]string{
		"package event",
		"",
		"type Type string",
		"",
		"type Event struct {",
		"\tType Type",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(eventDir, "event.go"), []byte(eventPkg), 0o644); err != nil {
		t.Fatalf("write event package file: %v", err)
	}

	emitterDir := filepath.Join(root, "internal", "services", "game", "domain", "sample")
	if err := os.MkdirAll(emitterDir, 0o755); err != nil {
		t.Fatalf("mkdir emitter dir: %v", err)
	}
	emitterSrc := strings.Join([]string{
		"package sample",
		"",
		"import event \"github.com/louisbranch/fracturing.space/internal/services/game/domain/event\"",
		`import camp "github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"`,
		"",
		"func emit() {",
		"\t_ = event.Event{Type: camp.TypeCampaignCreated}",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(emitterDir, "emit.go"), []byte(emitterSrc), 0o644); err != nil {
		t.Fatalf("write emitter file: %v", err)
	}

	catalogOut := "docs/events/generated-catalog.md"
	usageOut := "docs/events/generated-usage.md"

	oldArgs := os.Args
	oldFlagSet := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagSet
	}()

	flag.CommandLine = flag.NewFlagSet("eventdocgen-test", flag.ContinueOnError)
	os.Args = []string{
		"eventdocgen-test",
		"-root", root,
		"-out", catalogOut,
		"-usage-out", usageOut,
		"-commands-out", "",
	}

	main()

	catalogData, err := os.ReadFile(filepath.Join(root, catalogOut))
	if err != nil {
		t.Fatalf("read generated catalog: %v", err)
	}
	if !strings.Contains(string(catalogData), "## Core Events") {
		t.Fatalf("catalog missing Core Events section:\n%s", string(catalogData))
	}
	if !strings.Contains(string(catalogData), "## Daggerheart Events") {
		t.Fatalf("catalog missing Daggerheart Events section:\n%s", string(catalogData))
	}

	usageData, err := os.ReadFile(filepath.Join(root, usageOut))
	if err != nil {
		t.Fatalf("read generated usage map: %v", err)
	}
	if !strings.Contains(string(usageData), "# Event Usage Map") {
		t.Fatalf("usage map missing title:\n%s", string(usageData))
	}
	if !strings.Contains(string(usageData), "### `campaign.created`") {
		t.Fatalf("usage map missing campaign.created event:\n%s", string(usageData))
	}
}

func TestBuildEventValueLookup(t *testing.T) {
	packages := []packageDefs{
		{
			Events: []eventDef{
				{Name: "TypeOne", Value: "one.created"},
				{Name: "TypeTwo", Value: "two.updated"},
				{Name: "", Value: "invalid"},
				{Name: "TypeThree", Value: ""},
			},
		},
	}

	lookup := buildEventValueLookup(packages)
	if lookup["TypeOne"] != "one.created" {
		t.Fatalf("TypeOne lookup mismatch: %q", lookup["TypeOne"])
	}
	if lookup["TypeTwo"] != "two.updated" {
		t.Fatalf("TypeTwo lookup mismatch: %q", lookup["TypeTwo"])
	}
	if _, ok := lookup[""]; ok {
		t.Fatal("expected empty-name event to be ignored")
	}
	if _, ok := lookup["TypeThree"]; ok {
		t.Fatal("expected empty-value event to be ignored")
	}
}

func TestBuildLocalConstLookup(t *testing.T) {
	src := strings.Join([]string{
		"package sample",
		"",
		"import \"example.com/event\"",
		"",
		"const (",
		"\tBaseType = event.Type(\"campaign.created\")",
		"\tAliasType = BaseType",
		"\tRawType = \"session.started\"",
		")",
	}, "\n")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	lookup := buildLocalConstLookup(file, map[string]string{"SeedType": "seed.type"})
	if lookup["SeedType"] != "seed.type" {
		t.Fatalf("seed lookup mismatch: %q", lookup["SeedType"])
	}
	if lookup["BaseType"] != "campaign.created" {
		t.Fatalf("BaseType lookup mismatch: %q", lookup["BaseType"])
	}
	if lookup["AliasType"] != "campaign.created" {
		t.Fatalf("AliasType lookup mismatch: %q", lookup["AliasType"])
	}
	if lookup["RawType"] != "session.started" {
		t.Fatalf("RawType lookup mismatch: %q", lookup["RawType"])
	}
}

func TestEventValueFromExpr(t *testing.T) {
	lookup := map[string]string{
		"TypeFoo": "campaign.created",
	}

	t.Run("string literal", func(t *testing.T) {
		expr := &ast.BasicLit{Kind: token.STRING, Value: `"session.started"`}
		if got := eventValueFromExpr(expr, lookup); got != "session.started" {
			t.Fatalf("got %q, want session.started", got)
		}
	})

	t.Run("call wrapper", func(t *testing.T) {
		expr := &ast.CallExpr{
			Fun:  &ast.Ident{Name: "Type"},
			Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"invite.created"`}},
		}
		if got := eventValueFromExpr(expr, lookup); got != "invite.created" {
			t.Fatalf("got %q, want invite.created", got)
		}
	})

	t.Run("ident lookup", func(t *testing.T) {
		expr := &ast.Ident{Name: "TypeFoo"}
		if got := eventValueFromExpr(expr, lookup); got != "campaign.created" {
			t.Fatalf("got %q, want campaign.created", got)
		}
	})

	t.Run("selector lookup", func(t *testing.T) {
		expr := &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "TypeFoo"}}
		if got := eventValueFromExpr(expr, lookup); got != "campaign.created" {
			t.Fatalf("got %q, want campaign.created", got)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		expr := &ast.ArrayType{Elt: &ast.Ident{Name: "string"}}
		if got := eventValueFromExpr(expr, lookup); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})
}

func TestSwitchesOnEventType(t *testing.T) {
	if switchesOnEventType(&ast.SelectorExpr{X: &ast.Ident{Name: "evt"}, Sel: &ast.Ident{Name: "Type"}}) != true {
		t.Fatal("expected selector evt.Type to match")
	}
	if switchesOnEventType(&ast.SelectorExpr{X: &ast.Ident{Name: "evt"}, Sel: &ast.Ident{Name: "ID"}}) != false {
		t.Fatal("expected selector evt.ID to not match")
	}
	if switchesOnEventType(&ast.Ident{Name: "evt"}) != false {
		t.Fatal("expected non-selector to not match")
	}
}

func TestSortedUnique(t *testing.T) {
	var nilSlice []string
	if got := sortedUnique(nilSlice); got != nil {
		t.Fatalf("expected nil for nil input, got %#v", got)
	}
	values := []string{"b", "a", "b", "c", "a"}
	got := sortedUnique(values)
	want := []string{"a", "b", "c"}
	if !bytes.Equal([]byte(strings.Join(got, ",")), []byte(strings.Join(want, ","))) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestImportedResolutionHelpers(t *testing.T) {
	root := t.TempDir()
	importDir := filepath.Join(root, "internal", "example")
	if err := os.MkdirAll(importDir, 0o755); err != nil {
		t.Fatalf("mkdir import dir: %v", err)
	}

	src := strings.Join([]string{
		"package example",
		"",
		"const ExternalType = \"external.created\"",
		"",
		"type ExternalPayload struct {",
		"\tID string `json:\"id\"`",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(importDir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write sample.go: %v", err)
	}

	importAliases := map[string]string{
		"ext": "github.com/louisbranch/fracturing.space/internal/example",
	}

	if got := resolveImportDir("github.com/louisbranch/fracturing.space/internal/example", root); got == "" {
		t.Fatal("expected resolveImportDir to resolve project import path")
	}
	if got := resolveImportDir("example.com/not-project", root); got != "" {
		t.Fatalf("expected empty dir for non-project import, got %q", got)
	}

	payloadSelector := &ast.SelectorExpr{X: &ast.Ident{Name: "ext"}, Sel: &ast.Ident{Name: "ExternalPayload"}}
	fields, definedAt, ok := parseImportedPayload(payloadSelector, importAliases, root)
	if !ok {
		t.Fatal("expected parseImportedPayload to resolve payload")
	}
	if len(fields) != 1 || fields[0].Name != "ID" || fields[0].JSONTag != `json:"id"` {
		t.Fatalf("unexpected payload fields: %#v", fields)
	}
	if !strings.Contains(definedAt, "internal/example/sample.go:5") {
		t.Fatalf("unexpected definedAt: %s", definedAt)
	}

	constSelector := &ast.SelectorExpr{X: &ast.Ident{Name: "ext"}, Sel: &ast.Ident{Name: "ExternalType"}}
	value, ok := constFromSelector(constSelector, importAliases, root)
	if !ok {
		t.Fatal("expected constFromSelector to resolve external constant")
	}
	if value != "external.created" {
		t.Fatalf("unexpected constant value: %s", value)
	}
}

func TestParsePayloadFromTypeExpr_IdentAlias(t *testing.T) {
	src := strings.Join([]string{
		"package sample",
		"",
		"type BasePayload struct {",
		"\tID string `json:\"id\"`",
		"}",
		"",
		"type AliasPayload BasePayload",
	}, "\n")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	typeSpecs := make(map[string]ast.Expr)
	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok {
			return true
		}
		typeSpecs[typeSpec.Name.Name] = typeSpec.Type
		return true
	})

	fields, _, ok := parsePayloadFromTypeExpr(
		&ast.Ident{Name: "AliasPayload"},
		fset,
		nil,
		"",
		typeSpecs,
		map[string]struct{}{"AliasPayload": {}},
	)
	if ok {
		t.Fatal("expected cycle guard to prevent resolving seen type")
	}

	fields, _, ok = parsePayloadFromTypeExpr(
		&ast.Ident{Name: "AliasPayload"},
		fset,
		nil,
		"",
		typeSpecs,
		map[string]struct{}{},
	)
	if !ok {
		t.Fatal("expected alias payload to resolve")
	}
	if len(fields) != 1 || fields[0].Name != "ID" {
		t.Fatalf("unexpected resolved fields: %#v", fields)
	}
}

func TestRenderUsageMap(t *testing.T) {
	defs := packageDefs{
		Events: []eventDef{
			{
				Owner:     "Core",
				Name:      "TypeCampaignCreated",
				Value:     "campaign.created",
				DefinedAt: "internal/foo.go:10",
			},
			{
				Owner:     "Core",
				Name:      "TypeSessionStarted",
				Value:     "session.started",
				DefinedAt: "internal/bar.go:11",
			},
		},
		Payloads: map[string]payloadDef{},
	}

	emitters := map[string][]string{
		"campaign.created": {"internal/emit.go:12"},
	}
	appliers := map[string][]string{
		"campaign.created": {"internal/projection/applier.go:22"},
		"session.started":  {"internal/projection/applier.go:24"},
	}

	output, err := renderUsageMap([]packageDefs{defs}, emitters, appliers)
	if err != nil {
		t.Fatalf("renderUsageMap returned error: %v", err)
	}

	checks := []string{
		"# Event Usage Map",
		"## Core Events",
		"### `campaign.created`",
		"`internal/emit.go:12`",
		"`internal/projection/applier.go:22`",
		"### `session.started`",
		"- Emitters: none found",
		"`internal/projection/applier.go:24`",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Fatalf("expected output to contain %q", check)
		}
	}
}

func TestRenderCommandCatalog(t *testing.T) {
	definitions := []commandDef{
		{
			Owner:         "Core",
			Value:         "action.roll.resolve",
			GateScope:     "session",
			AllowWhenOpen: false,
		},
		{
			Owner: "Core",
			Value: "campaign.create",
		},
		{
			Owner: "System",
			Value: "sys.alpha.action.attack.resolve",
		},
	}

	output, err := renderCommandCatalog(definitions)
	if err != nil {
		t.Fatalf("renderCommandCatalog returned error: %v", err)
	}

	checks := []string{
		"# Command Catalog",
		"## Core Commands",
		"| `campaign.create` | `campaign` | `none` | n/a |",
		"| `action.roll.resolve` | `action` | `session` | no |",
		"## System Commands",
		"| `sys.alpha.action.attack.resolve` | `sys` | `none` | n/a |",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Fatalf("expected output to contain %q", check)
		}
	}
}

func TestParseCommandPackage(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "internal", "services", "game", "domain", "sample")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir sample package: %v", err)
	}

	deciderSrc := strings.Join([]string{
		"package sample",
		"",
		"import \"github.com/louisbranch/fracturing.space/internal/services/game/domain/command\"",
		"",
		"const (",
		"\tcommandTypeFoo command.Type = \"sample.foo\"",
		"\tcommandTypeBar command.Type = \"sample.bar\"",
		")",
	}, "\n")
	if err := os.WriteFile(filepath.Join(pkgDir, "decider.go"), []byte(deciderSrc), 0o644); err != nil {
		t.Fatalf("write decider.go: %v", err)
	}

	registrySrc := strings.Join([]string{
		"package sample",
		"",
		"import \"github.com/louisbranch/fracturing.space/internal/services/game/domain/command\"",
		"",
		"func RegisterCommands(registry *command.Registry) error {",
		"\tif err := registry.Register(command.Definition{",
		"\t\tType:  commandTypeFoo,",
		"\t\tOwner: command.OwnerCore,",
		"\t\tGate: command.GatePolicy{",
		"\t\t\tScope:         command.GateScopeSession,",
		"\t\t\tAllowWhenOpen: true,",
		"\t\t},",
		"\t}); err != nil {",
		"\t\treturn err",
		"\t}",
		"\treturn registry.Register(command.Definition{Type: commandTypeBar, Owner: command.OwnerCore})",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(pkgDir, "registry.go"), []byte(registrySrc), 0o644); err != nil {
		t.Fatalf("write registry.go: %v", err)
	}

	definitions, err := parseCommandPackage(pkgDir, root, "Core")
	if err != nil {
		t.Fatalf("parseCommandPackage returned error: %v", err)
	}
	if len(definitions) != 2 {
		t.Fatalf("expected 2 command definitions, got %d", len(definitions))
	}

	byValue := make(map[string]commandDef, len(definitions))
	for _, definition := range definitions {
		byValue[definition.Value] = definition
	}

	foo, ok := byValue["sample.foo"]
	if !ok {
		t.Fatal("expected sample.foo command definition")
	}
	if foo.Owner != "Core" {
		t.Fatalf("sample.foo owner = %q, want Core", foo.Owner)
	}
	if foo.GateScope != "session" {
		t.Fatalf("sample.foo gate scope = %q, want session", foo.GateScope)
	}
	if !foo.AllowWhenOpen {
		t.Fatal("sample.foo expected allow_when_open=true")
	}

	bar, ok := byValue["sample.bar"]
	if !ok {
		t.Fatal("expected sample.bar command definition")
	}
	if bar.GateScope != "" {
		t.Fatalf("sample.bar gate scope = %q, want empty", bar.GateScope)
	}
}

func TestMergeCommandDefinitions(t *testing.T) {
	target := map[string]commandDef{
		"sample.foo": {
			Owner:     "Core",
			Value:     "sample.foo",
			GateScope: "none",
		},
	}

	mergeCommandDefinitions(target, []commandDef{
		{
			Owner:         "Core",
			Name:          "commandTypeFoo",
			Value:         "sample.foo",
			GateScope:     "session",
			AllowWhenOpen: true,
			DefinedAt:     "sample/registry.go:10",
		},
	})

	merged := target["sample.foo"]
	if merged.GateScope != "session" {
		t.Fatalf("merged gate scope = %q, want session", merged.GateScope)
	}
	if !merged.AllowWhenOpen {
		t.Fatal("expected merged allow_when_open=true")
	}
	if merged.Name != "commandTypeFoo" {
		t.Fatalf("merged name = %q, want commandTypeFoo", merged.Name)
	}
}

func TestCommandDefinitionsFromMap(t *testing.T) {
	defs := commandDefinitionsFromMap(map[string]commandDef{
		"zeta":  {Value: "zeta"},
		"alpha": {Value: "alpha"},
	})
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
	if defs[0].Value != "alpha" || defs[1].Value != "zeta" {
		t.Fatalf("unexpected sorted order: %#v", defs)
	}
}

func TestIsDefinitionComposite(t *testing.T) {
	if !isDefinitionComposite(&ast.SelectorExpr{X: &ast.Ident{Name: "command"}, Sel: &ast.Ident{Name: "Definition"}}) {
		t.Fatal("expected selector command.Definition to match")
	}
	if !isDefinitionComposite(&ast.Ident{Name: "Definition"}) {
		t.Fatal("expected Definition ident to match")
	}
	if isDefinitionComposite(&ast.Ident{Name: "Other"}) {
		t.Fatal("did not expect Other ident to match")
	}
}

func TestCommandOwnerFromExpr(t *testing.T) {
	if owner, ok := commandOwnerFromExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "command"}, Sel: &ast.Ident{Name: "OwnerCore"}}); !ok || owner != "Core" {
		t.Fatalf("owner = %q ok=%v, want Core/true", owner, ok)
	}
	if owner, ok := commandOwnerFromExpr(&ast.SelectorExpr{X: &ast.Ident{Name: "command"}, Sel: &ast.Ident{Name: "OwnerSystem"}}); !ok || owner != "System" {
		t.Fatalf("owner = %q ok=%v, want System/true", owner, ok)
	}
	if _, ok := commandOwnerFromExpr(&ast.Ident{Name: "OwnerCore"}); ok {
		t.Fatal("did not expect non-selector owner expression to resolve")
	}
}

func TestParseGateScope(t *testing.T) {
	if got := parseGateScope(&ast.SelectorExpr{X: &ast.Ident{Name: "command"}, Sel: &ast.Ident{Name: "GateScopeSession"}}); got != "session" {
		t.Fatalf("got %q, want session", got)
	}
	if got := parseGateScope(&ast.SelectorExpr{X: &ast.Ident{Name: "command"}, Sel: &ast.Ident{Name: "GateScopeNone"}}); got != "none" {
		t.Fatalf("got %q, want none", got)
	}
	if got := parseGateScope(&ast.BasicLit{Kind: token.STRING, Value: `"custom"`}); got != "custom" {
		t.Fatalf("got %q, want custom", got)
	}
	if got := parseGateScope(&ast.Ident{Name: "other"}); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestMainGeneratesCommandCatalogFromRepo(t *testing.T) {
	root, err := resolveRoot("")
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}

	tempDir := t.TempDir()
	catalogOut := filepath.Join(tempDir, "catalog.md")
	usageOut := filepath.Join(tempDir, "usage.md")
	commandOut := filepath.Join(tempDir, "commands.md")

	oldArgs := os.Args
	oldFlagSet := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagSet
	}()

	flag.CommandLine = flag.NewFlagSet("eventdocgen-test-repo", flag.ContinueOnError)
	os.Args = []string{
		"eventdocgen-test-repo",
		"-root", root,
		"-out", catalogOut,
		"-usage-out", usageOut,
		"-commands-out", commandOut,
	}

	main()

	commandData, err := os.ReadFile(commandOut)
	if err != nil {
		t.Fatalf("read generated command catalog: %v", err)
	}
	if !strings.Contains(string(commandData), "# Command Catalog") {
		t.Fatalf("command catalog missing title:\n%s", string(commandData))
	}
}

func TestScanEmitterValues(t *testing.T) {
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
		"\t_ = event.Event{Type: TypeFoo}",
		"\t_ = event.Event{Type: event.Type(\"session.started\")}",
		"}",
	}, "\n")

	path := filepath.Join(dir, "emit.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write emit.go: %v", err)
	}

	lookup := map[string]string{
		"TypeFoo": "campaign.created",
	}
	emitters, err := scanEmitterValues(root, root, lookup)
	if err != nil {
		t.Fatalf("scanEmitterValues returned error: %v", err)
	}

	if len(emitters["campaign.created"]) != 1 {
		t.Fatalf("expected one emitter for campaign.created, got %d", len(emitters["campaign.created"]))
	}
	if len(emitters["session.started"]) != 1 {
		t.Fatalf("expected one emitter for session.started, got %d", len(emitters["session.started"]))
	}
}

func TestScanAppliers(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "applier")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir applier: %v", err)
	}

	src := strings.Join([]string{
		"package sample",
		"",
		"import \"example.com/event\"",
		"",
		"func Apply(evt event.Event) error {",
		"\tswitch evt.Type {",
		"\tcase event.Type(\"campaign.created\"):",
		"\t\treturn nil",
		"\tcase EventTypeSessionStarted:",
		"\t\treturn nil",
		"\tdefault:",
		"\t\treturn nil",
		"\t}",
		"}",
	}, "\n")

	path := filepath.Join(dir, "apply.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write apply.go: %v", err)
	}

	lookup := map[string]string{
		"EventTypeSessionStarted": "session.started",
	}
	appliers, err := scanAppliers(root, root, lookup)
	if err != nil {
		t.Fatalf("scanAppliers returned error: %v", err)
	}

	if len(appliers["campaign.created"]) != 1 {
		t.Fatalf("expected one applier for campaign.created, got %d", len(appliers["campaign.created"]))
	}
	if len(appliers["session.started"]) != 1 {
		t.Fatalf("expected one applier for session.started, got %d", len(appliers["session.started"]))
	}
}

// TestScanAppliers_QualifiedSelector verifies that package-qualified selectors
// (e.g. campaign.EventTypeCreated vs character.EventTypeCreated) resolve to
// distinct event values even when the constant names collide.
func TestScanAppliers_QualifiedSelector(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Create two packages that export the same constant name with different values.
	pkgA := filepath.Join(root, "pkga")
	pkgB := filepath.Join(root, "pkgb")
	if err := os.MkdirAll(pkgA, 0o755); err != nil {
		t.Fatalf("mkdir pkga: %v", err)
	}
	if err := os.MkdirAll(pkgB, 0o755); err != nil {
		t.Fatalf("mkdir pkgb: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgA, "types.go"), []byte(strings.Join([]string{
		"package pkga",
		"",
		"type Type string",
		"const EventTypeCreated Type = \"a.created\"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("write pkga: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgB, "types.go"), []byte(strings.Join([]string{
		"package pkgb",
		"",
		"type Type string",
		"const EventTypeCreated Type = \"b.created\"",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("write pkgb: %v", err)
	}

	// Create an applier file that uses both qualified selectors.
	applierDir := filepath.Join(root, "applier")
	if err := os.MkdirAll(applierDir, 0o755); err != nil {
		t.Fatalf("mkdir applier: %v", err)
	}
	applierSrc := strings.Join([]string{
		"package applier",
		"",
		"import (",
		"\t\"example.com/test/pkga\"",
		"\t\"example.com/test/pkgb\"",
		")",
		"",
		"type Event struct{ Type string }",
		"",
		"func Apply(evt Event) error {",
		"\tswitch evt.Type {",
		"\tcase pkga.EventTypeCreated:",
		"\t\treturn nil",
		"\tcase pkgb.EventTypeCreated:",
		"\t\treturn nil",
		"\tdefault:",
		"\t\treturn nil",
		"\t}",
		"}",
	}, "\n")
	if err := os.WriteFile(filepath.Join(applierDir, "apply.go"), []byte(applierSrc), 0o644); err != nil {
		t.Fatalf("write apply.go: %v", err)
	}

	// The global lookup has both constants with the same unqualified name.
	// Last write wins, so one value is lost — this is the bug.
	lookup := map[string]string{
		"EventTypeCreated": "b.created", // collision: only one value survives
	}

	appliers, err := scanAppliers(applierDir, root, lookup)
	if err != nil {
		t.Fatalf("scanAppliers: %v", err)
	}

	if len(appliers["a.created"]) != 1 {
		t.Fatalf("expected 1 applier for a.created, got %d (values: %v)", len(appliers["a.created"]), appliers["a.created"])
	}
	if len(appliers["b.created"]) != 1 {
		t.Fatalf("expected 1 applier for b.created, got %d", len(appliers["b.created"]))
	}
}
