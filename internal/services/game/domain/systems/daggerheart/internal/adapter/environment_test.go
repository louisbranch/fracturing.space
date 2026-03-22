package adapter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestEnvironmentEntityHandlersPersistProjectionState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	createdAt := time.Date(2026, time.March, 20, 9, 0, 0, 0, time.UTC)
	if err := a.HandleEnvironmentEntityCreated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		Timestamp:  createdAt,
	}, payload.EnvironmentEntityCreatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID(" env-1 "),
		EnvironmentID:       " environment.ruins ",
		Name:                " Falling Pillar ",
		Type:                " hazard ",
		Tier:                2,
		Difficulty:          14,
		SessionID:           ids.SessionID(" sess-1 "),
		SceneID:             ids.SceneID(" scene-1 "),
		Notes:               " unstable ",
	}); err != nil {
		t.Fatalf("HandleEnvironmentEntityCreated() returned error: %v", err)
	}

	got := store.environmentEntities[profileKey("camp-1", "env-1")]
	if got.EnvironmentID != "environment.ruins" || got.Name != "Falling Pillar" || got.Type != "hazard" {
		t.Fatalf("created environment entity = %+v, want trimmed fields", got)
	}
	if got.SessionID != "sess-1" || got.SceneID != "scene-1" || got.Notes != "unstable" {
		t.Fatalf("created environment entity = %+v, want trimmed scene metadata", got)
	}
	if got.CreatedAt != createdAt || got.UpdatedAt != createdAt {
		t.Fatalf("created timestamps = (%s, %s), want both %s", got.CreatedAt, got.UpdatedAt, createdAt)
	}

	updatedAt := createdAt.Add(90 * time.Minute)
	if err := a.HandleEnvironmentEntityUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
		Timestamp:  updatedAt,
	}, payload.EnvironmentEntityUpdatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID(" env-1 "),
		EnvironmentID:       " environment.keep ",
		Name:                " Arcane Lens ",
		Type:                " feature ",
		Tier:                3,
		Difficulty:          17,
		SessionID:           ids.SessionID(" sess-2 "),
		SceneID:             ids.SceneID(" scene-2 "),
		Notes:               " attuned ",
	}); err != nil {
		t.Fatalf("HandleEnvironmentEntityUpdated() returned error: %v", err)
	}

	got = store.environmentEntities[profileKey("camp-1", "env-1")]
	if got.CreatedAt != createdAt || got.UpdatedAt != updatedAt.UTC() {
		t.Fatalf("updated timestamps = (%s, %s), want created preserved and updated refreshed", got.CreatedAt, got.UpdatedAt)
	}
	if got.EnvironmentID != "environment.keep" || got.Name != "Arcane Lens" || got.Type != "feature" {
		t.Fatalf("updated environment entity = %+v, want replaced trimmed fields", got)
	}

	if err := a.HandleEnvironmentEntityDeleted(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EnvironmentEntityDeletedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID(" env-1 "),
	}); err != nil {
		t.Fatalf("HandleEnvironmentEntityDeleted() returned error: %v", err)
	}
	if _, ok := store.environmentEntities[profileKey("camp-1", "env-1")]; ok {
		t.Fatal("environment entity still present after delete")
	}
}

func TestEnvironmentEntityHandlersPropagateStoreErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	store.putEnvironmentEntityErr = errors.New("write failed")
	if err := a.HandleEnvironmentEntityCreated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EnvironmentEntityCreatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
	}); err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("HandleEnvironmentEntityCreated() error = %v, want write error", err)
	}
	store.putEnvironmentEntityErr = nil

	if err := a.HandleEnvironmentEntityUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EnvironmentEntityUpdatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("missing"),
	}); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("HandleEnvironmentEntityUpdated() error = %v, want storage.ErrNotFound", err)
	}

	store.environmentEntities[profileKey("camp-1", "env-1")] = projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          "camp-1",
		EnvironmentEntityID: "env-1",
	}
	store.getEnvironmentEntityErr = errors.New("read failed")
	if err := a.HandleEnvironmentEntityUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EnvironmentEntityUpdatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
	}); err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("HandleEnvironmentEntityUpdated() get error = %v, want get error", err)
	}
	store.getEnvironmentEntityErr = nil

	store.putEnvironmentEntityErr = errors.New("write failed")
	if err := a.HandleEnvironmentEntityUpdated(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EnvironmentEntityUpdatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
	}); err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("HandleEnvironmentEntityUpdated() put error = %v, want put error", err)
	}
	store.putEnvironmentEntityErr = nil

	store.deleteEnvironmentEntityErr = errors.New("delete failed")
	if err := a.HandleEnvironmentEntityDeleted(ctx, event.Event{
		CampaignID: ids.CampaignID("camp-1"),
	}, payload.EnvironmentEntityDeletedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
	}); err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("HandleEnvironmentEntityDeleted() error = %v, want delete error", err)
	}
}
