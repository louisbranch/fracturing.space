package projection

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartsys "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestParseGameSystem(t *testing.T) {
	// Exact proto enum name
	sys, err := parseGameSystem("GAME_SYSTEM_DAGGERHEART")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sys != bridge.SystemIDDaggerheart {
		t.Fatalf("expected DAGGERHEART, got %v", sys)
	}

	// Uppercase shorthand
	sys, err = parseGameSystem("DAGGERHEART")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sys != bridge.SystemIDDaggerheart {
		t.Fatalf("expected DAGGERHEART, got %v", sys)
	}

	// Empty
	_, err = parseGameSystem("")
	if err == nil {
		t.Fatal("expected error for empty game system")
	}

	// Unknown
	_, err = parseGameSystem("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for unknown game system")
	}
}

func TestApplySystemEvent_MissingAdapters(t *testing.T) {
	ctx := context.Background()
	evt := testevent.Event{Type: event.Type("system.custom"), SystemID: "daggerheart", PayloadJSON: []byte("{}")}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing adapters")
	}
}

func TestApplySystemEvent_UnknownAdapter(t *testing.T) {
	ctx := context.Background()
	registry := bridge.NewAdapterRegistry()
	applier := Applier{Adapters: registry}
	evt := testevent.Event{Type: event.Type("system.custom"), SystemID: "daggerheart", PayloadJSON: []byte("{}")}
	// No adapter registered for daggerheart in this registry, should error
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing adapter")
	}
}

func TestEnsureTimestamp(t *testing.T) {
	// Non-zero timestamp should be converted to UTC
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.FixedZone("EST", -5*3600))
	result, err := ensureTimestamp(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equal(ts) {
		t.Fatalf("expected equal time, got %v", result)
	}
	if result.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", result.Location())
	}

	// Zero timestamp should return an error for replay determinism
	_, err = ensureTimestamp(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero timestamp")
	}
}

func TestApplySystemEvent_UsesAdapter(t *testing.T) {
	ctx := context.Background()
	adapter := &fakeAdapter{}
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}

	evt := testevent.Event{
		Type:          event.Type("system.custom"),
		SystemID:      "daggerheart",
		SystemVersion: "v1",
		PayloadJSON:   []byte("{}"),
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !adapter.called {
		t.Fatal("expected adapter to be called")
	}
}

func TestApplySystemEvent_UsesDaggerheartAdapterForSysPrefixedEventType(t *testing.T) {
	ctx := context.Background()
	daggerheartStore := newProjectionDaggerheartStore()
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(daggerheartsys.NewAdapter(daggerheartStore)); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{
		Adapters: registry,
	}

	payload, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{
		Value:  4,
		Reason: "test",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	evt := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys." + daggerheartsys.SystemID + ".gm_fear_changed"),
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		PayloadJSON:   payload,
	}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}

	snapshot, err := daggerheartStore.GetDaggerheartSnapshot(ctx, "camp-1")
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snapshot.GMFear != 4 {
		t.Fatalf("snapshot gm fear = %d, want %d", snapshot.GMFear, 4)
	}
}

// --- Parse helper tests ---

func TestApply_UnhandledCoreEventReturnsError(t *testing.T) {
	applier := Applier{}
	evt := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.made_up"),
		PayloadJSON: []byte("{}"),
	}
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for unhandled core event")
	}
}

// --- system routing guard tests ---

func TestRouteEvent_RejectsPartialSystemMetadata(t *testing.T) {
	applier := Applier{}
	tests := []struct {
		name          string
		systemID      string
		systemVersion string
	}{
		{"system_id_only", "system-1", ""},
		{"system_version_only", "", "v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.test.action.fired"),
				SystemID:      tt.systemID,
				SystemVersion: tt.systemVersion,
				PayloadJSON:   []byte("{}"),
			}
			err := applier.routeEvent(context.Background(), evt)
			if err == nil {
				t.Fatal("expected error for partial system metadata")
			}
			if !strings.Contains(err.Error(), "system id and version are both required") {
				t.Fatalf("expected partial metadata error, got: %v", err)
			}
		})
	}
}

// --- applySystemEvent missing branches ---

func TestApplySystemEvent_MissingSystemID(t *testing.T) {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(&fakeAdapter{}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}
	// Call applySystemEvent directly to hit the empty SystemID guard
	evt := event.Event{CampaignID: "camp-1", Type: "system.custom", SystemID: "  ", PayloadJSON: []byte("{}")}
	if err := applier.applySystemEvent(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing system_id")
	}
}

func TestApplySystemEvent_InvalidGameSystem(t *testing.T) {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(&fakeAdapter{}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}
	evt := testevent.Event{CampaignID: "camp-1", Type: event.Type("system.custom"), SystemID: "INVALID_SYSTEM", PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid game system")
	}
}

func TestApplySystemEvent_UnhandledSystemEventReturnsError(t *testing.T) {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore())); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}
	evt := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.unhandled_system_event"),
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		PayloadJSON:   []byte("{}"),
	}
	if err := applier.applySystemEvent(context.Background(), evt); err == nil {
		t.Fatal("expected error for unhandled system event")
	}
}

