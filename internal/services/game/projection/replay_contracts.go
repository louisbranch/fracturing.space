package projection

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// EventApplier applies one replayed event to projection stores.
type EventApplier interface {
	Apply(context.Context, event.Event) error
}

// ReplayEventStore lists campaign events in ascending sequence order.
type ReplayEventStore interface {
	ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error)
}

// GapRepairEventStore is the event-store contract needed by gap repair.
type GapRepairEventStore interface {
	ReplayEventStore
	EventHighWaterStore
}

var _ EventApplier = Applier{}
