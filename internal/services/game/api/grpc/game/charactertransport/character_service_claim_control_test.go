package charactertransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestClaimCharacterControl_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.ClaimCharacterControl(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClaimCharacterControl_Success_WithUserIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Pronouns: "she/her", CreatedAt: now},
	}
	player := gametest.MemberUserParticipantRecord("c1", "player-1", "user-1", "Player One")
	player.AvatarSetID = assetcatalog.AvatarSetPeopleV1
	player.AvatarAssetID = "009"
	player.Pronouns = "they/them"
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"player-1": player,
	}
	domain := &fakeDomainEngine{store: ts.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("character.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "player-1",
				EntityType:  "character",
				EntityID:    "ch1",
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":"player-1"}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.ClaimCharacterControl(gametest.ContextWithUserID("user-1"), &statev1.ClaimCharacterControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	if err != nil {
		t.Fatalf("ClaimCharacterControl returned error: %v", err)
	}
	if resp.GetCampaignId() != "c1" || resp.GetCharacterId() != "ch1" {
		t.Fatalf("response ids = %q/%q, want c1/ch1", resp.GetCampaignId(), resp.GetCharacterId())
	}
	if got := resp.GetParticipantId().GetValue(); got != "player-1" {
		t.Fatalf("response participant id = %q, want %q", got, "player-1")
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].ActorType != command.ActorTypeParticipant || domain.commands[0].ActorID != "player-1" {
		t.Fatalf("command actor = %s/%q, want participant/player-1", domain.commands[0].ActorType, domain.commands[0].ActorID)
	}
	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.Fields["participant_id"] != "player-1" {
		t.Fatalf("participant_id = %q, want %q", payload.Fields["participant_id"], "player-1")
	}
	if payload.Fields["avatar_set_id"] != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], assetcatalog.AvatarSetPeopleV1)
	}
	if payload.Fields["avatar_asset_id"] != "009" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "009")
	}
	if _, ok := payload.Fields["pronouns"]; ok {
		t.Fatalf("pronouns field should be omitted, got %q", payload.Fields["pronouns"])
	}
	updated, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if updated.ParticipantID != "player-1" {
		t.Fatalf("ParticipantID = %q, want %q", updated.ParticipantID, "player-1")
	}
	if updated.Pronouns != "she/her" {
		t.Fatalf("Pronouns = %q, want %q", updated.Pronouns, "she/her")
	}
}

func TestClaimCharacterControl_RejectsAssignedCharacter(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", ParticipantID: "player-2", CreatedAt: now},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"player-1": gametest.MemberUserParticipantRecord("c1", "player-1", "user-1", "Player One"),
		"player-2": gametest.MemberUserParticipantRecord("c1", "player-2", "user-2", "Player Two"),
	}

	svc := NewService(ts.build())
	_, err := svc.ClaimCharacterControl(gametest.ContextWithUserID("user-1"), &statev1.ClaimCharacterControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestClaimCharacterControl_DeniesEmptyResolvedParticipantID(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", CreatedAt: now},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"seat-1": {
			ID:         "",
			CampaignID: "c1",
			UserID:     "user-1",
			Name:       "Player One",
		},
	}

	svc := NewService(ts.build())
	_, err := svc.ClaimCharacterControl(gametest.ContextWithUserID("user-1"), &statev1.ClaimCharacterControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}
