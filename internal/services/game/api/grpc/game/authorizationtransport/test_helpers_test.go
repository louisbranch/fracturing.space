package authorizationtransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func newAuthorizationServiceFixture(t *testing.T) *AuthorizationService {
	t.Helper()

	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{ID: "c1"}

	participantStore := gametest.NewFakeParticipantStore()
	for _, record := range []storage.ParticipantRecord{
		{ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner, UserID: "owner-user"},
		{ID: "manager-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessManager, UserID: "manager-user"},
		{ID: "member-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessMember, UserID: "member-user"},
	} {
		if err := participantStore.PutParticipant(context.Background(), record); err != nil {
			t.Fatalf("put participant: %v", err)
		}
	}

	eventStore := gametest.NewFakeEventStore()
	if _, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "c1",
		Type:        handler.EventTypeCharacterCreated,
		EntityType:  "character",
		EntityID:    "char-member-1",
		ActorType:   event.ActorTypeParticipant,
		ActorID:     "member-1",
		PayloadJSON: []byte(`{"character_id":"char-member-1","owner_participant_id":"member-1","name":"Member Hero","kind":"pc"}`),
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	characterStore := gametest.NewFakeCharacterStore()
	if err := characterStore.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-member-1",
		CampaignID:         "c1",
		OwnerParticipantID: "member-1",
		Name:               "Member Hero",
		Kind:               character.KindPC,
	}); err != nil {
		t.Fatalf("put character: %v", err)
	}

	return NewService(Deps{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
	})
}
