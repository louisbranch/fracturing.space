package aieval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPromptContextBaselineEmbedded(t *testing.T) {
	ctx := BuildPromptContext(string(PromptProfileBaseline), "")
	if ctx.Profile != string(PromptProfileBaseline) {
		t.Fatalf("profile = %q, want %q", ctx.Profile, PromptProfileBaseline)
	}
	if ctx.InstructionsSource != instructionsSourceEmbedded {
		t.Fatalf("instructions source = %q, want %q", ctx.InstructionsSource, instructionsSourceEmbedded)
	}
	if ctx.Summary == "" {
		t.Fatal("expected summary")
	}
	if ctx.InstructionsDigest == "" {
		t.Fatal("expected instructions digest")
	}
}

func TestBuildPromptContextMechanicsOverride(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel string, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	mustWrite("v1/core/skills.md", "# Core Skills")
	mustWrite("v1/core/interaction.md", "# Interaction")
	mustWrite("v1/core/memory-guide.md", "# Memory")
	mustWrite("v1/daggerheart/skills.md", "# System Skills")
	mustWrite("v1/daggerheart/reference-guide.md", "# Reference")

	ctx := BuildPromptContext(string(PromptProfileMechanicsHardened), root)
	if ctx.InstructionsSource != instructionsSourceFS {
		t.Fatalf("instructions source = %q, want %q", ctx.InstructionsSource, instructionsSourceFS)
	}
	if ctx.InstructionsRoot == "" {
		t.Fatal("expected absolute instructions root")
	}
	if ctx.Summary == "" {
		t.Fatal("expected summary")
	}
	if ctx.InstructionsDigest == "" {
		t.Fatal("expected instructions digest")
	}
}
