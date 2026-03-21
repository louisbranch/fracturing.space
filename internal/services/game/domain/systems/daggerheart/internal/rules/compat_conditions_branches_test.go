package rules

import "testing"

func TestNormalizeConditions_Branches(t *testing.T) {
	if got, err := NormalizeConditions(nil); err != nil || len(got) != 0 {
		t.Fatalf("NormalizeConditions(nil) = (%v, %v), want ([], nil)", got, err)
	}
	if _, err := NormalizeConditions([]string{" "}); err == nil {
		t.Fatal("expected empty condition error")
	}
	if _, err := NormalizeConditions([]string{"unknown"}); err == nil {
		t.Fatal("expected unsupported condition error")
	}
	got, err := NormalizeConditions([]string{" Vulnerable ", "hidden", "hidden"})
	if err != nil {
		t.Fatalf("NormalizeConditions(valid) error = %v", err)
	}
	if len(got) != 2 || got[0] != ConditionHidden || got[1] != ConditionVulnerable {
		t.Fatalf("NormalizeConditions(valid) = %v, want [hidden vulnerable]", got)
	}
}

func TestDiffConditionsAndConditionsEqual_Branches(t *testing.T) {
	added, removed := DiffConditions([]string{ConditionHidden}, []string{ConditionHidden, ConditionRestrained})
	if len(added) != 1 || added[0] != ConditionRestrained {
		t.Fatalf("added = %v, want [restrained]", added)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %v, want []", removed)
	}
	if ConditionsEqual([]string{ConditionHidden}, []string{ConditionHidden, ConditionRestrained}) {
		t.Fatal("expected ConditionsEqual length mismatch to be false")
	}
	if ConditionsEqual([]string{ConditionHidden}, []string{ConditionRestrained}) {
		t.Fatal("expected ConditionsEqual value mismatch to be false")
	}
	if !ConditionsEqual([]string{ConditionHidden}, []string{ConditionHidden}) {
		t.Fatal("expected equal condition sets to be true")
	}
}
