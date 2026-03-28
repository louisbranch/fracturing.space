package replay

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type accumulationState struct {
	Count        int
	SeqSum       uint64
	TypeBytes    int
	PayloadBytes int
}

type accumulationFolder struct{}

func (accumulationFolder) Fold(state any, evt event.Event) (any, error) {
	current, ok := state.(accumulationState)
	if !ok {
		return nil, ErrFolderRequired
	}
	current.Count++
	current.SeqSum += evt.Seq
	current.TypeBytes += len(evt.Type)
	current.PayloadBytes += len(evt.PayloadJSON)
	return current, nil
}

func FuzzReplay_PageSizeAndCheckpointIntervalPreserveResult(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4}, uint8(0), uint8(0), uint8(0), uint8(0), true)
	f.Add([]byte("event-sourcing"), uint8(3), uint8(2), uint8(4), uint8(7), false)
	f.Add([]byte{9, 8, 7, 6, 5, 4, 3}, uint8(1), uint8(5), uint8(2), uint8(3), true)

	f.Fuzz(func(t *testing.T, raw []byte, pageSizeSeed, checkpointSeed, afterSeed, checkpointStartSeed uint8, useUntil bool) {
		events := replayEventsFromBytes(raw)
		maxSeq := uint64(len(events))

		afterSeq := boundedSeq(afterSeed, maxSeq)
		checkpointSeq := boundedSeq(checkpointStartSeed, maxSeq)
		untilSeq := uint64(0)
		if useUntil && maxSeq > 0 {
			start := afterSeq
			if checkpointSeq > start {
				start = checkpointSeq
			}
			untilSeq = start + boundedSeq(uint8(len(raw)), maxSeq-start)
		}

		fixedClock := func() time.Time {
			return time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
		}

		runReplay := func(pageSize, checkpointInterval int) (Result, *fakeCheckpointStore) {
			t.Helper()
			checkpoints := &fakeCheckpointStore{seq: checkpointSeq}
			result, err := Replay(
				context.Background(),
				&fakeEventStore{events: events},
				checkpoints,
				accumulationFolder{},
				"camp-1",
				accumulationState{},
				Options{
					AfterSeq:           afterSeq,
					UntilSeq:           untilSeq,
					PageSize:           pageSize,
					CheckpointInterval: checkpointInterval,
					Clock:              fixedClock,
				},
			)
			if err != nil {
				t.Fatalf("Replay() error = %v", err)
			}
			return result, checkpoints
		}

		baseline, baselineCheckpoints := runReplay(1, 1)
		variantPageSize := int(pageSizeSeed % 7)
		variantCheckpointInterval := int(checkpointSeed % 7)
		variant, variantCheckpoints := runReplay(variantPageSize, variantCheckpointInterval)

		if baseline.Applied != variant.Applied {
			t.Fatalf("applied mismatch: baseline=%d variant=%d", baseline.Applied, variant.Applied)
		}
		if baseline.LastSeq != variant.LastSeq {
			t.Fatalf("last seq mismatch: baseline=%d variant=%d", baseline.LastSeq, variant.LastSeq)
		}
		if !reflect.DeepEqual(baseline.State, variant.State) {
			t.Fatalf("state mismatch: baseline=%#v variant=%#v", baseline.State, variant.State)
		}

		assertCheckpointMatchesResult(t, baselineCheckpoints, baseline, fixedClock())
		assertCheckpointMatchesResult(t, variantCheckpoints, variant, fixedClock())
	})
}

func replayEventsFromBytes(raw []byte) []event.Event {
	if len(raw) > 32 {
		raw = raw[:32]
	}
	events := make([]event.Event, 0, len(raw))
	for i, b := range raw {
		events = append(events, event.Event{
			CampaignID:  "camp-1",
			Seq:         uint64(i + 1),
			Type:        event.Type("replay.test." + string('a'+rune(b%26))),
			PayloadJSON: []byte{b, byte(i)},
		})
	}
	return events
}

func boundedSeq(seed uint8, max uint64) uint64 {
	if max == 0 {
		return 0
	}
	return uint64(seed) % (max + 1)
}

func assertCheckpointMatchesResult(t *testing.T, checkpoints *fakeCheckpointStore, result Result, now time.Time) {
	t.Helper()
	if checkpoints == nil {
		t.Fatal("checkpoint store is required")
	}
	if result.Applied == 0 {
		if checkpoints.saveCalls != 0 {
			t.Fatalf("checkpoint save calls = %d, want 0 when no events applied", checkpoints.saveCalls)
		}
		return
	}
	if checkpoints.seq != result.LastSeq {
		t.Fatalf("checkpoint seq = %d, want %d", checkpoints.seq, result.LastSeq)
	}
	if checkpoints.lastSaved.LastSeq != result.LastSeq {
		t.Fatalf("last saved seq = %d, want %d", checkpoints.lastSaved.LastSeq, result.LastSeq)
	}
	if !checkpoints.lastSaved.UpdatedAt.Equal(now) {
		t.Fatalf("checkpoint updated_at = %v, want %v", checkpoints.lastSaved.UpdatedAt, now)
	}
}
