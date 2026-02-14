package domain

import "testing"

func TestRulesVersionMetadata(t *testing.T) {
	metadata := RulesVersion()
	if metadata.System == "" {
		t.Fatal("expected system name")
	}
	if metadata.Module == "" {
		t.Fatal("expected module name")
	}
	if metadata.RulesVersion == "" {
		t.Fatal("expected rules version")
	}
	if metadata.CritRule != "critical success on matching hope/fear; always succeeds" {
		t.Fatalf("CritRule = %q", metadata.CritRule)
	}
	if metadata.DifficultyRule != "difficulty optional; total >= difficulty succeeds; critical success always succeeds" {
		t.Fatalf("DifficultyRule = %q", metadata.DifficultyRule)
	}
}
