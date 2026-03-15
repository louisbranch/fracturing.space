package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// ProjectionApplyOutboxProcessor drains due outbox rows and applies
// projections through the provided callback.
type ProjectionApplyOutboxProcessor interface {
	ProcessProjectionApplyOutbox(ctx context.Context, now time.Time, limit int, apply func(context.Context, event.Event) error) (int, error)
}

// ProjectionApplyOutboxShadowProcessor drains outbox rows in shadow mode
// (retry scheduling without projection mutation).
type ProjectionApplyOutboxShadowProcessor interface {
	ProcessProjectionApplyOutboxShadow(ctx context.Context, now time.Time, limit int) (int, error)
}

// ProjectionApplyOutboxSummary reports queue depth and the oldest retry-eligible row.
type ProjectionApplyOutboxSummary struct {
	PendingCount            int
	ProcessingCount         int
	FailedCount             int
	DeadCount               int
	OldestPendingCampaignID string
	OldestPendingSeq        uint64
	OldestPendingAt         time.Time
}

// ProjectionApplyOutboxEntry describes one outbox row for inspection tooling.
type ProjectionApplyOutboxEntry struct {
	CampaignID    string
	Seq           uint64
	EventType     event.Type
	Status        string
	AttemptCount  int
	NextAttemptAt time.Time
	LastError     string
	UpdatedAt     time.Time
}

// ProjectionApplyOutboxInspector reports queue state for maintenance tooling.
type ProjectionApplyOutboxInspector interface {
	GetProjectionApplyOutboxSummary(context.Context) (ProjectionApplyOutboxSummary, error)
	ListProjectionApplyOutboxRows(context.Context, string, int) ([]ProjectionApplyOutboxEntry, error)
}

// ProjectionApplyOutboxRequeuer transitions dead queue rows back to pending.
type ProjectionApplyOutboxRequeuer interface {
	RequeueProjectionApplyOutboxRow(context.Context, string, uint64, time.Time) (bool, error)
	RequeueProjectionApplyOutboxDeadRows(context.Context, int, time.Time) (int, error)
}

// ProjectionApplyOutboxStore groups worker, inspection, and requeue seams for
// the event-journal-backed projection apply queue.
type ProjectionApplyOutboxStore interface {
	ProjectionApplyOutboxProcessor
	ProjectionApplyOutboxShadowProcessor
	ProjectionApplyOutboxInspector
	ProjectionApplyOutboxRequeuer
}

// ProjectionApplyTxStore is the transaction-scoped projection contract needed
// by exactly-once projection apply callbacks. Core projection stores only —
// system-specific stores are extracted from explicit provider seams on the
// concrete implementation during adapter rebinding.
type ProjectionApplyTxStore interface {
	CampaignStore
	CharacterStore
	CampaignForkStore
	ClaimIndexStore
	InviteStore
	ParticipantStore
	SessionStore
	SessionGateStore
	SessionSpotlightStore
	SessionInteractionStore
	SceneStore
	SceneCharacterStore
	SceneGateStore
	SceneSpotlightStore
	SceneInteractionStore
	ProjectionWatermarkStore
}

// ProjectionApplyExactlyOnceStore applies one event to projections exactly once
// per campaign/sequence checkpoint.
type ProjectionApplyExactlyOnceStore interface {
	ApplyProjectionEventExactlyOnce(
		ctx context.Context,
		evt event.Event,
		apply func(context.Context, event.Event, ProjectionApplyTxStore) error,
	) (bool, error)
}

// EventIntegrityVerifier validates event-chain integrity for startup checks.
type EventIntegrityVerifier interface {
	VerifyEventIntegrity(ctx context.Context) error
}
