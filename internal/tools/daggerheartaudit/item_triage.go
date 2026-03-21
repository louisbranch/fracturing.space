package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// itemImportRecord captures the fields we need from items.json.
type itemImportRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	EffectText string `json:"effect_text"`
}

// itemEffectMatch records when a reference item/consumable resolves to an
// entry in the import catalog with classifiable effect text.
type itemEffectMatch struct {
	ImportID   string
	Name       string
	Kind       string
	EffectText string
}

// classifyItemEntries loads items from the import data and builds a map from
// reference ID to the item's effect text for classification.
func classifyItemEntries(repoRoot string) (map[string]itemEffectMatch, error) {
	itemsPath := filepath.Join(repoRoot,
		"internal", "tools", "importer", "content", "daggerheart", "v1", "en-US", "items.json")
	data, err := os.ReadFile(itemsPath)
	if err != nil {
		return nil, fmt.Errorf("read items: %w", err)
	}
	var wrapper struct {
		Items []itemImportRecord `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("decode items: %w", err)
	}

	result := make(map[string]itemEffectMatch, len(wrapper.Items)*2)
	for _, item := range wrapper.Items {
		match := itemEffectMatch{
			ImportID:   item.ID,
			Name:       item.Name,
			Kind:       item.Kind,
			EffectText: item.EffectText,
		}

		// Import IDs use "." separator (e.g., "item.acidpaste"),
		// reference IDs use "-" (e.g., "consumable-acidpaste").
		// The reference corpus may classify the same entry under a different
		// kind prefix (item vs consumable), so register under both prefixes
		// to ensure the lookup succeeds regardless of which prefix the
		// reference uses.
		refID := strings.Replace(item.ID, ".", "-", 1)
		result[refID] = match

		parts := strings.SplitN(item.ID, ".", 2)
		if len(parts) == 2 {
			slug := parts[1]
			altPrefix := "consumable"
			if parts[0] == "consumable" {
				altPrefix = "item"
			}
			altID := altPrefix + "-" + slug
			if _, exists := result[altID]; !exists {
				result[altID] = match
			}
		}
	}
	return result, nil
}
