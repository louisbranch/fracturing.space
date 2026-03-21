package referencecorpus

import (
	"fmt"
	"strings"
)

const supportedSystem = "daggerheart"

func validateSystem(system string) error {
	system = strings.TrimSpace(system)
	if system == "" || strings.EqualFold(system, supportedSystem) {
		return nil
	}
	return fmt.Errorf("system %q is not supported", system)
}

func metadataScore(entry indexEntry, queryLower string, queryTerms []string) int {
	score := 0
	idLower := strings.ToLower(entry.ID)
	titleLower := strings.ToLower(entry.Title)
	kindLower := strings.ToLower(entry.Kind)
	pathLower := strings.ToLower(entry.Path)
	switch {
	case titleLower == queryLower:
		score += 120
	case strings.Contains(titleLower, queryLower):
		score += 100
	}
	if idLower == queryLower {
		score += 110
	} else if strings.Contains(idLower, queryLower) {
		score += 90
	}
	if kindLower == queryLower {
		score += 60
	} else if strings.Contains(kindLower, queryLower) {
		score += 40
	}
	if strings.Contains(pathLower, queryLower) {
		score += 35
	}
	for _, alias := range entry.Aliases {
		aliasLower := strings.ToLower(alias)
		if aliasLower == queryLower {
			score += 105
			continue
		}
		if strings.Contains(aliasLower, queryLower) {
			score += 80
		}
	}
	if score == 0 {
		return 0
	}
	for _, term := range queryTerms {
		if term == "" {
			continue
		}
		if strings.Contains(titleLower, term) || strings.Contains(idLower, term) || strings.Contains(kindLower, term) || strings.Contains(pathLower, term) {
			score += 10
			continue
		}
		for _, alias := range entry.Aliases {
			if strings.Contains(strings.ToLower(alias), term) {
				score += 10
				break
			}
		}
	}
	return score
}

func metadataSnippet(entry indexEntry) string {
	parts := make([]string, 0, 3)
	if entry.Title != "" {
		parts = append(parts, entry.Title)
	}
	if entry.Kind != "" {
		parts = append(parts, "kind: "+entry.Kind)
	}
	if len(entry.Aliases) != 0 {
		parts = append(parts, "aliases: "+strings.Join(entry.Aliases, ", "))
	}
	return strings.Join(parts, " | ")
}

func containsAllTerms(text string, terms []string) bool {
	for _, term := range terms {
		if term == "" {
			continue
		}
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func contentSnippet(content, queryLower string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	lower := strings.ToLower(content)
	idx := strings.Index(lower, queryLower)
	if idx < 0 {
		const maxLen = 160
		if len(content) <= maxLen {
			return content
		}
		return content[:maxLen] + "..."
	}
	const radius = 72
	start := max(idx-radius, 0)
	end := min(idx+len(queryLower)+radius, len(content))
	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet += "..."
	}
	return snippet
}
