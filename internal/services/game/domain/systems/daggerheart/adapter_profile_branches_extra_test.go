package daggerheart

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestApplyCharacterProfileDeleted_DeleteError(t *testing.T) {
	store := newFaultDaggerheartStore()
	store.deleteCharacterProfileErr = errText("delete profile failed")
	adapter := NewAdapter(store)

	err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileDeleted,
		PayloadJSON:   []byte(`{"character_id":"char-1"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "delete daggerheart profile: delete profile failed") {
		t.Fatalf("Apply(delete) error = %v, want wrapped delete error", err)
	}
}

func TestApplyCharacterProfileReplaced_InvalidProfileValidationError(t *testing.T) {
	store := newFaultDaggerheartStore()
	adapter := NewAdapter(store)

	payload, _ := json.Marshal(CharacterProfileReplacedPayload{
		CharacterID: ids.CharacterID("char-1"),
		Profile: CharacterProfile{
			Level:           0,
			HpMax:           -1,
			StressMax:       0,
			Evasion:         0,
			MajorThreshold:  0,
			SevereThreshold: 0,
			Proficiency:     0,
			ArmorScore:      0,
			ArmorMax:        0,
		},
	})
	err := adapter.Apply(context.Background(), event.Event{
		CampaignID:    ids.CampaignID("camp-1"),
		EntityID:      "char-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		Type:          EventTypeCharacterProfileReplaced,
		PayloadJSON:   payload,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validate daggerheart character profile") {
		t.Fatalf("Apply() error = %v, want validation prefix", err)
	}
}

type errText string

func (e errText) Error() string { return string(e) }
