package instructionset

import (
	"os"
	"strings"
	"testing"
)

func TestLoaderLoadsCoreSkillsFromEmbed(t *testing.T) {
	loader := New("")
	content, err := loader.LoadCoreSkills()
	if err != nil {
		t.Fatalf("LoadCoreSkills() error = %v", err)
	}
	if !strings.Contains(content, "GM Operating Contract") {
		t.Fatalf("expected core skills content, got: %q", content[:min(len(content), 200)])
	}
}

func TestLoaderLoadsCoreInteractionFromEmbed(t *testing.T) {
	loader := New("")
	content, err := loader.LoadCoreInteraction()
	if err != nil {
		t.Fatalf("LoadCoreInteraction() error = %v", err)
	}
	if !strings.Contains(content, "Interaction Contract") {
		t.Fatalf("expected interaction content, got: %q", content[:min(len(content), 200)])
	}
}

func TestLoaderLoadsSystemSkillsFromEmbed(t *testing.T) {
	loader := New("")
	content, err := loader.LoadSystemSkills("daggerheart")
	if err != nil {
		t.Fatalf("LoadSystemSkills() error = %v", err)
	}
	if !strings.Contains(content, "Daggerheart") {
		t.Fatalf("expected daggerheart skills content, got: %q", content[:min(len(content), 200)])
	}
}

func TestLoaderReturnsEmptyForMissingSystem(t *testing.T) {
	loader := New("")
	content, err := loader.LoadSystemSkills("nonexistent")
	if err != nil {
		t.Fatalf("LoadSystemSkills() error = %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty content for missing system, got: %q", content[:min(len(content), 200)])
	}
}

func TestLoaderLoadSkillsComposesCoreAndSystem(t *testing.T) {
	loader := New("")
	content, err := loader.LoadSkills("daggerheart")
	if err != nil {
		t.Fatalf("LoadSkills() error = %v", err)
	}
	if !strings.Contains(content, "GM Operating Contract") {
		t.Fatalf("missing core skills in composed output")
	}
	if !strings.Contains(content, "Daggerheart GM Guidance") {
		t.Fatalf("missing system skills in composed output")
	}
	if !strings.Contains(content, "Memory Management") {
		t.Fatalf("missing memory guide in composed output")
	}
	if !strings.Contains(content, "Daggerheart Reference Lookup") {
		t.Fatalf("missing reference guide in composed output")
	}
}

func TestLoaderLoadSkillsWorksWithoutSystem(t *testing.T) {
	loader := New("")
	content, err := loader.LoadSkills("")
	if err != nil {
		t.Fatalf("LoadSkills() error = %v", err)
	}
	if !strings.Contains(content, "GM Operating Contract") {
		t.Fatalf("missing core skills in composed output")
	}
	if !strings.Contains(content, "Memory Management") {
		t.Fatalf("missing memory guide in composed output")
	}
}

func TestLoaderFilesystemOverride(t *testing.T) {
	dir := t.TempDir()
	coreDir := dir + "/v1/core"
	if err := makeDir(coreDir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeFile(coreDir+"/skills.md", "# Custom Skills"); err != nil {
		t.Fatalf("write: %v", err)
	}

	loader := New(dir)
	content, err := loader.LoadCoreSkills()
	if err != nil {
		t.Fatalf("LoadCoreSkills() error = %v", err)
	}
	if !strings.Contains(content, "Custom Skills") {
		t.Fatalf("expected custom content, got: %q", content)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func makeDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
