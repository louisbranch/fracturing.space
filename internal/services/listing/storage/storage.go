// Package storage defines persistence contracts for listing service state.
package storage

import (
	"context"
	"errors"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
)

var (
	// ErrNotFound indicates a requested listing record is missing.
	ErrNotFound = errors.New("record not found")
	// ErrAlreadyExists indicates a uniqueness-constrained listing already exists.
	ErrAlreadyExists = errors.New("record already exists")
)

// CampaignListing stores one public listing record for a source campaign.
type CampaignListing struct {
	CampaignID                 string
	Title                      string
	Description                string
	RecommendedParticipantsMin int
	RecommendedParticipantsMax int
	DifficultyTier             listingv1.CampaignDifficultyTier
	ExpectedDurationLabel      string
	System                     commonv1.GameSystem
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// CampaignListingPage stores one page of listing records.
type CampaignListingPage struct {
	Listings      []CampaignListing
	NextPageToken string
}

// CampaignListingStore persists campaign listing records.
type CampaignListingStore interface {
	CreateCampaignListing(ctx context.Context, listing CampaignListing) error
	GetCampaignListing(ctx context.Context, campaignID string) (CampaignListing, error)
	ListCampaignListings(ctx context.Context, pageSize int, pageToken string) (CampaignListingPage, error)
}
