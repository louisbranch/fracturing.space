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
	cachedResult []StarterDefinition
	loadErr      error
)

// StarterDefinition is the canonical builtin starter definition used for both
// public discovery browse data and internal starter-template reconciliation.
type StarterDefinition struct {
	Entry     storage.DiscoveryEntry
	Character StarterCharacterDefinition
}

// StarterCharacterDefinition captures the premade player character built for
// one starter template campaign.
type StarterCharacterDefinition struct {
	Name          string
	Pronouns      string
	Summary       string
	ClassID       string
	SubclassID    string
	AncestryID    string
	CommunityID   string
	WeaponIDs     []string
	ArmorID       string
	PotionItemID  string
	Description   string
	Background    string
	Connections   string
	DomainCardIDs []string
	Traits        StarterTraitDefinition
	Experiences   []StarterExperienceDefinition
}

// StarterTraitDefinition captures fixed starter trait allocation.
type StarterTraitDefinition struct {
	Agility   int32
	Strength  int32
	Finesse   int32
	Instinct  int32
	Presence  int32
	Knowledge int32
}

// StarterExperienceDefinition captures one premade experience line.
type StarterExperienceDefinition struct {
	Name     string
	Modifier int32
}

// starterJSON mirrors the JSON schema in discovery-entries.v1.json.
type starterJSON struct {
	EntryID                    string               `json:"entry_id"`
	Kind                       string               `json:"kind"`
	SourceID                   string               `json:"source_id"`
	Title                      string               `json:"title"`
	Description                string               `json:"description"`
	CampaignTheme              string               `json:"campaign_theme"`
	RecommendedParticipantsMin int                  `json:"recommended_participants_min"`
	RecommendedParticipantsMax int                  `json:"recommended_participants_max"`
	DifficultyTier             string               `json:"difficulty_tier"`
	ExpectedDurationLabel      string               `json:"expected_duration_label"`
	System                     string               `json:"system"`
	GmMode                     string               `json:"gm_mode"`
	Intent                     string               `json:"intent"`
	Level                      int                  `json:"level"`
	CharacterCount             int                  `json:"character_count"`
	StorylineFile              string               `json:"storyline_file"`
	Tags                       []string             `json:"tags"`
	PreviewHook                string               `json:"preview_hook"`
	PreviewPlaystyleLabel      string               `json:"preview_playstyle_label"`
	PreviewCharacterName       string               `json:"preview_character_name"`
	PreviewCharacterSummary    string               `json:"preview_character_summary"`
	PremadeCharacter           starterCharacterJSON `json:"premade_character"`
}

type starterCharacterJSON struct {
	Name          string                  `json:"name"`
	Pronouns      string                  `json:"pronouns"`
	Summary       string                  `json:"summary"`
	ClassID       string                  `json:"class_id"`
	SubclassID    string                  `json:"subclass_id"`
	AncestryID    string                  `json:"ancestry_id"`
	CommunityID   string                  `json:"community_id"`
	WeaponIDs     []string                `json:"weapon_ids"`
	ArmorID       string                  `json:"armor_id"`
	PotionItemID  string                  `json:"potion_item_id"`
	Description   string                  `json:"description"`
	Background    string                  `json:"background"`
	Connections   string                  `json:"connections"`
	DomainCardIDs []string                `json:"domain_card_ids"`
	Traits        starterTraitsJSON       `json:"traits"`
	Experiences   []starterExperienceJSON `json:"experiences"`
}

type starterTraitsJSON struct {
	Agility   int32 `json:"agility"`
	Strength  int32 `json:"strength"`
	Finesse   int32 `json:"finesse"`
	Instinct  int32 `json:"instinct"`
	Presence  int32 `json:"presence"`
	Knowledge int32 `json:"knowledge"`
}

type starterExperienceJSON struct {
	Name     string `json:"name"`
	Modifier int32  `json:"modifier"`
}

// BuiltinEntries returns a deep copy of all embedded discovery entries.
func BuiltinEntries() ([]storage.DiscoveryEntry, error) {
	starters, err := BuiltinStarters()
	if err != nil {
		return nil, err
	}
	entries := make([]storage.DiscoveryEntry, 0, len(starters))
	for _, starter := range starters {
		entries = append(entries, starter.Entry)
	}
	return entries, nil
}

