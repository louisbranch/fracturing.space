package eventtransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestListTimelineEntries_CharacterStateChanges(t *testing.T) {
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()

	now := time.Now().UTC()
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Frodo", Kind: character.KindPC},
	}

	hp := 6
	hope := 2
	hopeMax := 6
	stress := 0
	armor := 0
	lifeState := "alive"
	payload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		HopeMax:     &hopeMax,
		Stress:      &stress,
		Armor:       &armor,
		LifeState:   &lifeState,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	eventStore.Events["c1"] = []event.Event{{
		CampaignID:  "c1",
		Seq:         1,
		Type:        event.Type("sys.daggerheart.character_state_patched"),
		EntityType:  "character",
		EntityID:    "ch1",
		Timestamp:   now,
		PayloadJSON: payloadJSON,
	}}

	svc := NewService(Deps{Event: eventStore, Character: characterStore})
	resp, err := svc.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		OrderBy:    "seq",
	})
	if err != nil {
		t.Fatalf("list timeline entries: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}

	fields := resp.Entries[0].GetProjection().GetFields()
	if len(fields) == 0 {
		t.Fatal("expected change fields")
	}
	fieldMap := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldMap[field.GetLabel()] = field.GetValue()
	}
	assertField := func(label, value string) {
		t.Helper()
		if got := fieldMap[label]; got != value {
			t.Fatalf("field %q = %q, want %q", label, got, value)
		}
	}
	assertField("HP", "= 6")
	assertField("Hope", "= 2")
	assertField("Hope Max", "= 6")
	assertField("Stress", "= 0")
	assertField("Armor", "= 0")
	assertField("Life State", "= alive")
}

func TestListTimelineEntries_CharacterStateChanges_WithBefore(t *testing.T) {
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()

	now := time.Now().UTC()
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Frodo", Kind: character.KindPC},
	}

	hp := 6
	hope := 2
	hopeMax := 7
	stress := 1
	armor := 2
	lifeState := "dying"
	payload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		HopeMax:     &hopeMax,
		Stress:      &stress,
		Armor:       &armor,
		LifeState:   &lifeState,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	eventStore.Events["c1"] = []event.Event{{
		CampaignID:  "c1",
		Seq:         1,
		Type:        event.Type("sys.daggerheart.character_state_patched"),
		EntityType:  "character",
		EntityID:    "ch1",
		Timestamp:   now,
		PayloadJSON: payloadJSON,
	}}

	svc := NewService(Deps{Event: eventStore, Character: characterStore})
	resp, err := svc.ListTimelineEntries(requestctx.WithAdminOverride(context.Background(), "timeline-test"), &campaignv1.ListTimelineEntriesRequest{
		CampaignId: "c1",
		OrderBy:    "seq",
	})
	if err != nil {
		t.Fatalf("list timeline entries: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}

	fields := resp.Entries[0].GetProjection().GetFields()
	if len(fields) == 0 {
		t.Fatal("expected change fields")
	}
	fieldMap := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldMap[field.GetLabel()] = field.GetValue()
	}
	assertField := func(label, value string) {
		t.Helper()
		if got := fieldMap[label]; got != value {
			t.Fatalf("field %q = %q, want %q", label, got, value)
		}
	}
	assertField("HP", "= 6")
	assertField("Hope", "= 2")
	assertField("Hope Max", "= 7")
	assertField("Stress", "= 1")
	assertField("Armor", "= 2")
	assertField("Life State", "= dying")
}
