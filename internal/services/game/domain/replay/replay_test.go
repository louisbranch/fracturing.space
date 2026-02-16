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
	seq uint64
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
	s.seq = checkpoint.LastSeq
	return nil
}

type recordingApplier struct {
	seqs []uint64
}

func (a *recordingApplier) Apply(state any, evt event.Event) (any, error) {
	a.seqs = append(a.seqs, evt.Seq)
	return state, nil
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
