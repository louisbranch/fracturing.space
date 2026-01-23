package storage

import (
	"context"
	"errors"

	"github.com/louisbranch/duality-engine/internal/campaign/domain"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New("record not found")

// CampaignStore persists campaign metadata records.
type CampaignStore interface {
	Put(ctx context.Context, campaign domain.Campaign) error
	Get(ctx context.Context, id string) (domain.Campaign, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []domain.Campaign
	NextPageToken string
}
