package game

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type EventStoreAdapter struct {
	store storage.EventStore
}

// NewEventStoreAdapter adapts the event store for replay.
func NewEventStoreAdapter(store storage.EventStore) replay.EventStore {
	return EventStoreAdapter{store: store}
}

func (a EventStoreAdapter) ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if a.store == nil {
		return nil, nil
	}
	entries, err := a.store.ListEvents(ctx, campaignID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

type JournalAdapter struct {
	store storage.EventStore
}

// NewJournalAdapter adapts the event store for journaling.
func NewJournalAdapter(store storage.EventStore) engine.EventJournal {
	return JournalAdapter{store: store}
}

func (a JournalAdapter) Append(ctx context.Context, evt event.Event) (event.Event, error) {
	if a.store == nil {
		return event.Event{}, nil
	}
	stored, err := a.store.AppendEvent(ctx, evt)
	if err != nil {
		return event.Event{}, err
	}
	return stored, nil
}
