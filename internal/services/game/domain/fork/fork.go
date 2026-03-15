package fork

import (
	"fmt"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

var (
	// ErrEmptyCampaignID indicates a missing campaign ID.
	ErrEmptyCampaignID = apperrors.New(apperrors.CodeForkEmptyCampaignID, "source campaign id is required")
)

// ForkPoint identifies where to fork in the source campaign's event history.
type ForkPoint struct {
	// EventSeq is the event sequence number to fork at.
	// If 0, forks at the latest event (current HEAD).
	EventSeq uint64
	// SessionID optionally specifies a session boundary to fork at.
	// If set, EventSeq is ignored and the fork occurs at the end of this session.
	SessionID string
}

// IsSessionBoundary reports whether this fork point is a session boundary.
func (fp ForkPoint) IsSessionBoundary() bool {
	return fp.SessionID != ""
}

// Fork represents a completed fork operation result.
type Fork struct {
	// SourceCampaignID is the campaign that was forked.
	SourceCampaignID string
	// NewCampaignID is the ID of the new forked campaign.
	NewCampaignID string
	// ForkEventSeq is the actual event sequence at which the fork occurred.
	ForkEventSeq uint64
	// OriginCampaignID is the root of the lineage.
	OriginCampaignID string
	// CreatedAt is when the fork was created.
	CreatedAt time.Time
}

// CreateForkInput describes the input for creating a fork.
type CreateForkInput struct {
	SourceCampaignID string
	ForkPoint        ForkPoint
	NewCampaignName  string
	CopyParticipants bool
}

// CreateFork creates a new Fork with a generated ID and timestamps.
func CreateFork(input CreateForkInput, originCampaignID string, forkEventSeq uint64, now func() time.Time, idGenerator func() (string, error)) (Fork, error) {
	if now == nil {
		panic("fork: now function must not be nil")
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	if strings.TrimSpace(input.SourceCampaignID) == "" {
		return Fork{}, ErrEmptyCampaignID
	}

	newCampaignID, err := idGenerator()
	if err != nil {
		return Fork{}, fmt.Errorf("generate fork campaign id: %w", err)
	}

	// If the source campaign has no origin (is itself an origin), use its ID
	if originCampaignID == "" {
		originCampaignID = input.SourceCampaignID
	}

	return Fork{
		SourceCampaignID: input.SourceCampaignID,
		NewCampaignID:    newCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: originCampaignID,
		CreatedAt:        now().UTC(),
	}, nil
}

// Lineage represents the ancestry chain of a campaign.
type Lineage struct {
	// CampaignID is the campaign this lineage describes.
	CampaignID string
	// ParentCampaignID is the immediate parent (empty if original).
	ParentCampaignID string
	// ForkEventSeq is the event sequence at which this campaign was forked.
	ForkEventSeq uint64
	// OriginCampaignID is the root of the lineage.
	OriginCampaignID string
	// Depth is the number of forks from the origin (0 for originals).
	Depth int
}

// IsOriginal reports whether this campaign is an original (not forked).
func (l Lineage) IsOriginal() bool {
	return l.ParentCampaignID == ""
}
