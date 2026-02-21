package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/metadata"
)

func TestCreateCharacter_InheritsActorParticipantAvatarWhenAvatarNotProvided(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			Name:           "Alice",
			CampaignAccess: participant.CampaignAccessMember,
			AvatarSetID:    "avatar_set_v1",
			AvatarAssetID:  "007",
			CreatedAt:      now,
		},
	}

	domain := &fakeDomainEngine{
		store: eventStore,
		resultsByType: map[command.Type]engine.Result{
			command.Type("character.create"): {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        event.Type("character.created"),
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					ActorID:     "part-1",
					EntityType:  "character",
					EntityID:    "char-123",
					PayloadJSON: []byte(`{"character_id":"char-123","name":"Hero","kind":"pc","avatar_set_id":"avatar_set_v1","avatar_asset_id":"007"}`),
				}),
			},
			command.Type("character.profile_update"): {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        event.Type("character.profile_updated"),
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					ActorID:     "part-1",
					EntityType:  "character",
					EntityID:    "char-123",
					PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
				}),
			},
			command.Type("sys.daggerheart.character_state.patch"): {
				Decision: command.Accept(event.Event{
					CampaignID:    "c1",
					Type:          event.Type("sys.daggerheart.character_state_patched"),
					Timestamp:     now,
					ActorType:     event.ActorTypeParticipant,
					ActorID:       "part-1",
					EntityType:    "character",
					EntityID:      "char-123",
					SystemID:      "GAME_SYSTEM_DAGGERHEART",
					SystemVersion: "1.0.0",
					PayloadJSON:   []byte(`{"character_id":"char-123","hp_after":6}`),
				}),
			},
		},
	}

	svc := &CharacterService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
			Character:   characterStore,
			Daggerheart: dhStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-123"),
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.ParticipantIDHeader, "part-1",
	))
	resp, err := svc.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("CreateCharacter returned error: %v", err)
	}
	if resp.GetCharacter() == nil {
		t.Fatal("CreateCharacter response has nil character")
	}
	if len(domain.commands) == 0 {
		t.Fatal("expected at least one domain command")
	}

	var payload character.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}
	if payload.AvatarSetID != "avatar_set_v1" {
		t.Fatalf("avatar_set_id = %q, want %q", payload.AvatarSetID, "avatar_set_v1")
	}
	if payload.AvatarAssetID != "007" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.AvatarAssetID, "007")
	}
}
