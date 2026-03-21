package charactertransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetCharacterSheet_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.GetCharacterSheet(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_MissingCampaignId(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{CharacterId: "ch1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_MissingCharacterId(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(ts.build())
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCharacterSheet_CampaignNotFound(t *testing.T) {
	svc := NewService(newTestStores().withCharacter().build())
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCharacterSheet_DeniesMissingIdentity(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())
	_, err := svc.GetCharacterSheet(context.Background(), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCharacterSheet_CharacterNotFound(t *testing.T) {
	ts := newTestStores().withCharacter()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())
	_, err := svc.GetCharacterSheet(gametest.ContextWithParticipantID("p1"), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCharacterSheet_Success(t *testing.T) {
	ts := newTestStores().withCharacter()
	now := time.Now().UTC()

	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Character.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, CreatedAt: now},
	}
	ts.Daggerheart.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 12, StressMax: 6, Evasion: 10, MajorThreshold: 5, SevereThreshold: 10, Agility: 2, Strength: 1},
	}
	ts.Daggerheart.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": gametest.NamedRoleMemberParticipantRecord("c1", "p1", "GM", participant.RoleGM),
	}

	svc := NewService(ts.build())

	resp, err := svc.GetCharacterSheet(gametest.ContextWithParticipantID("p1"), &statev1.GetCharacterSheetRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	if err != nil {
		t.Fatalf("GetCharacterSheet returned error: %v", err)
	}
	if resp.Character == nil {
		t.Fatal("GetCharacterSheet response has nil character")
	}
	if resp.Profile == nil {
		t.Fatal("GetCharacterSheet response has nil profile")
	}
	if resp.State == nil {
		t.Fatal("GetCharacterSheet response has nil state")
	}
	if resp.Character.Name != "Hero" {
		t.Errorf("Character Name = %q, want %q", resp.Character.Name, "Hero")
	}
	if dh := resp.Profile.GetDaggerheart(); dh == nil || dh.GetHpMax() != 12 {
		t.Errorf("Profile HpMax = %d, want %d", dh.GetHpMax(), 12)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHope() != 3 {
		t.Errorf("State Hope = %d, want %d", dh.GetHope(), 3)
	}
}
