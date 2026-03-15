package gametest

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// FakeEventStore is a test double for storage.EventStore.
type FakeEventStore struct {
	Events    map[string][]event.Event // campaignID -> Events
	ByHash    map[string]event.Event   // hash -> event
	AppendErr error
	ListErr   error
	GetErr    error
	NextSeq   map[string]uint64 // campaignID -> NextSeq
}

// NewFakeEventStore returns a ready-to-use event store fake.
func NewFakeEventStore() *FakeEventStore {
	return &FakeEventStore{
		Events:  make(map[string][]event.Event),
		ByHash:  make(map[string]event.Event),
		NextSeq: make(map[string]uint64),
	}
}

func (s *FakeEventStore) AppendEvent(_ context.Context, evt event.Event) (event.Event, error) {
	if s.AppendErr != nil {
		return event.Event{}, s.AppendErr
	}
	cid := string(evt.CampaignID)
	seq := s.NextSeq[cid]
	if seq == 0 {
		seq = 1
	}
	evt.Seq = seq
	evt.Hash = "fakehash-" + cid + "-" + string(rune('0'+seq))
	s.NextSeq[cid] = seq + 1
	s.Events[cid] = append(s.Events[cid], evt)
	s.ByHash[evt.Hash] = evt
	return evt, nil
}

// FakeBatchEventStore is a test double for storage.BatchEventStore.
type FakeBatchEventStore struct {
	*FakeEventStore
}

// NewFakeBatchEventStore returns a batch-capable event store fake.
func NewFakeBatchEventStore() *FakeBatchEventStore {
	return &FakeBatchEventStore{FakeEventStore: NewFakeEventStore()}
}

func (s *FakeBatchEventStore) BatchAppendEvents(ctx context.Context, events []event.Event) ([]event.Event, error) {
	if s.AppendErr != nil {
		return nil, s.AppendErr
	}
	stored := make([]event.Event, 0, len(events))
	for _, evt := range events {
		storedEvent, err := s.AppendEvent(ctx, evt)
		if err != nil {
			return nil, err
		}
		stored = append(stored, storedEvent)
	}
	return stored, nil
}

func (s *FakeEventStore) GetEventByHash(_ context.Context, hash string) (event.Event, error) {
	if s.GetErr != nil {
		return event.Event{}, s.GetErr
	}
	evt, ok := s.ByHash[hash]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	return evt, nil
}

func (s *FakeEventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	if s.GetErr != nil {
		return event.Event{}, s.GetErr
	}
	events, ok := s.Events[campaignID]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	for _, evt := range events {
		if evt.Seq == seq {
			return evt, nil
		}
	}
	return event.Event{}, storage.ErrNotFound
}

func (s *FakeEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	events, ok := s.Events[campaignID]
	if !ok {
		return nil, nil
	}
	var result []event.Event
	for _, e := range events {
		if e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *FakeEventStore) ListEventsBySession(_ context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	events, ok := s.Events[campaignID]
	if !ok {
		return nil, nil
	}
	var result []event.Event
	for _, e := range events {
		if e.SessionID.String() == sessionID && e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *FakeEventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	if s.GetErr != nil {
		return 0, s.GetErr
	}
	seq := s.NextSeq[campaignID]
	if seq == 0 {
		return 0, nil
	}
	return seq - 1, nil
}

func (s *FakeEventStore) ListEventsPage(_ context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if s.ListErr != nil {
		return storage.ListEventsPageResult{}, s.ListErr
	}
	events, ok := s.Events[req.CampaignID]
	if !ok {
		return storage.ListEventsPageResult{TotalCount: 0}, nil
	}

	sorted := make([]event.Event, len(events))
	copy(sorted, events)

	needsReverse := req.Descending
	if req.CursorReverse {
		needsReverse = !needsReverse
	}
	if needsReverse {
		for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}
	}

	base := make([]event.Event, 0, len(sorted))
	for _, e := range sorted {
		if req.AfterSeq > 0 && e.Seq <= req.AfterSeq {
			continue
		}
		if filter := req.Filter; filter.EventType != "" && string(e.Type) != filter.EventType {
			continue
		}
		if filter := req.Filter; filter.EntityType != "" && e.EntityType != filter.EntityType {
			continue
		}
		if filter := req.Filter; filter.EntityID != "" && e.EntityID != filter.EntityID {
			continue
		}
		base = append(base, e)
	}

	var filtered []event.Event
	for _, e := range base {
		if req.CursorSeq > 0 {
			if req.CursorDir == "bwd" {
				if e.Seq >= req.CursorSeq {
					continue
				}
			} else {
				if e.Seq <= req.CursorSeq {
					continue
				}
			}
		}
		filtered = append(filtered, e)
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	var result []event.Event
	hasMore := false
	if len(filtered) > pageSize {
		result = filtered[:pageSize]
		hasMore = true
	} else {
		result = filtered
	}

	if req.CursorReverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	var hasNextPage, hasPrevPage bool
	if req.CursorReverse {
		hasNextPage = true
		hasPrevPage = hasMore
	} else {
		hasNextPage = hasMore
		hasPrevPage = req.CursorSeq > 0
	}

	return storage.ListEventsPageResult{
		Events:      result,
		HasNextPage: hasNextPage,
		HasPrevPage: hasPrevPage,
		TotalCount:  len(base),
	}, nil
}
