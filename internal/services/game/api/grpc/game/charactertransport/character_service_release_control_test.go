package charactertransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestReleaseCharacterControl_Success_WithUserIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", ParticipantID: "player-1", CreatedAt: now},
	}
	player := gametest.MemberUserParticipantRecord("c1", "player-1", "user-1", "Player One")
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
				PayloadJSON: []byte(`{"character_id":"ch1","fields":{"participant_id":""}}`),
			}),
		},
	}}

	svc := NewService(ts.withDomain(domain).build())
	resp, err := svc.ReleaseCharacterControl(requestctx.WithUserID("user-1"), &statev1.ReleaseCharacterControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	if err != nil {
		t.Fatalf("ReleaseCharacterControl returned error: %v", err)
	}
	if resp.GetCampaignId() != "c1" || resp.GetCharacterId() != "ch1" {
		t.Fatalf("response ids = %q/%q, want c1/ch1", resp.GetCampaignId(), resp.GetCharacterId())
	}
	if resp.GetParticipantId() != nil {
		t.Fatalf("response participant id = %v, want nil", resp.GetParticipantId())
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].ActorType != command.ActorTypeParticipant || domain.commands[0].ActorID != "player-1" {
		t.Fatalf("command actor = %s/%q, want participant/player-1", domain.commands[0].ActorType, domain.commands[0].ActorID)
	}
	updated, err := ts.Character.GetCharacter(context.Background(), "c1", "ch1")
	if err != nil {
		t.Fatalf("Character not persisted: %v", err)
	}
	if updated.ParticipantID != "" {
		t.Fatalf("ParticipantID = %q, want empty", updated.ParticipantID)
	}
}

func TestReleaseCharacterControl_DeniesNonController(t *testing.T) {
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
	_, err := svc.ReleaseCharacterControl(requestctx.WithUserID("user-1"), &statev1.ReleaseCharacterControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestReleaseCharacterControl_RejectsUnassignedCharacter(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", ParticipantID: "", CreatedAt: now},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"player-1": gametest.MemberUserParticipantRecord("c1", "player-1", "user-1", "Player One"),
	}

	svc := NewService(ts.build())
	_, err := svc.ReleaseCharacterControl(requestctx.WithUserID("user-1"), &statev1.ReleaseCharacterControlRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
