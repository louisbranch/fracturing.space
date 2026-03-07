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
// by exactly-once projection apply callbacks. Core projection stores only —
// system-specific stores are accessed via type assertion on the concrete
// implementation (e.g. DaggerheartStore) during adapter rebinding.
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
