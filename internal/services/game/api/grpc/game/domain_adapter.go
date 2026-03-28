package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

var (
	errReplayEventStoreRequired  = errors.New("replay event store is not configured")
	errJournalEventStoreRequired = errors.New("journal event store is not configured")
)

type EventStoreAdapter struct {
	store storage.EventHistoryStore
}

// NewEventStoreAdapter adapts the event store for replay.
func NewEventStoreAdapter(store storage.EventHistoryStore) replay.EventStore {
	return EventStoreAdapter{store: store}
}

func (a EventStoreAdapter) ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if a.store == nil {
		return nil, errReplayEventStoreRequired
	}
	entries, err := a.store.ListEvents(ctx, campaignID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

type JournalAdapter struct {
	store storage.EventAppender
}

// NewJournalAdapter adapts the event store for journaling.
func NewJournalAdapter(store storage.EventAppender) engine.EventJournal {
	return JournalAdapter{store: store}
}

func (a JournalAdapter) Append(ctx context.Context, evt event.Event) (event.Event, error) {
	if a.store == nil {
		return event.Event{}, errJournalEventStoreRequired
	}
	stored, err := a.store.AppendEvent(ctx, evt)
	if err != nil {
		return event.Event{}, err
	}
	return stored, nil
}

// BatchAppend atomically appends all events from a single command decision.
//
// The underlying store must implement BatchAppendEvents for atomic semantics.
// A sequential fallback was removed because partial failure mid-batch would
// persist a partial decision with chain hashes that bind permanently.
func (a JournalAdapter) BatchAppend(ctx context.Context, events []event.Event) ([]event.Event, error) {
	if a.store == nil {
		return nil, errJournalEventStoreRequired
	}
	ba, ok := a.store.(storage.BatchEventAppender)
	if !ok {
		return nil, fmt.Errorf("batch append not supported by underlying store")
	}
	return ba.BatchAppendEvents(ctx, events)
}
