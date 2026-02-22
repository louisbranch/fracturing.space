package replay

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type fakeEventStore struct {
	events []event.Event
}

func (s *fakeEventStore) ListEvents(_ context.Context, _ string, afterSeq uint64, limit int) ([]event.Event, error) {
	var result []event.Event
	for _, evt := range s.events {
		if evt.Seq > afterSeq {
			result = append(result, evt)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

type fakeCheckpointStore struct {
	seq       uint64
	saveCalls int
}

func (s *fakeCheckpointStore) Get(_ context.Context, _ string) (Checkpoint, error) {
	if s.seq == 0 {
		return Checkpoint{}, ErrCheckpointNotFound
	}
	return Checkpoint{CampaignID: "camp-1", LastSeq: s.seq}, nil
}

func (s *fakeCheckpointStore) Save(_ context.Context, checkpoint Checkpoint) error {
	if checkpoint.CampaignID == "" {
		return errors.New("campaign id required")
	}
	s.saveCalls++
	s.seq = checkpoint.LastSeq
	return nil
}

type recordingApplier struct {
	seqs []uint64
}

func (a *recordingApplier) Fold(state any, evt event.Event) (any, error) {
	a.seqs = append(a.seqs, evt.Seq)
	return state, nil
}

// cancelingApplier cancels the context after a given number of Fold calls.
type cancelingApplier struct {
	cancel      context.CancelFunc
	cancelAfter int
	calls       int
}

func (a *cancelingApplier) Fold(state any, _ event.Event) (any, error) {
	a.calls++
	if a.calls >= a.cancelAfter {
		a.cancel()
	}
	return state, nil
}

func TestReplay_CheckpointInterval(t *testing.T) {
	events := make([]event.Event, 10)
	for i := range events {
		events[i] = event.Event{CampaignID: "camp-1", Seq: uint64(i + 1)}
	}
	store := &fakeEventStore{events: events}
	checkpoints := &fakeCheckpointStore{}
	applier := &recordingApplier{}

	result, err := Replay(context.Background(), store, checkpoints, applier, "camp-1", "state", Options{
		CheckpointInterval: 3,
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if result.Applied != 10 {
		t.Fatalf("applied = %d, want 10", result.Applied)
	}
	// With interval=3 over 10 events: saves at 3, 6, 9, and final at 10 = 4 saves.
	if checkpoints.saveCalls != 4 {
		t.Fatalf("checkpoint save calls = %d, want 4", checkpoints.saveCalls)
	}
	if checkpoints.seq != 10 {
		t.Fatalf("checkpoint seq = %d, want 10", checkpoints.seq)
	}
}

func TestReplay_RespectsContextCancellation(t *testing.T) {
	events := make([]event.Event, 100)
	for i := range events {
		events[i] = event.Event{CampaignID: "camp-1", Seq: uint64(i + 1)}
	}
	store := &fakeEventStore{events: events}
	checkpoints := &fakeCheckpointStore{}

	ctx, cancel := context.WithCancel(context.Background())
	// cancelingApplier cancels context after 5 events.
	applier := &cancelingApplier{cancel: cancel, cancelAfter: 5}

	_, err := Replay(ctx, store, checkpoints, applier, "camp-1", "state", Options{})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestReplay_UsesCheckpoint(t *testing.T) {
	store := &fakeEventStore{events: []event.Event{
		{CampaignID: "camp-1", Seq: 1},
		{CampaignID: "camp-1", Seq: 2},
		{CampaignID: "camp-1", Seq: 3},
	}}
	checkpoints := &fakeCheckpointStore{seq: 1}
	applier := &recordingApplier{}

	result, err := Replay(context.Background(), store, checkpoints, applier, "camp-1", "state", Options{})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if result.LastSeq != 3 {
		t.Fatalf("last seq = %d, want %d", result.LastSeq, 3)
	}
	if result.Applied != 2 {
		t.Fatalf("applied = %d, want %d", result.Applied, 2)
	}
	if len(applier.seqs) != 2 || applier.seqs[0] != 2 || applier.seqs[1] != 3 {
		t.Fatalf("applied seqs = %v, want [2 3]", applier.seqs)
	}
	if checkpoints.seq != 3 {
		t.Fatalf("checkpoint seq = %d, want %d", checkpoints.seq, 3)
	}
}
