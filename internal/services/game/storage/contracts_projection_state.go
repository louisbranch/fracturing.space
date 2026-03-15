package storage

import (
	"context"
	"time"
)

// Snapshot is a materialized campaign/session state checkpoint derived from the event journal.
// Snapshots are accelerators for replay, not the source of authority.
type Snapshot struct {
	CampaignID          string
	SessionID           string
	EventSeq            uint64
	CharacterStatesJSON []byte
	GMStateJSON         []byte
	SystemStateJSON     []byte
	CreatedAt           time.Time
}

// SnapshotStore persists replay checkpoints used to jump event replay work.
type SnapshotStore interface {
	// PutSnapshot stores a snapshot.
	PutSnapshot(ctx context.Context, snap Snapshot) error
	// GetSnapshot retrieves a snapshot by campaign and session ID.
	GetSnapshot(ctx context.Context, campaignID, sessionID string) (Snapshot, error)
	// GetLatestSnapshot retrieves the most recent snapshot for a campaign.
	GetLatestSnapshot(ctx context.Context, campaignID string) (Snapshot, error)
	// ListSnapshots returns snapshots ordered by event sequence descending.
	ListSnapshots(ctx context.Context, campaignID string, limit int) ([]Snapshot, error)
}

// ParticipantClaim describes enforced uniqueness of user-to-seat binding.
type ParticipantClaim struct {
	CampaignID    string
	UserID        string
	ParticipantID string
	ClaimedAt     time.Time
}

// ClaimIndexStore keeps seat claim uniqueness from drifting during concurrent joins.
type ClaimIndexStore interface {
	// PutParticipantClaim stores a user claim for a participant seat.
	PutParticipantClaim(ctx context.Context, campaignID, userID, participantID string, claimedAt time.Time) error
	// GetParticipantClaim returns the claim for a user in a campaign.
	GetParticipantClaim(ctx context.Context, campaignID, userID string) (ParticipantClaim, error)
	// DeleteParticipantClaim removes a claim by user.
	DeleteParticipantClaim(ctx context.Context, campaignID, userID string) error
}

// ForkMetadata tracks campaign lineage needed for fork navigation and support tooling.
type ForkMetadata struct {
	ParentCampaignID string
	ForkEventSeq     uint64
	OriginCampaignID string
}

// CampaignForkStore persists fork lineage metadata for derived-campaign workflows.
type CampaignForkStore interface {
	// GetCampaignForkMetadata retrieves fork metadata for a campaign.
	GetCampaignForkMetadata(ctx context.Context, campaignID string) (ForkMetadata, error)
	// SetCampaignForkMetadata sets fork metadata for a campaign.
	SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata ForkMetadata) error
}

// ProjectionWatermark tracks the highest event sequence successfully applied
// to projections for a given campaign. Comparing watermarks against the event
// journal high-water mark reveals projection gaps that need repair.
type ProjectionWatermark struct {
	CampaignID      string
	AppliedSeq      uint64
	ExpectedNextSeq uint64
	UpdatedAt       time.Time
}

// ProjectionWatermarkStore tracks per-campaign projection application progress
// so startup can detect and repair gaps between the event journal and projections.
type ProjectionWatermarkStore interface {
	// GetProjectionWatermark returns the watermark for a campaign.
	// Returns ErrNotFound if no watermark exists.
	GetProjectionWatermark(ctx context.Context, campaignID string) (ProjectionWatermark, error)
	// SaveProjectionWatermark upserts the watermark for a campaign.
	SaveProjectionWatermark(ctx context.Context, wm ProjectionWatermark) error
	// ListProjectionWatermarks returns all watermarks, typically for startup gap detection.
	ListProjectionWatermarks(ctx context.Context) ([]ProjectionWatermark, error)
}

// ProjectionStore groups read-model-oriented stores consumed by APIs and queries.
// System-specific stores (for example Daggerheart gameplay state) are accessed
// through explicit consumer-owned seams rather than embedded in this core
// composite.
type ProjectionStore interface {
	CampaignStore
	ParticipantStore
	ClaimIndexStore
	InviteStore
	CharacterStore
	SessionStore
	SnapshotStore
	CampaignForkStore
	StatisticsStore
	ProjectionWatermarkStore
}

// Store is a composite interface for all persistence concerns used across event
// sourcing, projection application, and queries. System-specific stores are
// accessed via explicit provider seams on the concrete implementation.
type Store interface {
	CampaignStore
	ParticipantStore
	ClaimIndexStore
	CharacterStore
	InviteStore
	SessionStore
	EventStore
	AuditEventStore
	StatisticsStore
	SnapshotStore
	CampaignForkStore
	Close() error
}
