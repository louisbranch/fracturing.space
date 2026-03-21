package orchestration

import (
	"sort"
	"strings"
)

// BriefSection is one composable piece of the prompt.
type BriefSection struct {
	ID       string // e.g. "characters", "daggerheart_dice_rules"
	Priority int    // 100=critical, 200=important, 300=contextual, 400=supplemental
	Label    string // Section heading rendered in the prompt
	Content  string
	Required bool // Never dropped even if over budget
}

// BriefAssembler composes BriefSections into a single prompt string, respecting
// a token budget. Lower-priority sections are dropped first when the assembled
// brief exceeds the budget.
type BriefAssembler struct {
	// MaxTokens is the approximate token budget. Zero means unlimited.
	MaxTokens int
}

// charsPerToken is the byte heuristic for token estimation.
const charsPerToken = 4

// Assemble sorts sections by priority, applies the token budget, and renders
// the surviving sections into a single prompt string.
func (a BriefAssembler) Assemble(sections []BriefSection) string {
	if len(sections) == 0 {
		return ""
	}

	// Stable sort by priority (lower number = higher priority).
	sorted := make([]BriefSection, len(sections))
	copy(sorted, sections)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	// Apply token budget: keep required sections always, drop lowest-priority
	// non-required sections first when over budget.
	if a.MaxTokens > 0 {
		sorted = budgetSections(sorted, a.MaxTokens)
	}

	var b strings.Builder
	for i, s := range sorted {
		content := strings.TrimSpace(s.Content)
		if content == "" {
			continue
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		if label := strings.TrimSpace(s.Label); label != "" {
			b.WriteString(label)
			b.WriteString(":\n")
		}
		b.WriteString(content)
	}
	return b.String()
}

// estimateTokens returns a rough token count using the chars/token heuristic.
func estimateTokens(text string) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	return (n + charsPerToken - 1) / charsPerToken
}

// budgetSections keeps required sections unconditionally and fills remaining
// budget with non-required sections in priority order. Sections that would push
// the total over budget are dropped.
func budgetSections(sorted []BriefSection, maxTokens int) []BriefSection {
	// First pass: account for required sections.
	usedTokens := 0
	for _, s := range sorted {
		if s.Required {
			usedTokens += estimateTokens(s.Content)
		}
	}

	// Second pass: include non-required sections in priority order.
	kept := make([]BriefSection, 0, len(sorted))
	for _, s := range sorted {
		if s.Required {
			kept = append(kept, s)
			continue
		}
		cost := estimateTokens(s.Content)
		if usedTokens+cost > maxTokens {
			continue
		}
		usedTokens += cost
		kept = append(kept, s)
	}

	// Re-sort to preserve original priority order for rendering.
	sort.SliceStable(kept, func(i, j int) bool {
		return kept[i].Priority < kept[j].Priority
	})
	return kept
}