// --- applyCampaignForked missing branches ---

func TestMarshalOptionalMap(t *testing.T) {
	// Empty map returns nil
	result, err := marshalOptionalMap(nil)
	if err != nil || result != nil {
		t.Fatalf("expected nil, got %v, %v", result, err)
	}

	result, err = marshalOptionalMap(map[string]any{})
	if err != nil || result != nil {
		t.Fatalf("expected nil for empty map, got %v, %v", result, err)
	}

	// Non-empty map returns JSON
	result, err = marshalOptionalMap(map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestApply_SavesWatermarkOnSuccess(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	watermarks := newFakeWatermarkStore()

	applier := Applier{
		Campaign:   campaignStore,
		Watermarks: watermarks,
	}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        campaign.EventTypeCreated,
		PayloadJSON: data,
		Timestamp:   stamp,
		Seq:         5,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	wm, err := watermarks.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark: %v", err)
	}
	if wm.AppliedSeq != 5 {
		t.Fatalf("applied_seq = %d, want 5", wm.AppliedSeq)
	}
}

func TestApply_UsesInjectedNowForWatermarkTimestamp(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	watermarks := newFakeWatermarkStore()
	frozen := time.Date(2026, 2, 10, 18, 30, 0, 0, time.FixedZone("EST", -5*60*60))

	applier := Applier{
		Campaign:   campaignStore,
		Watermarks: watermarks,
		Now: func() time.Time {
			return frozen
		},
	}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        campaign.EventTypeCreated,
		PayloadJSON: data,
		Timestamp:   time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Seq:         5,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	wm, err := watermarks.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark: %v", err)
	}
	if !wm.UpdatedAt.Equal(frozen.UTC()) {
		t.Fatalf("updated_at = %v, want %v", wm.UpdatedAt, frozen.UTC())
	}
}

func TestApply_SkipsWatermarkWhenNil(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()

	// No watermarks store configured — should not panic.
	applier := Applier{Campaign: campaignStore}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        campaign.EventTypeCreated,
		PayloadJSON: data,
		Timestamp:   stamp,
		Seq:         5,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApply_SkipsWatermarkForZeroSeq(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	watermarks := newFakeWatermarkStore()

	applier := Applier{
		Campaign:   campaignStore,
		Watermarks: watermarks,
	}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        campaign.EventTypeCreated,
		PayloadJSON: data,
		Timestamp:   time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Seq:         0, // no seq — should not save watermark
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	_, err := watermarks.GetProjectionWatermark(ctx, "camp-1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for zero-seq, got %v", err)
	}
}

func TestApply_TracksExpectedNextSeqForGapDetection(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	watermarks := newFakeWatermarkStore()

	applier := Applier{
		Campaign:   campaignStore,
		Watermarks: watermarks,
	}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)

	// Apply seq 1 — no gap, expectedNextSeq should be 2.
	evt1 := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        campaign.EventTypeCreated,
		PayloadJSON: data,
		Timestamp:   time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Seq:         1,
	}
	if err := applier.Apply(ctx, eventToEvent(evt1)); err != nil {
		t.Fatalf("apply seq 1: %v", err)
	}
	wm, err := watermarks.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark: %v", err)
	}
	if wm.ExpectedNextSeq != 2 {
		t.Fatalf("expected_next_seq = %d, want 2", wm.ExpectedNextSeq)
	}

	// Apply seq 3 (skipping seq 2) — gap detected. AppliedSeq advances
	// but ExpectedNextSeq is preserved at the gap boundary so gap-detect
	// can find the mid-stream gap later.
	updatePayload, _ := json.Marshal(map[string]any{"name": "Updated"})
	evt3 := testevent.Event{
		CampaignID:  "camp-1",
		Type:        campaign.EventTypeUpdated,
		PayloadJSON: updatePayload,
		Timestamp:   time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC),
		Seq:         3,
	}
	if err := applier.Apply(ctx, eventToEvent(evt3)); err != nil {
		t.Fatalf("apply seq 3: %v", err)
	}
	wm, err = watermarks.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark after gap: %v", err)
	}
	if wm.AppliedSeq != 3 {
		t.Fatalf("applied_seq = %d, want 3", wm.AppliedSeq)
	}
	// ExpectedNextSeq stays at gap boundary (2) so gap-detect can find it.
	if wm.ExpectedNextSeq != 2 {
		t.Fatalf("expected_next_seq = %d, want 2 (preserved at gap boundary)", wm.ExpectedNextSeq)
	}
}

func TestEnsureTimestamp_ZeroReturnsError(t *testing.T) {
	_, err := ensureTimestamp(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero timestamp")
	}
	if !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("error should mention timestamp, got: %v", err)
	}
}

func TestEnsureTimestamp_NonZeroReturnsUTC(t *testing.T) {
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.FixedZone("EST", -5*60*60))
	got, err := ensureTimestamp(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", got.Location())
	}
	if !got.Equal(ts) {
		t.Fatalf("time mismatch: got %v, want %v", got, ts)
	}
}

// Scene store fakes
