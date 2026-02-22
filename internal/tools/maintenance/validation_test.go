package maintenance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func testEventRegistry(t *testing.T) *event.Registry {
	t.Helper()
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	return registries.Events
}

func TestIsSnapshotEvent(t *testing.T) {
	tests := []struct {
		name     string
		systemID string
		want     bool
	}{
		{"empty system id", "", false},
		{"whitespace only", "   ", false},
		{"daggerheart", "daggerheart", true},
		{"any system", "custom", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evt := event.Event{SystemID: tc.systemID}
			if got := isSnapshotEvent(evt); got != tc.want {
				t.Errorf("isSnapshotEvent(systemID=%q) = %v, want %v", tc.systemID, got, tc.want)
			}
		})
	}
}

func TestValidateSnapshotEvent_CharacterStatePatched(t *testing.T) {
	registry := testEventRegistry(t)
	makeEvent := func(payload daggerheart.CharacterStatePatchedPayload) event.Event {
		data, _ := json.Marshal(payload)
		return event.Event{
			CampaignID:    "camp-1",
			Type:          event.Type("sys.daggerheart.character_state_patched"),
			Timestamp:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			EntityType:    "action",
			EntityID:      "entity-1",
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   data,
		}
	}

	t.Run("valid", func(t *testing.T) {
		hpBefore := 5
		hpAfter := 4
		evt := makeEvent(daggerheart.CharacterStatePatchedPayload{
			CharacterID: "char-1",
			HPBefore:    &hpBefore,
			HPAfter:     &hpAfter,
		})
		if err := validateSnapshotEvent(registry, evt); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing character id", func(t *testing.T) {
		evt := makeEvent(daggerheart.CharacterStatePatchedPayload{})
		if err := validateSnapshotEvent(registry, evt); err == nil {
			t.Error("expected error for missing character id")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		evt := event.Event{
			CampaignID:    "camp-1",
			Type:          event.Type("sys.daggerheart.character_state_patched"),
			Timestamp:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			EntityType:    "action",
			EntityID:      "entity-1",
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   []byte(`{invalid`),
		}
		if err := validateSnapshotEvent(registry, evt); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestValidateSnapshotEvent_GMFearChanged(t *testing.T) {
	registry := testEventRegistry(t)
	makeEvent := func(payload daggerheart.GMFearChangedPayload) event.Event {
		data, _ := json.Marshal(payload)
		return event.Event{
			CampaignID:    "camp-1",
			Type:          event.Type("sys.daggerheart.gm_fear_changed"),
			Timestamp:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			EntityType:    "action",
			EntityID:      "entity-1",
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   data,
		}
	}

	t.Run("valid", func(t *testing.T) {
		evt := makeEvent(daggerheart.GMFearChangedPayload{After: 3})
		if err := validateSnapshotEvent(registry, evt); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		evt := makeEvent(daggerheart.GMFearChangedPayload{After: daggerheart.GMFearMax + 1})
		if err := validateSnapshotEvent(registry, evt); err == nil {
			t.Error("expected error for out of range fear")
		}
	})
}

func TestValidateSnapshotEvent_UnknownType(t *testing.T) {
	registry := testEventRegistry(t)
	evt := event.Event{CampaignID: "camp-1", Type: "unknown.event", Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), PayloadJSON: []byte("{}")}
	if err := validateSnapshotEvent(registry, evt); err != nil {
		t.Errorf("unknown event type should not error: %v", err)
	}
}
