package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/metadata"
)

func TestCreateCharacter_InheritsControllerIdentityWhenAutoAssigned(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	ts.Campaign.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	ts.Participant.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			Name:           "Alice",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			AvatarSetID:    assetcatalog.AvatarSetPeopleV1,
			AvatarAssetID:  "007",
			Pronouns:       "they/them",
			CreatedAt:      now,
		},
	}

	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			commandTypeCharacterCreateWithProfile: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("character.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeParticipant,
						ActorID:     "part-1",
						EntityType:  "character",
						EntityID:    "char-123",
						PayloadJSON: []byte(`{"character_id":"char-123","name":"Hero","kind":"pc","avatar_set_id":"avatar_set_blank_v1","avatar_asset_id":"blank_faceless_silhouette"}`),
					},
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("character.profile_updated"),
						Timestamp:   now,
						ActorType:   event.ActorTypeParticipant,
						ActorID:     "part-1",
						EntityType:  "character",
						EntityID:    "char-123",
						PayloadJSON: []byte(`{"character_id":"char-123","system_profile":{"daggerheart":{"hp_max":6}}}`),
					},
				),
			},
		},
	}

	svc := &CharacterService{
		stores:      ts.withDomain(domain).build(),
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

	var payload character.CreateWithProfilePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if payload.Create.ParticipantID != "part-1" {
		t.Fatalf("participant_id = %q, want %q", payload.Create.ParticipantID, "part-1")
	}
	if payload.Create.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Create.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if payload.Create.AvatarAssetID != "007" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Create.AvatarAssetID, "007")
	}
	if payload.Create.Pronouns != "they/them" {
		t.Fatalf("pronouns = %q, want %q", payload.Create.Pronouns, "they/them")
	}
}

func TestCreateCharacter_ExplicitIdentityOverridesControllerSnapshot(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	ts.Campaign.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	ts.Participant.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			Name:           "Alice",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			AvatarSetID:    assetcatalog.AvatarSetPeopleV1,
			AvatarAssetID:  "007",
			Pronouns:       "they/them",
			CreatedAt:      now,
		},
	}

	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			commandTypeCharacterCreateWithProfile: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("character.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeParticipant,
						ActorID:     "part-1",
						EntityType:  "character",
						EntityID:    "char-456",
						PayloadJSON: []byte(`{"character_id":"char-456","name":"Hero","kind":"pc","avatar_set_id":"avatar_set_v1","avatar_asset_id":"010","participant_id":"part-1","pronouns":"she/her"}`),
					},
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("character.profile_updated"),
						Timestamp:   now,
						ActorType:   event.ActorTypeParticipant,
						ActorID:     "part-1",
						EntityType:  "character",
						EntityID:    "char-456",
						PayloadJSON: []byte(`{"character_id":"char-456","system_profile":{"daggerheart":{"hp_max":6}}}`),
					},
				),
			},
		},
	}

	svc := &CharacterService{
		stores:      ts.withDomain(domain).build(),
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-456"),
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.ParticipantIDHeader, "part-1",
	))
	resp, err := svc.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId:    "c1",
		Name:          "Hero",
		Kind:          statev1.CharacterKind_PC,
		AvatarSetId:   assetcatalog.AvatarSetPeopleV1,
		AvatarAssetId: "010",
		Pronouns:      sharedpronouns.ToProto("she/her"),
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

	var payload character.CreateWithProfilePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if payload.Create.ParticipantID != "part-1" {
		t.Fatalf("participant_id = %q, want %q", payload.Create.ParticipantID, "part-1")
	}
	if payload.Create.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Create.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if payload.Create.AvatarAssetID != "010" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Create.AvatarAssetID, "010")
	}
	if payload.Create.Pronouns != "she/her" {
		t.Fatalf("pronouns = %q, want %q", payload.Create.Pronouns, "she/her")
	}
}

func TestCreateCharacter_ExplicitEmptyPronounsDoesNotInheritControllerPronouns(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	ts.Campaign.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	ts.Participant.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			Name:           "Alice",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			AvatarSetID:    assetcatalog.AvatarSetPeopleV1,
			AvatarAssetID:  "007",
			Pronouns:       "they/them",
			CreatedAt:      now,
		},
	}

	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			commandTypeCharacterCreateWithProfile: {
				Decision: command.Accept(
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("character.created"),
						Timestamp:   now,
						ActorType:   event.ActorTypeParticipant,
						ActorID:     "part-1",
						EntityType:  "character",
						EntityID:    "char-789",
						PayloadJSON: []byte(`{"character_id":"char-789","name":"Hero","kind":"pc","avatar_set_id":"avatar_set_v1","avatar_asset_id":"007","participant_id":"part-1","pronouns":""}`),
					},
					event.Event{
						CampaignID:  "c1",
						Type:        event.Type("character.profile_updated"),
						Timestamp:   now,
						ActorType:   event.ActorTypeParticipant,
						ActorID:     "part-1",
						EntityType:  "character",
						EntityID:    "char-789",
						PayloadJSON: []byte(`{"character_id":"char-789","system_profile":{"daggerheart":{"hp_max":6}}}`),
					},
				),
			},
		},
	}

	svc := &CharacterService{
		stores:      ts.withDomain(domain).build(),
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("char-789"),
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.ParticipantIDHeader, "part-1",
	))
	resp, err := svc.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
		CampaignId: "c1",
		Name:       "Hero",
		Kind:       statev1.CharacterKind_PC,
		Pronouns: &commonv1.Pronouns{
			Value: &commonv1.Pronouns_Kind{
				Kind: commonv1.Pronoun_PRONOUN_UNSPECIFIED,
			},
		},
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

	var payload character.CreateWithProfilePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create workflow payload: %v", err)
	}
	if payload.Create.ParticipantID != "part-1" {
		t.Fatalf("participant_id = %q, want %q", payload.Create.ParticipantID, "part-1")
	}
	if payload.Create.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Create.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if payload.Create.AvatarAssetID != "007" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Create.AvatarAssetID, "007")
	}
	if payload.Create.Pronouns != "" {
		t.Fatalf("pronouns = %q, want empty", payload.Create.Pronouns)
	}
}
