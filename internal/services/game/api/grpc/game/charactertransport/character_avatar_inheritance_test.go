package charactertransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCreateCharacter_InheritsControllerAvatarWhenAutoAssigned(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
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
		resultsByType: testCreateCharacterResults(
			t,
			now,
			"c1",
			"char-123",
			event.ActorTypeParticipant,
			"part-1",
			character.CreatePayload{
				CharacterID:   "char-123",
				Name:          "Hero",
				Kind:          "pc",
				AvatarSetID:   assetcatalog.AvatarSetBlankV1,
				AvatarAssetID: "blank_faceless_silhouette",
			},
		),
	}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), gametest.FixedClock(now), gametest.FixedIDGenerator("char-123"))

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
	if payload.ParticipantID != "part-1" {
		t.Fatalf("participant_id = %q, want %q", payload.ParticipantID, "part-1")
	}
	if payload.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if payload.AvatarAssetID != "007" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.AvatarAssetID, "007")
	}
	if payload.Pronouns != "" {
		t.Fatalf("pronouns = %q, want empty", payload.Pronouns)
	}
}

func TestCreateCharacter_ExplicitIdentityOverridesControllerSnapshot(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
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
		resultsByType: testCreateCharacterResults(
			t,
			now,
			"c1",
			"char-456",
			event.ActorTypeParticipant,
			"part-1",
			character.CreatePayload{
				CharacterID:   "char-456",
				Name:          "Hero",
				Kind:          "pc",
				AvatarSetID:   assetcatalog.AvatarSetPeopleV1,
				AvatarAssetID: "010",
				ParticipantID: "part-1",
				Pronouns:      "she/her",
			},
		),
	}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), gametest.FixedClock(now), gametest.FixedIDGenerator("char-456"))

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

	var payload character.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}
	if payload.ParticipantID != "part-1" {
		t.Fatalf("participant_id = %q, want %q", payload.ParticipantID, "part-1")
	}
	if payload.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if payload.AvatarAssetID != "010" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.AvatarAssetID, "010")
	}
	if payload.Pronouns != "she/her" {
		t.Fatalf("pronouns = %q, want %q", payload.Pronouns, "she/her")
	}
}

func TestCreateCharacter_ExplicitEmptyPronounsDoesNotInheritControllerPronouns(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	ts.Campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
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
		resultsByType: testCreateCharacterResults(
			t,
			now,
			"c1",
			"char-789",
			event.ActorTypeParticipant,
			"part-1",
			character.CreatePayload{
				CharacterID:   "char-789",
				Name:          "Hero",
				Kind:          "pc",
				AvatarSetID:   assetcatalog.AvatarSetPeopleV1,
				AvatarAssetID: "007",
				ParticipantID: "part-1",
				Pronouns:      "",
			},
		),
	}

	svc := newCharacterServiceForTest(ts.withDomain(domain).build(), gametest.FixedClock(now), gametest.FixedIDGenerator("char-789"))

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

	var payload character.CreatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}
	if payload.ParticipantID != "part-1" {
		t.Fatalf("participant_id = %q, want %q", payload.ParticipantID, "part-1")
	}
	if payload.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", payload.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if payload.AvatarAssetID != "007" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.AvatarAssetID, "007")
	}
	if payload.Pronouns != "" {
		t.Fatalf("pronouns = %q, want empty", payload.Pronouns)
	}
}

func TestResolveCharacterIdentitySnapshot_RequiresParticipantStoreWhenParticipantProvided(t *testing.T) {
	app := characterApplication{}

	_, err := app.resolveCharacterIdentitySnapshot(context.Background(), "c1", "part-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("resolveCharacterIdentitySnapshot error code = %v, want %v", status.Code(err), codes.Internal)
	}
}
