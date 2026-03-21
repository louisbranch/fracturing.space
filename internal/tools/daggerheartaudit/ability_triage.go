package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// abilityDomainCardMatch records when a reference ability resolves to a
// domain card in the content catalog, proving the mapping.
type abilityDomainCardMatch struct {
	DomainCardID   string
	DomainCardName string
	DomainID       string
	FeatureText    string
}

// domainCardImportRecord captures the fields we need from the import JSON.
type domainCardImportRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainID    string `json:"domain_id"`
	FeatureText string `json:"feature_text"`
}

// classifyAbilityEntries loads domain cards from the import data and matches
// each ability reference entry to its corresponding domain card by name.
// Returns a map from ability reference ID to its matched domain card.
func classifyAbilityEntries(repoRoot string, entries []corpusIndexEntry) (map[string]abilityDomainCardMatch, error) {
	cardsPath := filepath.Join(repoRoot,
		"internal", "tools", "importer", "content", "daggerheart", "v1", "en-US", "domain_cards.json")
	data, err := os.ReadFile(cardsPath)
	if err != nil {
		return nil, fmt.Errorf("read domain cards: %w", err)
	}
	var wrapper struct {
		Items []domainCardImportRecord `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("decode domain cards: %w", err)
	}
	cards := wrapper.Items
	byName := make(map[string]domainCardImportRecord, len(cards))
	for _, card := range cards {
		key := strings.ToLower(strings.TrimSpace(card.Name))
		if key != "" {
			byName[key] = card
		}
	}

	result := make(map[string]abilityDomainCardMatch)
	for _, entry := range entries {
		if entry.Kind != "ability" {
			continue
		}
		titleKey := strings.ToLower(strings.TrimSpace(entry.Title))
		if card, ok := byName[titleKey]; ok {
			result[entry.ID] = abilityDomainCardMatch{
				DomainCardID:   card.ID,
				DomainCardName: card.Name,
				DomainID:       card.DomainID,
				FeatureText:    card.FeatureText,
			}
		}
	}
	return result, nil
}
