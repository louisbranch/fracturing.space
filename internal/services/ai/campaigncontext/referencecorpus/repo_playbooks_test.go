package referencecorpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type playbookLessonManifestEntry struct {
	LessonID      string   `json:"lesson_id"`
	Title         string   `json:"title"`
	PlaybookID    string   `json:"playbook_id"`
	PlaybookPath  string   `json:"playbook_path"`
	RequiredTools []string `json:"required_tools"`
	Scenarios     []string `json:"scenarios"`
}

func TestRepoPlaybookManifestMatchesFilesScenariosAndTools(t *testing.T) {
	playbookDir := repoRootPath(t, "docs/reference/daggerheart-playbooks")
	manifestPath := filepath.Join(playbookDir, "scenario-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var entries []playbookLessonManifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one lesson entry")
	}

	toolNames := declaredToolNames()
	for _, entry := range entries {
		if entry.LessonID == "" || entry.PlaybookID == "" || entry.PlaybookPath == "" {
			t.Fatalf("manifest entry is missing identifiers: %+v", entry)
		}
		if len(entry.RequiredTools) == 0 {
			t.Fatalf("manifest entry %q must declare required tools", entry.LessonID)
		}
		if len(entry.Scenarios) == 0 {
			t.Fatalf("manifest entry %q must declare scenarios", entry.LessonID)
		}
		playbookPath := repoRootPath(t, entry.PlaybookPath)
		if _, err := os.Stat(playbookPath); err != nil {
			t.Fatalf("playbook path %q: %v", playbookPath, err)
		}
		for _, scenarioPath := range entry.Scenarios {
			fullScenarioPath := repoRootPath(t, scenarioPath)
			if _, err := os.Stat(fullScenarioPath); err != nil {
				t.Fatalf("scenario path %q: %v", fullScenarioPath, err)
			}
		}
		for _, toolName := range entry.RequiredTools {
			if _, ok := toolNames[toolName]; !ok {
				t.Fatalf("manifest entry %q references unknown tool %q", entry.LessonID, toolName)
			}
		}
	}
}

func TestLoadRepoPlaybookEntriesFromDirIgnoresIndexGuide(t *testing.T) {
	dir := t.TempDir()
	writeRepoPlaybookFile(t, dir, "index.md", `---
title: "Daggerheart playbooks"
parent: "Reference"
nav_order: 19
has_children: true
---

# Daggerheart playbooks
`)
	writeRepoPlaybookFile(t, dir, "playbook-example.md", `---
id: "playbook-example"
title: "Example Playbook"
kind: "playbook"
aliases: ["example"]
---

# Example Playbook
`)

	entries, err := loadRepoPlaybookEntriesFromDir(dir)
	if err != nil {
		t.Fatalf("loadRepoPlaybookEntriesFromDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("loadRepoPlaybookEntriesFromDir() len = %d, want 1", len(entries))
	}
	if entries[0].ID != "playbook-example" {
		t.Fatalf("loadRepoPlaybookEntriesFromDir() id = %q, want %q", entries[0].ID, "playbook-example")
	}
}

func TestLoadRepoPlaybookEntriesFromDirRejectsBrokenPlaybookFrontMatter(t *testing.T) {
	dir := t.TempDir()
	writeRepoPlaybookFile(t, dir, "playbook-broken.md", `---
title: "Broken Playbook"
kind: "playbook"
---

# Broken Playbook
`)

	_, err := loadRepoPlaybookEntriesFromDir(dir)
	if err == nil || !strings.Contains(err.Error(), "missing id, title, or kind front matter") {
		t.Fatalf("loadRepoPlaybookEntriesFromDir() error = %v, want invalid front matter error", err)
	}
}

func repoRootPath(t *testing.T, relative string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "..", filepath.FromSlash(relative)))
}

func declaredToolNames() map[string]struct{} {
	return map[string]struct{}{
		"character_sheet_read":                        {},
		"daggerheart_action_roll_resolve":             {},
		"daggerheart_combat_board_read":               {},
		"daggerheart_gm_move_apply":                   {},
		"daggerheart_adversary_create":                {},
		"daggerheart_adversary_update":                {},
		"daggerheart_scene_countdown_create":          {},
		"daggerheart_scene_countdown_advance":         {},
		"daggerheart_scene_countdown_resolve_trigger": {},
		"daggerheart_attack_flow_resolve":             {},
		"daggerheart_adversary_attack_flow_resolve":   {},
		"daggerheart_reaction_flow_resolve":           {},
		"daggerheart_group_action_flow_resolve":       {},
		"daggerheart_tag_team_flow_resolve":           {},
		"system_reference_search":                     {},
		"system_reference_read":                       {},
	}
}

func writeRepoPlaybookFile(t *testing.T, dir, name, contents string) {
	t.Helper()

	fullPath := filepath.Join(dir, name)
	if err := os.WriteFile(fullPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write repo playbook %q: %v", fullPath, err)
	}
}
