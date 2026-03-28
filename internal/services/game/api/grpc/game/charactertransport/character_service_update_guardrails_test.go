package charactertransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUpdateCharacter_NoFields(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-owner", Name: "Hero", Kind: character.KindPC},
	}

	svc := NewService(ts.build())
	_, err := svc.UpdateCharacter(context.Background(), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCharacter_DeniesMemberWhenNotOwner(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 18, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1":     gametest.MemberParticipantRecord("c1", "member-1"),
		"member-owner": gametest.MemberParticipantRecord("c1", "member-owner"),
	}
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", OwnerParticipantID: "member-owner", Name: "Hero", Kind: character.KindPC},
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
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "member-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"name":"Renamed"}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	_, err := svc.UpdateCharacter(requestctx.WithParticipantID("member-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestUpdateCharacter_DeniesControllerWhenOwnershipUnresolved(t *testing.T) {
	// A member whose participant ID matches ParticipantID (controller) but has
	// no ownership events should be denied; controller is not owner.
	ts := newTestStores().withCharacter()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"controller-1": gametest.MemberParticipantRecord("c1", "controller-1"),
	}
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, ParticipantID: "controller-1"},
	}
	ts.Event.NextSeq["c1"] = 1

	domain := &fakeDomainEngine{store: ts.Event}
	svc := NewService(ts.withDomain(domain).build())
	_, err := svc.UpdateCharacter(requestctx.WithParticipantID("controller-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("Renamed"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestUpdateCharacter_DeniesManagerOwnershipTransfer(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 19, 35, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
		"member-1":  gametest.MemberParticipantRecord("c1", "member-1"),
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
			ActorID:     "manager-1",
			EntityType:  "character",
			EntityID:    "ch1",
			PayloadJSON: []byte(`{"character_id":"ch1","name":"Hero","kind":"pc","owner_participant_id":"manager-1"}`),
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
				ActorID:     "manager-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"owner_participant_id":"member-1"}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	_, err := svc.UpdateCharacter(requestctx.WithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
		CampaignId:         "c1",
		CharacterId:        "ch1",
		OwnerParticipantId: wrapperspb.String("member-1"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestUpdateCharacter_RequiresDomainEngine(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant = characterManagerParticipantStore("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, OwnerParticipantID: "manager-1"},
	}

	svc := NewService(ts.build())
	_, err := svc.UpdateCharacter(requestctx.WithParticipantID("manager-1"), &statev1.UpdateCharacterRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		Name:        wrapperspb.String("New Hero"),
	})
	assertStatusCode(t, err, codes.Internal)
}