// BuiltinStarters returns the canonical builtin starter definitions.
func BuiltinStarters() ([]StarterDefinition, error) {
	loadOnce.Do(func() {
		cachedResult, loadErr = loadStarterDefinitions()
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return copyStarters(cachedResult), nil
}

func loadStarterDefinitions() ([]StarterDefinition, error) {
	var entries []starterJSON
	if err := json.Unmarshal(discoveryEntriesJSON, &entries); err != nil {
		return nil, fmt.Errorf("decode discovery entries JSON: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("discovery entries catalog is empty")
	}

	out := make([]StarterDefinition, 0, len(entries))
	for i, raw := range entries {
		entry, err := convertStarter(raw)
		if err != nil {
			return nil, fmt.Errorf("discovery entry [%d] %q: %w", i, raw.EntryID, err)
		}
		out = append(out, entry)
	}
	return out, nil
}

func convertStarter(raw starterJSON) (StarterDefinition, error) {
	entryID := strings.TrimSpace(raw.EntryID)
	if entryID == "" {
		return StarterDefinition{}, fmt.Errorf("entry_id is required")
	}
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return StarterDefinition{}, fmt.Errorf("title is required")
	}

	kind, err := parseKind(raw.Kind)
	if err != nil {
		return StarterDefinition{}, err
	}
	difficultyTier, err := parseDifficultyTier(raw.DifficultyTier)
	if err != nil {
		return StarterDefinition{}, err
	}
	system, err := parseGameSystem(raw.System)
	if err != nil {
		return StarterDefinition{}, err
	}
	gmMode, err := parseGmMode(raw.GmMode)
	if err != nil {
		return StarterDefinition{}, err
	}
	intent, err := parseIntent(raw.Intent)
	if err != nil {
		return StarterDefinition{}, err
	}
	storyline, err := loadStoryline(raw.StorylineFile)
	if err != nil {
		return StarterDefinition{}, err
	}

	tags := make([]string, len(raw.Tags))
	copy(tags, raw.Tags)
	character, err := convertStarterCharacter(raw.PremadeCharacter)
	if err != nil {
		return StarterDefinition{}, err
	}
	if strings.TrimSpace(raw.PreviewCharacterName) == "" {
		raw.PreviewCharacterName = character.Name
	}
	if strings.TrimSpace(raw.PreviewCharacterSummary) == "" {
		raw.PreviewCharacterSummary = character.Summary
	}

	return StarterDefinition{
		Entry: storage.DiscoveryEntry{
			EntryID:                    entryID,
			Kind:                       kind,
			SourceID:                   strings.TrimSpace(raw.SourceID),
			Title:                      title,
			Description:                strings.TrimSpace(raw.Description),
			CampaignTheme:              strings.TrimSpace(raw.CampaignTheme),
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
			PreviewHook:                strings.TrimSpace(raw.PreviewHook),
			PreviewPlaystyleLabel:      strings.TrimSpace(raw.PreviewPlaystyleLabel),
			PreviewCharacterName:       strings.TrimSpace(raw.PreviewCharacterName),
			PreviewCharacterSummary:    strings.TrimSpace(raw.PreviewCharacterSummary),
		},
		Character: character,
	}, nil
}

func convertStarterCharacter(raw starterCharacterJSON) (StarterCharacterDefinition, error) {
	character := StarterCharacterDefinition{
		Name:          strings.TrimSpace(raw.Name),
		Pronouns:      strings.TrimSpace(raw.Pronouns),
		Summary:       strings.TrimSpace(raw.Summary),
		ClassID:       strings.TrimSpace(raw.ClassID),
		SubclassID:    strings.TrimSpace(raw.SubclassID),
		AncestryID:    strings.TrimSpace(raw.AncestryID),
		CommunityID:   strings.TrimSpace(raw.CommunityID),
		ArmorID:       strings.TrimSpace(raw.ArmorID),
		PotionItemID:  strings.TrimSpace(raw.PotionItemID),
		Description:   strings.TrimSpace(raw.Description),
		Background:    strings.TrimSpace(raw.Background),
		Connections:   strings.TrimSpace(raw.Connections),
		WeaponIDs:     trimValues(raw.WeaponIDs),
		DomainCardIDs: trimValues(raw.DomainCardIDs),
		Traits: StarterTraitDefinition{
			Agility:   raw.Traits.Agility,
			Strength:  raw.Traits.Strength,
			Finesse:   raw.Traits.Finesse,
			Instinct:  raw.Traits.Instinct,
			Presence:  raw.Traits.Presence,
			Knowledge: raw.Traits.Knowledge,
		},
	}
	for _, experience := range raw.Experiences {
		name := strings.TrimSpace(experience.Name)
		if name == "" {
			continue
		}
		character.Experiences = append(character.Experiences, StarterExperienceDefinition{
			Name:     name,
			Modifier: experience.Modifier,
		})
	}
	if character.Name == "" {
		return StarterCharacterDefinition{}, fmt.Errorf("premade_character.name is required")
	}
	if character.ClassID == "" || character.SubclassID == "" {
		return StarterCharacterDefinition{}, fmt.Errorf("premade_character class and subclass are required")
	}
	if character.AncestryID == "" || character.CommunityID == "" {
		return StarterCharacterDefinition{}, fmt.Errorf("premade_character ancestry and community are required")
	}
	return character, nil
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

func copyStarters(src []StarterDefinition) []StarterDefinition {
	out := make([]StarterDefinition, len(src))
	for i, starter := range src {
		out[i] = starter
		out[i].Entry.Tags = append([]string(nil), starter.Entry.Tags...)
		out[i].Character.WeaponIDs = append([]string(nil), starter.Character.WeaponIDs...)
		out[i].Character.DomainCardIDs = append([]string(nil), starter.Character.DomainCardIDs...)
		out[i].Character.Experiences = append([]StarterExperienceDefinition(nil), starter.Character.Experiences...)
	}
	return out
}

func trimValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
