package campaignartifact

import (
	"context"
	"time"
)

// Artifact stores one campaign-scoped GM working artifact.
type Artifact struct {
	CampaignID string
	Path       string
	Content    string
	ReadOnly   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Store persists campaign-scoped GM working artifacts.
type Store interface {
	PutCampaignArtifact(ctx context.Context, record Artifact) error
	GetCampaignArtifact(ctx context.Context, campaignID string, path string) (Artifact, error)
	ListCampaignArtifacts(ctx context.Context, campaignID string) ([]Artifact, error)
}
