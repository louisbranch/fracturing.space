package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// fakeEventHighWaterStore provides GetLatestEventSeq for gap detection tests.
type fakeEventHighWaterStore struct {
	seqs map[string]uint64
}

func (s *fakeEventHighWaterStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	seq, ok := s.seqs[campaignID]
	if !ok {
		return 0, nil
	}
	return seq, nil
}

func TestDetectProjectionGaps_FindsGaps(t *testing.T) {
	watermarks := newFakeWatermarkStore()
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Camp-1 has watermark at 5 but journal at 10 → gap.
	_ = watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-1",
		AppliedSeq: 5,
		UpdatedAt:  now,
	})
	// Camp-2 is up to date.
	_ = watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-2",
		AppliedSeq: 10,
		UpdatedAt:  now,
	})

	events := &fakeEventHighWaterStore{
		seqs: map[string]uint64{
			"camp-1": 10,
			"camp-2": 10,
		},
	}

	gaps, err := DetectProjectionGaps(ctx, watermarks, events)
	if err != nil {
		t.Fatalf("detect gaps: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if gaps[0].CampaignID != "camp-1" {
		t.Fatalf("gap campaign = %s, want camp-1", gaps[0].CampaignID)
	}
	if gaps[0].WatermarkSeq != 5 {
		t.Fatalf("watermark = %d, want 5", gaps[0].WatermarkSeq)
	}
	if gaps[0].JournalSeq != 10 {
		t.Fatalf("journal = %d, want 10", gaps[0].JournalSeq)
	}
}

func TestDetectProjectionGaps_NoGaps(t *testing.T) {
	watermarks := newFakeWatermarkStore()
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_ = watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-1",
		AppliedSeq: 10,
		UpdatedAt:  now,
	})

	events := &fakeEventHighWaterStore{
		seqs: map[string]uint64{"camp-1": 10},
	}

	gaps, err := DetectProjectionGaps(ctx, watermarks, events)
	if err != nil {
		t.Fatalf("detect gaps: %v", err)
	}
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestDetectProjectionGaps_EmptyWatermarks(t *testing.T) {
	watermarks := newFakeWatermarkStore()
	events := &fakeEventHighWaterStore{}

	gaps, err := DetectProjectionGaps(context.Background(), watermarks, events)
	if err != nil {
		t.Fatalf("detect gaps: %v", err)
	}
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(gaps))
	}
}

// gapEventStore is a fake that implements storage.EventStore with configurable
// high-water marks and event lists, suitable for gap repair tests.
type gapEventStore struct {
	seqs   map[string]uint64
	events []event.Event
}

func (s *gapEventStore) AppendEvent(context.Context, event.Event) (event.Event, error) {
	return event.Event{}, nil
}
func (s *gapEventStore) GetEventByHash(context.Context, string) (event.Event, error) {
	return event.Event{}, nil
}
func (s *gapEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	return event.Event{}, nil
}
func (s *gapEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	var results []event.Event
	for _, evt := range s.events {
		if evt.CampaignID != campaignID || evt.Seq <= afterSeq {
			continue
		}
		results = append(results, evt)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}
func (s *gapEventStore) ListEventsBySession(context.Context, string, string, uint64, int) ([]event.Event, error) {
	return nil, nil
}
func (s *gapEventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	return s.seqs[campaignID], nil
}
func (s *gapEventStore) ListEventsPage(context.Context, storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, nil
}

func TestRepairProjectionGaps_ReplaysGaps(t *testing.T) {
	ctx := context.Background()
	watermarks := newFakeWatermarkStore()
	now := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)

	// Camp-1: watermark at seq 1, journal at seq 2 → gap.
	_ = watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-1", AppliedSeq: 1, UpdatedAt: now,
	})

	// Seed the campaign in the campaign store (needed for projection apply).
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{
		ID: "camp-1", Status: campaign.StatusActive,
	}

	// The gap event: a campaign.updated at seq 2.
	payload, _ := json.Marshal(testevent.CampaignUpdatedPayload{
		Fields: map[string]any{"name": "Repaired"},
	})
	gapEvent := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.updated"),
		Seq:         2,
		Timestamp:   now,
		PayloadJSON: payload,
	}

	eventStore := &gapEventStore{
		seqs:   map[string]uint64{"camp-1": 2},
		events: []event.Event{gapEvent},
	}

	applier := Applier{Campaign: campaignStore}
	results, err := RepairProjectionGaps(ctx, watermarks, eventStore, applier)
	if err != nil {
		t.Fatalf("repair gaps: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].CampaignID != "camp-1" {
		t.Fatalf("result campaign = %s, want camp-1", results[0].CampaignID)
	}
	if results[0].EventsReplayed != 1 {
		t.Fatalf("events replayed = %d, want 1", results[0].EventsReplayed)
	}

	// Verify the projection was actually updated.
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.Name != "Repaired" {
		t.Fatalf("campaign name = %q, want %q", c.Name, "Repaired")
	}
}

func TestRepairProjectionGaps_NoGaps(t *testing.T) {
	ctx := context.Background()
	watermarks := newFakeWatermarkStore()
	now := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)

	_ = watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID: "camp-1", AppliedSeq: 5, UpdatedAt: now,
	})
	eventStore := &gapEventStore{
		seqs: map[string]uint64{"camp-1": 5},
	}

	results, err := RepairProjectionGaps(ctx, watermarks, eventStore, Applier{})
	if err != nil {
		t.Fatalf("repair gaps: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
