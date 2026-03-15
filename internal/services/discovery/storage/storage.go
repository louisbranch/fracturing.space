// Package storage defines persistence contracts for discovery service state.
package storage

import (
	"context"
	"errors"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
)

var (
	// ErrNotFound indicates a requested discovery record is missing.
	ErrNotFound = errors.New("record not found")
	// ErrAlreadyExists indicates a uniqueness-constrained discovery record already exists.
	ErrAlreadyExists = errors.New("record already exists")
)

// DiscoveryEntry stores one public discovery record.
type DiscoveryEntry struct {
	EntryID                    string
	Kind                       discoveryv1.DiscoveryEntryKind
	SourceID                   string
	Title                      string
	Description                string
	CampaignTheme              string
	RecommendedParticipantsMin int
	RecommendedParticipantsMax int
	DifficultyTier             discoveryv1.DiscoveryDifficultyTier
	ExpectedDurationLabel      string
	System                     commonv1.GameSystem
	GmMode                     discoveryv1.DiscoveryGmMode
	Intent                     discoveryv1.DiscoveryIntent
	Level                      int
	CharacterCount             int
	Storyline                  string
	Tags                       []string
	PreviewHook                string
	PreviewPlaystyleLabel      string
	PreviewCharacterName       string
	PreviewCharacterSummary    string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// DiscoveryEntryPage stores one page of discovery records.
type DiscoveryEntryPage struct {
	Entries       []DiscoveryEntry
	NextPageToken string
}

// DiscoveryEntryStore persists discovery entry records.
type DiscoveryEntryStore interface {
	CreateDiscoveryEntry(ctx context.Context, entry DiscoveryEntry) error
	GetDiscoveryEntry(ctx context.Context, entryID string) (DiscoveryEntry, error)
	ListDiscoveryEntries(ctx context.Context, pageSize int, pageToken string, kind discoveryv1.DiscoveryEntryKind) (DiscoveryEntryPage, error)
}
