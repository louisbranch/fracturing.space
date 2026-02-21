package sqlite

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

func TestApplyProjectionEventExactlyOnceSkipsDuplicateSeq(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 18, 19, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-exactly-once", now)

	payload, err := json.Marshal(participant.JoinPayload{
		ParticipantID:  "part-1",
		Name:           "Rook",
		Role:           "player",
		Controller:     "human",
		CampaignAccess: "member",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	evt := event.Event{
		CampaignID:  "camp-exactly-once",
		Seq:         42,
		Type:        event.Type("participant.joined"),
		Timestamp:   now.Add(time.Second),
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: payload,
	}

	calls := 0
	apply := func(ctx context.Context, evt event.Event, txStore *Store) error {
		calls++
		applier := projection.Applier{
			Campaign:    txStore,
			Participant: txStore,
		}
		return applier.Apply(ctx, evt)
	}

	applied, err := store.ApplyProjectionEventExactlyOnce(context.Background(), evt, apply)
	if err != nil {
		t.Fatalf("first apply exactly once: %v", err)
	}
	if !applied {
		t.Fatal("expected first apply to mutate projections")
	}

	campaignRecord, err := store.Get(context.Background(), evt.CampaignID)
	if err != nil {
		t.Fatalf("get campaign after first apply: %v", err)
	}
	if campaignRecord.ParticipantCount != 1 {
		t.Fatalf("expected participant count 1 after first apply, got %d", campaignRecord.ParticipantCount)
	}
	if calls != 1 {
		t.Fatalf("expected one apply callback invocation after first apply, got %d", calls)
	}

	applied, err = store.ApplyProjectionEventExactlyOnce(context.Background(), evt, apply)
	if err != nil {
		t.Fatalf("second apply exactly once: %v", err)
	}
	if applied {
		t.Fatal("expected second apply to be skipped as duplicate")
	}
	if calls != 1 {
		t.Fatalf("expected duplicate apply to skip callback, got %d calls", calls)
	}

	campaignRecord, err = store.Get(context.Background(), evt.CampaignID)
	if err != nil {
		t.Fatalf("get campaign after duplicate apply: %v", err)
	}
	if campaignRecord.ParticipantCount != 1 {
		t.Fatalf("expected participant count to remain 1 after duplicate apply, got %d", campaignRecord.ParticipantCount)
	}
}

func TestApplyProjectionEventExactlyOnceConcurrentDuplicateSeq(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 18, 19, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-exactly-once-concurrent", now)

	evt := event.Event{
		CampaignID:  "camp-exactly-once-concurrent",
		Seq:         43,
		Type:        event.Type("participant.joined"),
		Timestamp:   now.Add(time.Second),
		EntityType:  "participant",
		EntityID:    "part-1",
		PayloadJSON: []byte(`{"participant_id":"part-1","name":"Rook","role":"player","controller":"human","campaign_access":"member"}`),
	}

	var callbackCalls atomic.Int32
	apply := func(_ context.Context, _ event.Event, _ *Store) error {
		callbackCalls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	type result struct {
		applied bool
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, 2)

	run := func() {
		<-start
		applied, err := store.ApplyProjectionEventExactlyOnce(context.Background(), evt, apply)
		results <- result{applied: applied, err: err}
	}

	go run()
	go run()
	close(start)

	first := <-results
	second := <-results

	if first.err != nil {
		t.Fatalf("first concurrent apply returned error: %v", first.err)
	}
	if second.err != nil {
		t.Fatalf("second concurrent apply returned error: %v", second.err)
	}

	appliedCount := 0
	if first.applied {
		appliedCount++
	}
	if second.applied {
		appliedCount++
	}
	if appliedCount != 1 {
		t.Fatalf("expected exactly one concurrent apply to mutate projections, got %d", appliedCount)
	}

	if got := callbackCalls.Load(); got != 1 {
		t.Fatalf("expected apply callback to run once, got %d", got)
	}
}
