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

// ProjectionApplyTxStore is the transaction-scoped projection contract needed
// by exactly-once projection apply callbacks.
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
	SceneStore
	SceneCharacterStore
	SceneGateStore
	SceneSpotlightStore
	DaggerheartStore
	ProjectionWatermarkStore
}

// ProjectionApplyExactlyOnceStore applies one event to projections exactly once
// per campaign/sequence checkpoint.
type ProjectionApplyExactlyOnceStore interface {
	// DaggerheartStore is required so callers can bind system adapters before
	// running the exactly-once callback.
	DaggerheartStore
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
