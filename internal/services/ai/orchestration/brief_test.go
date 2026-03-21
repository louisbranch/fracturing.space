package orchestration

import (
	"strings"
	"testing"
)

func TestBriefAssemblerRendersInPriorityOrder(t *testing.T) {
	sections := []BriefSection{
		{ID: "campaign", Priority: 200, Label: "Campaign", Content: "Ashes of the Empire"},
		{ID: "skills", Priority: 100, Label: "Skills", Content: "GM operating contract"},
		{ID: "memory", Priority: 400, Label: "memory.md", Content: "Session memory"},
	}

	got := BriefAssembler{}.Assemble(sections)
	skillsIdx := strings.Index(got, "GM operating contract")
	campaignIdx := strings.Index(got, "Ashes of the Empire")
	memoryIdx := strings.Index(got, "Session memory")

	if skillsIdx < 0 || campaignIdx < 0 || memoryIdx < 0 {
		t.Fatalf("missing section in output:\n%s", got)
	}
	if skillsIdx > campaignIdx || campaignIdx > memoryIdx {
		t.Fatalf("sections not in priority order:\n%s", got)
	}
}

func TestBriefAssemblerDropsLowPrioritySectionsOverBudget(t *testing.T) {
	sections := []BriefSection{
		{ID: "skills", Priority: 100, Label: "Skills", Content: "GM contract", Required: true},
		{ID: "campaign", Priority: 200, Label: "Campaign", Content: "Ashes of the Empire"},
		{ID: "memory", Priority: 400, Label: "memory.md", Content: strings.Repeat("x", 1000)},
	}

	// Budget enough for skills + campaign but not memory.
	got := BriefAssembler{MaxTokens: 20}.Assemble(sections)
	if !strings.Contains(got, "GM contract") {
		t.Fatalf("required section dropped:\n%s", got)
	}
	if !strings.Contains(got, "Ashes of the Empire") {
		t.Fatalf("important section dropped:\n%s", got)
	}
	if strings.Contains(got, strings.Repeat("x", 100)) {
		t.Fatalf("low priority section should have been dropped:\n%s", got)
	}
}

func TestBriefAssemblerKeepsRequiredSectionsEvenOverBudget(t *testing.T) {
	sections := []BriefSection{
		{ID: "skills", Priority: 100, Label: "Skills", Content: strings.Repeat("a", 200), Required: true},
		{ID: "campaign", Priority: 200, Label: "Campaign", Content: "Ashes"},
	}

	// Budget is tiny but required section must survive.
	got := BriefAssembler{MaxTokens: 1}.Assemble(sections)
	if !strings.Contains(got, strings.Repeat("a", 200)) {
		t.Fatalf("required section was dropped:\n%s", got)
	}
}

func TestBriefAssemblerSkipsEmptyContent(t *testing.T) {
	sections := []BriefSection{
		{ID: "skills", Priority: 100, Label: "Skills", Content: "GM contract"},
		{ID: "empty", Priority: 200, Label: "Empty", Content: "   "},
		{ID: "campaign", Priority: 300, Label: "Campaign", Content: "Ashes"},
	}

	got := BriefAssembler{}.Assemble(sections)
	if strings.Contains(got, "Empty:") {
		t.Fatalf("empty section should have been skipped:\n%s", got)
	}
}

func TestBriefAssemblerHandlesEmptyInput(t *testing.T) {
	got := BriefAssembler{}.Assemble(nil)
	if got != "" {
		t.Fatalf("expected empty output, got: %q", got)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abcd", 1},
		{"abcde", 2},
		{"abcdefgh", 2},
		{strings.Repeat("x", 100), 25},
	}
	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.want {
			t.Errorf("estimateTokens(%d chars) = %d, want %d", len(tt.input), got, tt.want)
		}
	}
}
