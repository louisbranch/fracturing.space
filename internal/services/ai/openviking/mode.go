package openviking

import (
	"fmt"
	"strings"
)

// IntegrationMode controls how the AI service uses OpenViking during prompt
// assembly and post-turn session sync.
type IntegrationMode string

const (
	// ModeLegacy preserves the initial OpenViking pilot behavior.
	ModeLegacy IntegrationMode = "legacy"
	// ModeDocsAlignedSupplement shifts story retrieval toward resources and
	// session-derived memory while preserving local prompt-time memory.md.
	ModeDocsAlignedSupplement IntegrationMode = "docs_aligned_supplement"
)

// ParseIntegrationMode normalizes the configured OpenViking mode.
func ParseIntegrationMode(raw string) (IntegrationMode, error) {
	mode := IntegrationMode(strings.TrimSpace(raw))
	switch mode {
	case "", ModeLegacy:
		return ModeLegacy, nil
	case ModeDocsAlignedSupplement:
		return ModeDocsAlignedSupplement, nil
	default:
		return "", fmt.Errorf("unsupported openviking mode %q", raw)
	}
}

// SuppressStoryPrompt reports whether raw story.md should be omitted from the
// prompt's core context sources.
func (m IntegrationMode) SuppressStoryPrompt() bool {
	return m == ModeLegacy || m == ModeDocsAlignedSupplement
}

// SuppressMemoryPrompt reports whether raw memory.md should be omitted from the
// prompt's core context sources.
func (m IntegrationMode) SuppressMemoryPrompt() bool {
	return m == ModeLegacy
}

// MirrorsStory reports whether story.md should be mirrored into OpenViking
// resources for prompt retrieval.
func (m IntegrationMode) MirrorsStory() bool {
	return m == ModeLegacy || m == ModeDocsAlignedSupplement
}

// MirrorsMemory reports whether memory.md should be mirrored into OpenViking
// resources for prompt retrieval.
func (m IntegrationMode) MirrorsMemory() bool {
	return m == ModeLegacy
}

// UsesScopedRetrieval reports whether resource and memory searches should be
// performed independently with explicit target URIs.
func (m IntegrationMode) UsesScopedRetrieval() bool {
	return m == ModeDocsAlignedSupplement
}

// UsesSessionMemorySupplement reports whether session-derived OpenViking memory
// should be queried as supplemental prompt input.
func (m IntegrationMode) UsesSessionMemorySupplement() bool {
	return m == ModeDocsAlignedSupplement
}

// RecordsUsedContexts reports whether actually retrieved contexts should be
// recorded on the session before commit.
func (m IntegrationMode) RecordsUsedContexts() bool {
	return m == ModeDocsAlignedSupplement
}
