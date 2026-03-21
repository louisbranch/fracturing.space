package charactertransport

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUpdateCharacter_AllowsMemberWhenOwnershipTransferred(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 19, 30, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1":     gametest.MemberParticipantRecord("c1", "member-1"),
		"member-owner": gametest.MemberParticipantRecord("c1", "member-owner"),
	}
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-1", Name: "Hero", Kind: character.KindPC},
	}
	ts.Event.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
		{
			Seq:         2,
			CampaignID:  "c1",
			Type:        event.Type("character.updated"),
			Timestamp:   now.Add(time.Minute),
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","fields":{"owner_participant_id":"member-1"}}`),
		},
	}
	ts.Event.NextSeq["c1"] = 3

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"Renamed"}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.UpdateCharacter(gametest.ContextWithParticipantID("member-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.GetCharacter().GetName() != "Renamed" {
		t.Fatalf("character name = %q, want %q", resp.GetCharacter().GetName(), "Renamed")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestUpdateCharacter_AllowsOwnerOwnershipTransfer(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 19, 40, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":  gametest.OwnerParticipantRecord("c1", "owner-1"),
		"member-1": gametest.MemberParticipantRecord("c1", "member-1"),
	}
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	ts.Event.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"owner-1"}`),
		},
	}
	ts.Event.NextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now.Add(time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"owner_participant_id":"member-1"}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.UpdateCharacter(gametest.ContextWithParticipantID("owner-1"), &statev1.UpdateCharacterRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		OwnerParticipantId: wrapperspb.String("member-1"),
	})
	if err != nil {
		t.Fatalf("UpdateCharacter returned error: %v", err)
	}
	if resp.GetCharacter().GetId() != "ch1" {
		t.Fatalf("character id = %q, want %q", resp.GetCharacter().GetId(), "ch1")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("domain command count = %d, want 1", len(domain.commands))
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["owner_participant_id"] != "member-1" {
		t.Fatalf("owner_participant_id = %q, want %q", payload.Fields["owner_participant_id"], "member-1")
	}
}
