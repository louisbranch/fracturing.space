package daggerheart

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func loadSourceFile(t *testing.T, filename string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	sourcePath := filepath.Join(filepath.Dir(thisFile), filename)
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	return string(sourceBytes)
}

func assertNoBypassPatterns(t *testing.T, source, filename string, patterns []string) {
	t.Helper()
	for _, pattern := range patterns {
		if strings.Contains(source, pattern) {
			t.Fatalf("%s must not contain bypass pattern %q", filename, pattern)
		}
	}
}

func TestAdversaryHandlersUseSharedDomainWriteHelper(t *testing.T) {
	source := loadSourceFile(t, "adversaries.go")
	if strings.Contains(source, "s.stores.Domain.Execute(") {
		t.Fatalf("adversaries.go must use shared domain write helper instead of direct Domain.Execute")
	}
}

func TestActionHandlersUseSharedDomainWriteHelper(t *testing.T) {
	source := loadSourceFile(t, "actions.go")
	if strings.Contains(source, "s.stores.Domain.Execute(") {
		t.Fatalf("actions.go must use shared domain write helper instead of direct Domain.Execute")
	}
}

func TestAdversaryHandlersDoNotInlineApplyEvents(t *testing.T) {
	source := loadSourceFile(t, "adversaries.go")
	if strings.Contains(source, ".Apply(ctx, evt)") {
		t.Fatalf("adversaries.go must route emitted-event apply through shared helper")
	}
}

func TestActionHandlersDoNotInlineApplyEvents(t *testing.T) {
	source := loadSourceFile(t, "actions.go")
	if strings.Contains(source, ".Apply(ctx, evt)") {
		t.Fatalf("actions.go must route emitted-event apply through shared helper")
	}
}

func TestAdversaryHandlersNoDirectStorageMutationBypass(t *testing.T) {
	source := loadSourceFile(t, "adversaries.go")
	assertNoBypassPatterns(t, source, "adversaries.go", []string{
		"s.stores.Event.AppendEvent(",
		"PutDaggerheart",
		"UpdateDaggerheart",
		"DeleteDaggerheart",
	})
}

func TestActionHandlersNoDirectStorageMutationBypass(t *testing.T) {
	source := loadSourceFile(t, "actions.go")
	assertNoBypassPatterns(t, source, "actions.go", []string{
		"s.stores.Event.AppendEvent(",
		"s.stores.Outcome.ApplyRollOutcome(",
		"PutDaggerheart",
		"UpdateDaggerheart",
		"DeleteDaggerheart",
	})
}
