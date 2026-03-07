// Package catalog provides embedded starter listing content that ships with the
// listing service binary. Listings are self-contained descriptions of what a
// player would get if they fork the listing; they do not require a real
// game-service campaign to exist.
package catalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"github.com/louisbranch/fracturing.space/internal/services/listing/storage"
)

//go:embed data/starter-listings.v1.json
var starterListingsJSON []byte

//go:embed data/storylines
var storylinesFS embed.FS

var (
	loadOnce     sync.Once
	cachedResult []storage.CampaignListing
	loadErr      error
)

// listingJSON mirrors the JSON schema in starter-listings.v1.json.
type listingJSON struct {
	CampaignID                 string   `json:"campaign_id"`
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

// BuiltinListings returns a deep copy of all embedded starter listings.
// The result is parsed and cached on first call; subsequent calls return
// fresh copies so callers cannot mutate package state.
func BuiltinListings() ([]storage.CampaignListing, error) {
	loadOnce.Do(func() {
		cachedResult, loadErr = loadStarterListings()
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return copyListings(cachedResult), nil
}

func loadStarterListings() ([]storage.CampaignListing, error) {
	var entries []listingJSON
	if err := json.Unmarshal(starterListingsJSON, &entries); err != nil {
		return nil, fmt.Errorf("decode starter listings JSON: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("starter listings catalog is empty")
	}

	listings := make([]storage.CampaignListing, 0, len(entries))
	for i, entry := range entries {
		listing, err := convertEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("starter listing [%d] %q: %w", i, entry.CampaignID, err)
		}
		listings = append(listings, listing)
	}
	return listings, nil
}

func convertEntry(entry listingJSON) (storage.CampaignListing, error) {
	campaignID := strings.TrimSpace(entry.CampaignID)
	if campaignID == "" {
		return storage.CampaignListing{}, fmt.Errorf("campaign_id is required")
	}
	title := strings.TrimSpace(entry.Title)
	if title == "" {
		return storage.CampaignListing{}, fmt.Errorf("title is required")
	}

	difficultyTier, err := parseDifficultyTier(entry.DifficultyTier)
	if err != nil {
		return storage.CampaignListing{}, err
	}
	system, err := parseGameSystem(entry.System)
	if err != nil {
		return storage.CampaignListing{}, err
	}
	gmMode, err := parseGmMode(entry.GmMode)
	if err != nil {
		return storage.CampaignListing{}, err
	}
	intent, err := parseIntent(entry.Intent)
	if err != nil {
		return storage.CampaignListing{}, err
	}

	storyline, err := loadStoryline(entry.StorylineFile)
	if err != nil {
		return storage.CampaignListing{}, err
	}

	tags := make([]string, len(entry.Tags))
	copy(tags, entry.Tags)

	return storage.CampaignListing{
		CampaignID:                 campaignID,
		Title:                      title,
		Description:                strings.TrimSpace(entry.Description),
		RecommendedParticipantsMin: entry.RecommendedParticipantsMin,
		RecommendedParticipantsMax: entry.RecommendedParticipantsMax,
		DifficultyTier:             difficultyTier,
		ExpectedDurationLabel:      strings.TrimSpace(entry.ExpectedDurationLabel),
		System:                     system,
		GmMode:                     gmMode,
		Intent:                     intent,
		Level:                      entry.Level,
		CharacterCount:             entry.CharacterCount,
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

func parseDifficultyTier(s string) (listingv1.CampaignDifficultyTier, error) {
	s = strings.TrimSpace(s)
	if v, ok := listingv1.CampaignDifficultyTier_value[s]; ok {
		return listingv1.CampaignDifficultyTier(v), nil
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

func parseGmMode(s string) (listingv1.CampaignListingGmMode, error) {
	s = strings.TrimSpace(s)
	if v, ok := listingv1.CampaignListingGmMode_value[s]; ok {
		return listingv1.CampaignListingGmMode(v), nil
	}
	return 0, fmt.Errorf("unknown gm_mode %q", s)
}

func parseIntent(s string) (listingv1.CampaignListingIntent, error) {
	s = strings.TrimSpace(s)
	if v, ok := listingv1.CampaignListingIntent_value[s]; ok {
		return listingv1.CampaignListingIntent(v), nil
	}
	return 0, fmt.Errorf("unknown intent %q", s)
}

func copyListings(src []storage.CampaignListing) []storage.CampaignListing {
	out := make([]storage.CampaignListing, len(src))
	for i, l := range src {
		out[i] = l
		out[i].Tags = make([]string, len(l.Tags))
		copy(out[i].Tags, l.Tags)
	}
	return out
}
