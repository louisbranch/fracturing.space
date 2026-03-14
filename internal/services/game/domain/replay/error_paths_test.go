package replay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type replayStoreStub struct {
	events    []event.Event
	listErr   error
	listCalls int
}

func (s *replayStoreStub) ListEvents(_ context.Context, _ string, afterSeq uint64, limit int) ([]event.Event, error) {
	s.listCalls++
	if s.listErr != nil {
		return nil, s.listErr
	}
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

type replayCheckpointStub struct {
	getErr    error
	seq       uint64
	saveErr   error
	saveCalls int
	lastSaved Checkpoint
}

func (s *replayCheckpointStub) Get(_ context.Context, campaignID string) (Checkpoint, error) {
	if s.getErr != nil {
		return Checkpoint{}, s.getErr
	}
	if s.seq == 0 {
		return Checkpoint{}, ErrCheckpointNotFound
	}
	return Checkpoint{CampaignID: campaignID, LastSeq: s.seq, UpdatedAt: time.Now().UTC()}, nil
}

func (s *replayCheckpointStub) Save(_ context.Context, checkpoint Checkpoint) error {
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.lastSaved = checkpoint
	s.seq = checkpoint.LastSeq
	return nil
}

type replayFolderStub struct {
	foldErr error
	calls   int
}

func (s *replayFolderStub) Fold(state any, _ event.Event) (any, error) {
	s.calls++
	if s.foldErr != nil {
		return nil, s.foldErr
	}
	return state, nil
}

func TestReplay_RequiresDependenciesAndCampaignID(t *testing.T) {
	checkpoints := &replayCheckpointStub{}
	folder := &replayFolderStub{}
	store := &replayStoreStub{}

	if _, err := Replay(context.Background(), nil, checkpoints, folder, "camp-1", nil, Options{}); !errors.Is(err, ErrEventStoreRequired) {
		t.Fatalf("expected ErrEventStoreRequired, got %v", err)
	}
	if _, err := Replay(context.Background(), store, nil, folder, "camp-1", nil, Options{}); !errors.Is(err, ErrCheckpointStoreRequired) {
		t.Fatalf("expected ErrCheckpointStoreRequired, got %v", err)
	}
	if _, err := Replay(context.Background(), store, checkpoints, nil, "camp-1", nil, Options{}); !errors.Is(err, ErrFolderRequired) {
		t.Fatalf("expected ErrFolderRequired, got %v", err)
	}
	if _, err := Replay(context.Background(), store, checkpoints, folder, "   ", nil, Options{}); !errors.Is(err, ErrCampaignIDRequired) {
		t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
	}
}

func TestReplay_PropagatesCheckpointGetError(t *testing.T) {
	expected := errors.New("checkpoint load failed")
	_, err := Replay(
		context.Background(),
		&replayStoreStub{},
		&replayCheckpointStub{getErr: expected},
		&replayFolderStub{},
		"camp-1",
		nil,
		Options{},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected checkpoint load error, got %v", err)
	}
}

func TestReplay_PropagatesListEventsError(t *testing.T) {
	expected := errors.New("list events failed")
	_, err := Replay(
		context.Background(),
		&replayStoreStub{listErr: expected},
		&replayCheckpointStub{},
		&replayFolderStub{},
		"camp-1",
		nil,
		Options{},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected list error, got %v", err)
	}
}

func TestReplay_PropagatesFoldError(t *testing.T) {
	expected := errors.New("fold failed")
	_, err := Replay(
		context.Background(),
		&replayStoreStub{events: []event.Event{{CampaignID: "camp-1", Seq: 1}}},
		&replayCheckpointStub{},
		&replayFolderStub{foldErr: expected},
		"camp-1",
		nil,
		Options{},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected fold error, got %v", err)
	}
}

func TestReplay_PropagatesCheckpointSaveError(t *testing.T) {
	expected := errors.New("save failed")
	_, err := Replay(
		context.Background(),
		&replayStoreStub{events: []event.Event{{CampaignID: "camp-1", Seq: 1}}},
		&replayCheckpointStub{saveErr: expected},
		&replayFolderStub{},
		"camp-1",
		nil,
		Options{CheckpointInterval: 1},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected checkpoint save error, got %v", err)
	}
}

func TestReplay_DetectsSequenceGap(t *testing.T) {
	_, err := Replay(
		context.Background(),
		&replayStoreStub{events: []event.Event{
			{CampaignID: "camp-1", Seq: 1},
			{CampaignID: "camp-1", Seq: 3},
		}},
		&replayCheckpointStub{},
		&replayFolderStub{},
		"camp-1",
		nil,
		Options{},
	)
	if err == nil {
		t.Fatal("expected sequence gap error")
	}
	if got := err.Error(); got != "event sequence gap: expected 2 got 3" {
		t.Fatalf("sequence gap error = %q, want %q", got, "event sequence gap: expected 2 got 3")
	}
}

func TestReplay_UntilSeqStopsAndFlushesCheckpoint(t *testing.T) {
	checkpoints := &replayCheckpointStub{}
	now := time.Date(2026, 3, 9, 21, 15, 0, 0, time.UTC)
	result, err := Replay(
		context.Background(),
		&replayStoreStub{events: []event.Event{
			{CampaignID: "camp-1", Seq: 1},
			{CampaignID: "camp-1", Seq: 2},
			{CampaignID: "camp-1", Seq: 3},
		}},
		checkpoints,
		&replayFolderStub{},
		"camp-1",
		"state",
		Options{UntilSeq: 1, CheckpointInterval: 10, Clock: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatalf("Replay() unexpected error: %v", err)
	}
	if result.Applied != 1 || result.LastSeq != 1 {
		t.Fatalf("result = %+v, want applied=1 last_seq=1", result)
	}
	if checkpoints.saveCalls != 1 {
		t.Fatalf("checkpoint save calls = %d, want 1", checkpoints.saveCalls)
	}
	if checkpoints.lastSaved.LastSeq != 1 {
		t.Fatalf("checkpoint last seq = %d, want 1", checkpoints.lastSaved.LastSeq)
	}
	if !checkpoints.lastSaved.UpdatedAt.Equal(now) {
		t.Fatalf("checkpoint updated_at = %v, want %v", checkpoints.lastSaved.UpdatedAt, now)
	}
}

func TestReplay_PropagatesFinalCheckpointSaveError(t *testing.T) {
	expected := errors.New("final save failed")
	_, err := Replay(
		context.Background(),
		&replayStoreStub{events: []event.Event{{CampaignID: "camp-1", Seq: 1}}},
		&replayCheckpointStub{saveErr: expected},
		&replayFolderStub{},
		"camp-1",
		nil,
		Options{CheckpointInterval: 10},
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected final checkpoint save error, got %v", err)
	}
}
