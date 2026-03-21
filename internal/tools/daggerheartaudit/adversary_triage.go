package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// adversaryFeatureClass describes how one parsed adversary feature maps to
// the runtime classification for audit purposes.
type adversaryFeatureClass struct {
	Name     string // parsed feature name (e.g. "Momentum")
	Kind     string // parsed kind (e.g. "passive", "action", "reaction")
	Status   string // "runtime_supported", "recurring_rule", "narrative"
	RuleKind string // if runtime_supported, the matched rule kind
}

// adversaryEntryClass captures the aggregate classification of one adversary
// entry based on its parsed features.
type adversaryEntryClass struct {
	Features       []adversaryFeatureClass
	SupportedCount int
	RecurringCount int
	NarrativeCount int
}

// allCovered returns true when every feature is either runtime-supported,
// a recurring rule, or classified as narrative-only.
func (c adversaryEntryClass) allCovered() bool {
	return len(c.Features) == c.SupportedCount+c.RecurringCount+c.NarrativeCount
}

// hasRuntimeSupported returns true when at least one feature resolves to a
// typed runtime rule kind.
func (c adversaryEntryClass) hasRuntimeSupported() bool {
	return c.SupportedCount > 0
}

// supportedAdversaryFeatureNames maps lowercase feature names to their
// corresponding runtime rule kinds, mirroring the recognition logic in
// rules.ResolveAdversaryFeatureRuntime.
var supportedAdversaryFeatureNames = map[string]string{
	"momentum":       "momentum_gain_fear_on_successful_attack",
	"terrifying":     "terrifying_hope_loss_on_successful_attack",
	"group attack":   "group_attack",
	"cloaked":        "hidden_until_next_attack",
	"backstab":       "damage_replacement_on_advantaged_attack",
	"pack tactics":   "conditional_damage_replacement_with_contributor",
	"flying":         "difficulty_bonus_while_active",
	"warding sphere": "retaliatory_damage_on_close_hit",
	"box in":         "focus_target_disadvantage",
}

// recurringRuleNames are features that resolve to dedicated adversary
// recurring rules (Relentless, Minion, Horde) which are already modeled as
// separate rule structs rather than feature automations.
var recurringRuleNames = map[string]bool{
	"relentless": true,
	"minion":     true,
	"horde":      true,
}

var reFeatureHeading = regexp.MustCompile(`^###\s+(.+?)\s*$`)
var reFeatureNameKind = regexp.MustCompile(`^(.+?)\s+-\s+(.+)$`)
var reParenSuffix = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// parsedFeature captures the name, kind, and description of one adversary
// feature extracted from the reference corpus markdown.
type parsedFeature struct {
	Name        string
	Kind        string
	Description string
}

// parseAdversaryFeatures extracts feature entries from the ## Feature section
// of an adversary reference corpus markdown file.
func parseAdversaryFeatures(content string) []parsedFeature {
	lines := strings.Split(content, "\n")
	var inFeatureSection bool
	var features []parsedFeature
	var current *parsedFeature
	var descBuf strings.Builder

	flush := func() {
		if current != nil {
			current.Description = strings.TrimSpace(descBuf.String())
			features = append(features, *current)
			current = nil
			descBuf.Reset()
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect ## Feature section start.
		if strings.HasPrefix(trimmed, "## ") {
			if strings.EqualFold(trimmed, "## Feature") || strings.EqualFold(trimmed, "## Features") {
				inFeatureSection = true
				continue
			}
			// Another ## section ends the feature section.
			if inFeatureSection {
				flush()
				break
			}
			continue
		}

		if !inFeatureSection {
			continue
		}

		// Parse ### headings within the feature section.
		if m := reFeatureHeading.FindStringSubmatch(line); m != nil {
			flush()
			raw := strings.TrimSpace(m[1])
			name, kind := parseFeatureNameKind(raw)
			current = &parsedFeature{Name: name, Kind: kind}
			continue
		}

		if current != nil {
			descBuf.WriteString(line)
			descBuf.WriteString("\n")
		}
	}
	flush()
	return features
}

// parseFeatureNameKind splits "Name (N) - Kind" into the clean feature name
// and normalized kind.
func parseFeatureNameKind(raw string) (string, string) {
	var name, kind string
	if m := reFeatureNameKind.FindStringSubmatch(raw); m != nil {
		name = strings.TrimSpace(m[1])
		kind = strings.TrimSpace(m[2])
	} else {
		name = raw
		kind = ""
	}
	// Strip parenthetical suffixes like "(3)" from the name.
	name = reParenSuffix.ReplaceAllString(name, "")
	name = strings.TrimSpace(name)
	// Normalize the kind to the base type.
	kind = normalizeFeatureKind(kind)
	return name, kind
}

// normalizeFeatureKind collapses reaction countdown variants and other
// suffixed kinds into their base classification.
func normalizeFeatureKind(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(lower, "reaction") {
		return "reaction"
	}
	if strings.HasPrefix(lower, "action") {
		return "action"
	}
	if lower == "passive" {
		return "passive"
	}
	return lower
}

// classifyFeature determines whether a parsed feature maps to a supported
// runtime rule kind, a recurring rule, or narrative-only GM guidance.
func classifyFeature(f parsedFeature) adversaryFeatureClass {
	nameLower := strings.ToLower(f.Name)

	// Check supported runtime feature names.
	if ruleKind, ok := supportedAdversaryFeatureNames[nameLower]; ok {
		return adversaryFeatureClass{
			Name:     f.Name,
			Kind:     f.Kind,
			Status:   "runtime_supported",
			RuleKind: ruleKind,
		}
	}

	// Check description for armor-shred pattern.
	if strings.Contains(strings.ToLower(f.Description), "mark an armor slot") {
		return adversaryFeatureClass{
			Name:     f.Name,
			Kind:     f.Kind,
			Status:   "runtime_supported",
			RuleKind: "armor_shred_on_successful_attack",
		}
	}

	// Check recurring rule names.
	if recurringRuleNames[nameLower] {
		return adversaryFeatureClass{
			Name:   f.Name,
			Kind:   f.Kind,
			Status: "recurring_rule",
		}
	}

	// All remaining features are GM-narrated: actions, reactions, and passive
	// features that describe GM guidance rather than automated mechanics.
	return adversaryFeatureClass{
		Name:   f.Name,
		Kind:   f.Kind,
		Status: "narrative",
	}
}

// classifyAdversaryEntries parses adversary features from their reference
// corpus markdown files and classifies each entry for audit purposes.
func classifyAdversaryEntries(referenceRoot string, entries []corpusIndexEntry) (map[string]adversaryEntryClass, error) {
	result := make(map[string]adversaryEntryClass)
	for _, entry := range entries {
		if entry.Kind != "adversary" {
			continue
		}
		path := filepath.Join(referenceRoot, filepath.FromSlash(entry.Path))
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		features := parseAdversaryFeatures(string(content))
		cls := adversaryEntryClass{
			Features: make([]adversaryFeatureClass, 0, len(features)),
		}
		for _, f := range features {
			fc := classifyFeature(f)
			cls.Features = append(cls.Features, fc)
			switch fc.Status {
			case "runtime_supported":
				cls.SupportedCount++
			case "recurring_rule":
				cls.RecurringCount++
			case "narrative":
				cls.NarrativeCount++
			}
		}
		result[entry.ID] = cls
	}
	return result, nil
}
