package charactertransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListCharacters_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.ListCharacters(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCharacters_MissingCampaignId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCharacters_CampaignNotFound(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListCharacters_DeniesMissingIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())
	_, err := svc.ListCharacters(context.Background(), &statev1.ListCharactersRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListCharacters_EmptyList(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())
	resp, err := svc.ListCharacters(gametest.ContextWithParticipantID("p1"), &statev1.ListCharactersRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListCharacters returned error: %v", err)
	}
	if len(resp.Characters) != 0 {
		t.Errorf("ListCharacters returned %d characters, want 0", len(resp.Characters))
	}
}

func TestListCharacters_WithCharacters(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, CreatedAt: now},
		"ch2": {ID: "ch2", CampaignID: "c1", Name: "Sidekick", Kind: character.KindNPC, CreatedAt: now},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())
	resp, err := svc.ListCharacters(gametest.ContextWithParticipantID("p1"), &statev1.ListCharactersRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListCharacters returned error: %v", err)
	}
	if len(resp.Characters) != 2 {
		t.Errorf("ListCharacters returned %d characters, want 2", len(resp.Characters))
	}
}

func TestListCharacterProfiles_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.ListCharacterProfiles(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCharacterProfiles_MissingCampaignID(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.ListCharacterProfiles(context.Background(), &statev1.ListCharacterProfilesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListCharacterProfiles_DeniesMissingIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())
	_, err := svc.ListCharacterProfiles(context.Background(), &statev1.ListCharacterProfilesRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListCharacterProfiles_EmptyForNonDaggerheartCampaigns(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Level: 2, ClassID: "warrior"},
	}

	svc := NewService(ts.build())
	resp, err := svc.ListCharacterProfiles(gametest.ContextWithParticipantID("p1"), &statev1.ListCharacterProfilesRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListCharacterProfiles returned error: %v", err)
	}
	if len(resp.GetProfiles()) != 0 {
		t.Fatalf("ListCharacterProfiles returned %d profiles, want 0", len(resp.GetProfiles()))
	}
}

func TestListCharacterProfiles_WithProfiles(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.DaggerheartCampaignRecord("c1", "Campaign", campaign.StatusActive, campaign.GmModeHuman)
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch-b": {
			CampaignID:  "c1",
			CharacterID: "ch-b",
			Level:       3,
			ClassID:     "warrior",
			SubclassID:  "guardian",
			Heritage: projectionstore.DaggerheartHeritageSelection{
				FirstFeatureAncestryID:  "clank",
				FirstFeatureID:          "clank.feature-1",
				SecondFeatureAncestryID: "clank",
				SecondFeatureID:         "clank.feature-2",
				CommunityID:             "ridgeborne",
			},
		},
		"ch-a": {
			CampaignID:  "c1",
			CharacterID: "ch-a",
			Level:       2,
			ClassID:     "seraph",
			SubclassID:  "wingguard",
			Heritage: projectionstore.DaggerheartHeritageSelection{
				FirstFeatureAncestryID:  "drakona",
				FirstFeatureID:          "drakona.feature-1",
				SecondFeatureAncestryID: "drakona",
				SecondFeatureID:         "drakona.feature-2",
				CommunityID:             "wanderborne",
			},
		},
	}

	svc := NewService(ts.build())
	resp, err := svc.ListCharacterProfiles(gametest.ContextWithParticipantID("p1"), &statev1.ListCharacterProfilesRequest{CampaignId: "c1", PageSize: 1})
	if err != nil {
		t.Fatalf("ListCharacterProfiles returned error: %v", err)
	}
	if len(resp.GetProfiles()) != 1 {
		t.Fatalf("ListCharacterProfiles returned %d profiles, want 1", len(resp.GetProfiles()))
	}
	if got := resp.GetProfiles()[0].GetCharacterId(); got != "ch-a" {
		t.Fatalf("first profile character id = %q, want %q", got, "ch-a")
	}
	if got := resp.GetProfiles()[0].GetDaggerheart().GetLevel(); got != 2 {
		t.Fatalf("first profile level = %d, want 2", got)
	}
	if got := resp.GetNextPageToken(); got != "ch-a" {
		t.Fatalf("next page token = %q, want %q", got, "ch-a")
	}

	resp, err = svc.ListCharacterProfiles(gametest.ContextWithParticipantID("p1"), &statev1.ListCharacterProfilesRequest{
		CampaignId: "c1",
		PageSize:   1,
		PageToken:  resp.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("ListCharacterProfiles second page returned error: %v", err)
	}
	if len(resp.GetProfiles()) != 1 {
		t.Fatalf("second page returned %d profiles, want 1", len(resp.GetProfiles()))
	}
	if got := resp.GetProfiles()[0].GetCharacterId(); got != "ch-b" {
		t.Fatalf("second page character id = %q, want %q", got, "ch-b")
	}
	if got := resp.GetNextPageToken(); got != "" {
		t.Fatalf("second page next page token = %q, want empty", got)
	}
}
