// Package catalog provides embedded starter discovery content that ships with
// the discovery service binary.
package catalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
)

//go:embed data/discovery-entries.v1.json
var discoveryEntriesJSON []byte

//go:embed data/storylines
var storylinesFS embed.FS

var (
	loadOnce     sync.Once
	cachedResult []storage.DiscoveryEntry
	loadErr      error
)

// entryJSON mirrors the JSON schema in discovery-entries.v1.json.
type entryJSON struct {
	EntryID                    string   `json:"entry_id"`
	Kind                       string   `json:"kind"`
	SourceID                   string   `json:"source_id"`
	Title                      string   `json:"title"`
	Description                string   `json:"description"`
	RecommendedParticipantsMin int      `json:"recommended_participants_min"`
	RecommendedParticipantsMax int      `json:"recommended_participants_max"`
	DifficultyTier             string   `json:"difficulty_tier"`
	ExpectedDurationLabel      string   `json:"expected_duration_label"`
	System                     string   `json:"system"`
	GmMode                     string   `json:"gm_mode"`
	Intent                     string   `json:"intent"`
	Level                      int      `json:"level"`
	CharacterCount             int      `json:"character_count"`
	StorylineFile              string   `json:"storyline_file"`
	Tags                       []string `json:"tags"`
}

// BuiltinEntries returns a deep copy of all embedded discovery entries.
func BuiltinEntries() ([]storage.DiscoveryEntry, error) {
	loadOnce.Do(func() {
		cachedResult, loadErr = loadDiscoveryEntries()
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return copyEntries(cachedResult), nil
}

func loadDiscoveryEntries() ([]storage.DiscoveryEntry, error) {
	var entries []entryJSON
	if err := json.Unmarshal(discoveryEntriesJSON, &entries); err != nil {
		return nil, fmt.Errorf("decode discovery entries JSON: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("discovery entries catalog is empty")
	}

	out := make([]storage.DiscoveryEntry, 0, len(entries))
	for i, raw := range entries {
		entry, err := convertEntry(raw)
		if err != nil {
			return nil, fmt.Errorf("discovery entry [%d] %q: %w", i, raw.EntryID, err)
		}
		out = append(out, entry)
	}
	return out, nil
}

func convertEntry(raw entryJSON) (storage.DiscoveryEntry, error) {
	entryID := strings.TrimSpace(raw.EntryID)
	if entryID == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("entry_id is required")
	}
	sourceID := strings.TrimSpace(raw.SourceID)
	if sourceID == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("source_id is required")
	}
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return storage.DiscoveryEntry{}, fmt.Errorf("title is required")
	}

	kind, err := parseKind(raw.Kind)
	if err != nil {
		return storage.DiscoveryEntry{}, err
	}
	difficultyTier, err := parseDifficultyTier(raw.DifficultyTier)
	if err != nil {
		return storage.DiscoveryEntry{}, err
	}
	system, err := parseGameSystem(raw.System)
	if err != nil {
		return storage.DiscoveryEntry{}, err
	}
	gmMode, err := parseGmMode(raw.GmMode)
	if err != nil {
		return storage.DiscoveryEntry{}, err
	}
	intent, err := parseIntent(raw.Intent)
	if err != nil {
		return storage.DiscoveryEntry{}, err
	}
	storyline, err := loadStoryline(raw.StorylineFile)
	if err != nil {
		return storage.DiscoveryEntry{}, err
	}

	tags := make([]string, len(raw.Tags))
	copy(tags, raw.Tags)

	return storage.DiscoveryEntry{
		EntryID:                    entryID,
		Kind:                       kind,
		SourceID:                   sourceID,
		Title:                      title,
		Description:                strings.TrimSpace(raw.Description),
		RecommendedParticipantsMin: raw.RecommendedParticipantsMin,
		RecommendedParticipantsMax: raw.RecommendedParticipantsMax,
		DifficultyTier:             difficultyTier,
		ExpectedDurationLabel:      strings.TrimSpace(raw.ExpectedDurationLabel),
		System:                     system,
		GmMode:                     gmMode,
		Intent:                     intent,
		Level:                      raw.Level,
		CharacterCount:             raw.CharacterCount,
		Storyline:                  storyline,
		Tags:                       tags,
	}, nil
}

func loadStoryline(filename string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", fmt.Errorf("storyline_file is required")
	}
	data, err := storylinesFS.ReadFile("data/storylines/" + filename)
	if err != nil {
		return "", fmt.Errorf("read storyline %q: %w", filename, err)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("storyline %q is empty", filename)
	}
	return content, nil
}

func parseKind(s string) (discoveryv1.DiscoveryEntryKind, error) {
	s = strings.TrimSpace(s)
	if v, ok := discoveryv1.DiscoveryEntryKind_value[s]; ok {
		return discoveryv1.DiscoveryEntryKind(v), nil
	}
	return 0, fmt.Errorf("unknown kind %q", s)
}

func parseDifficultyTier(s string) (discoveryv1.DiscoveryDifficultyTier, error) {
	s = strings.TrimSpace(s)
	if v, ok := discoveryv1.DiscoveryDifficultyTier_value[s]; ok {
		return discoveryv1.DiscoveryDifficultyTier(v), nil
	}
	return 0, fmt.Errorf("unknown difficulty_tier %q", s)
}

func parseGameSystem(s string) (commonv1.GameSystem, error) {
	s = strings.TrimSpace(s)
	if v, ok := commonv1.GameSystem_value[s]; ok {
		return commonv1.GameSystem(v), nil
	}
	return 0, fmt.Errorf("unknown system %q", s)
}

func parseGmMode(s string) (discoveryv1.DiscoveryGmMode, error) {
	s = strings.TrimSpace(s)
	if v, ok := discoveryv1.DiscoveryGmMode_value[s]; ok {
		return discoveryv1.DiscoveryGmMode(v), nil
	}
	return 0, fmt.Errorf("unknown gm_mode %q", s)
}

func parseIntent(s string) (discoveryv1.DiscoveryIntent, error) {
	s = strings.TrimSpace(s)
	if v, ok := discoveryv1.DiscoveryIntent_value[s]; ok {
		return discoveryv1.DiscoveryIntent(v), nil
	}
	return 0, fmt.Errorf("unknown intent %q", s)
}

func copyEntries(src []storage.DiscoveryEntry) []storage.DiscoveryEntry {
	out := make([]storage.DiscoveryEntry, len(src))
	for i, e := range src {
		out[i] = e
		out[i].Tags = make([]string, len(e.Tags))
		copy(out[i].Tags, e.Tags)
	}
	return out
}
