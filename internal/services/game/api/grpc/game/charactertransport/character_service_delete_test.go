package charactertransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestDeleteCharacter_Success(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecordWithCharacterCount("c1", 1)
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.deleted"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","reason":"retired"}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.DeleteCharacter(gametest.ContextWithParticipantID("manager-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Reason:      "retired",
	})
	if err != nil {
		t.Fatalf("DeleteCharacter returned error: %v", err)
	}
	if resp.Character.Id != "ch1" {
		t.Errorf("Character ID = %q, want %q", resp.Character.Id, "ch1")
	}
	if _, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1"); err == nil {
		t.Fatal("expected character to be deleted")
	}
	updatedCampaign, err := ts.Campaign.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.CharacterCount != 0 {
		t.Errorf("CharacterCount = %d, want 0", updatedCampaign.CharacterCount)
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.deleted") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.deleted"))
	}
}

func TestDeleteCharacter_DeniesMemberWhenNotOwner(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 18, 10, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecordWithCharacterCount("c1", 1)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1":     gametest.MemberParticipantRecord("c1", "member-1"),
		"member-owner": gametest.MemberParticipantRecord("c1", "member-owner"),
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
			ActorID:     "member-owner",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"member-owner"}`),
		},
	}
	ts.Event.NextSeq["c1"] = 2

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.deleted"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","reason":"retired"}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	_, err := svc.DeleteCharacter(gametest.ContextWithParticipantID("member-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Reason:      "retired",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestDeleteCharacter_RequiresDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecordWithCharacterCount("c1", 1)
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, OwnerParticipantID: "manager-1"},
	}

	svc := NewService(ts.build())
	_, err := svc.DeleteCharacter(gametest.ContextWithParticipantID("manager-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteCharacter_UsesDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Participant = characterManagerParticipantStore("c1")

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecordWithCharacterCount("c1", 1)
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.deleted"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","reason":"retired"}`),
			}),
		},
	}}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), gametest.FixedClock(now), nil)

	resp, err := svc.DeleteCharacter(gametest.ContextWithParticipantID("manager-1"), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Reason:      "retired",
	})
	if err != nil {
		t.Fatalf("DeleteCharacter returned error: %v", err)
	}
	if resp.Character.Id != "ch1" {
		t.Fatalf("Character ID = %q, want %q", resp.Character.Id, "ch1")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("character.delete") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "character.delete")
	}
	if got := len(ts.Event.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if ts.Event.Events["c1"][0].Type != event.Type("character.deleted") {
		t.Fatalf("event type = %s, want %s", ts.Event.Events["c1"][0].Type, event.Type("character.deleted"))
	}
	if _, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1"); err == nil {
		t.Fatal("expected character to be deleted")
	}
	updatedCampaign, err := ts.Campaign.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.CharacterCount != 0 {
		t.Fatalf("CharacterCount = %d, want 0", updatedCampaign.CharacterCount)
	}
}

func TestDeleteCharacter_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.DeleteCharacter(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCharacter_MissingCampaignId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCharacter_CampaignNotFound(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestDeleteCharacter_MissingCharacterId(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	svc := NewService(ts.build())
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCharacter_CharacterNotFound(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	svc := NewService(ts.build())
	_, err := svc.DeleteCharacter(context.Background(), &statev1.DeleteCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}
