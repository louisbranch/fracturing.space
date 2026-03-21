package campaigntransport

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestGetCampaignSessionReadiness_BlocksWhenCharacterIncompleteIncludesAction(t *testing.T) {
	svc, stores := newReadinessServiceFixture(readinessServiceFixtureConfig{})
	stores.campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Campaign One",
		Locale: "en-US",
		Status: campaign.StatusActive,
		GmMode: campaign.GmModeHuman,
		System: bridge.SystemIDDaggerheart,
	}
	stores.character.Characters["c1"]["char-1"] = storage.CharacterRecord{
		ID:            "char-1",
		CampaignID:    "c1",
		ParticipantID: "player-1",
		Name:          "Aria",
	}

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}

	blocker := findReadinessBlocker(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessCharacterSystemRequired)
	action := blocker.GetAction()
	if action == nil {
		t.Fatal("blocker action is nil")
	}
	if got := action.GetResolutionKind(); got != statev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_COMPLETE_CHARACTER {
		t.Fatalf("resolution kind = %v, want %v", got, statev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_COMPLETE_CHARACTER)
	}
	assertStringSliceEqual(t, action.GetResponsibleUserIds(), []string{"user-player-1"})
	assertStringSliceEqual(t, action.GetResponsibleParticipantIds(), []string{"player-1"})
	if got := action.GetTargetParticipantId(); got != "player-1" {
		t.Fatalf("target participant id = %q, want %q", got, "player-1")
	}
	if got := action.GetTargetCharacterId(); got != "char-1" {
		t.Fatalf("target character id = %q, want %q", got, "char-1")
	}
}

func TestGetCampaignSessionReadiness_BlocksWhenPlayerNeedsCharacterIncludesAction(t *testing.T) {
	svc, stores := newReadinessServiceFixture(readinessServiceFixtureConfig{})
	delete(stores.character.Characters["c1"], "char-1")

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}

	blocker := findReadinessBlocker(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessPlayerCharacterRequired)
	action := blocker.GetAction()
	if action == nil {
		t.Fatal("blocker action is nil")
	}
	if got := action.GetResolutionKind(); got != statev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_CREATE_CHARACTER {
		t.Fatalf("resolution kind = %v, want %v", got, statev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_CREATE_CHARACTER)
	}
	assertStringSliceEqual(t, action.GetResponsibleUserIds(), []string{"user-player-1"})
	assertStringSliceEqual(t, action.GetResponsibleParticipantIds(), []string{"player-1"})
	if got := action.GetTargetParticipantId(); got != "player-1" {
		t.Fatalf("target participant id = %q, want %q", got, "player-1")
	}
	if got := action.GetTargetCharacterId(); got != "" {
		t.Fatalf("target character id = %q, want empty", got)
	}
}

func TestGetCampaignSessionReadiness_CharacterControllerUsesCharacterName(t *testing.T) {
	svc, stores := newReadinessServiceFixture(readinessServiceFixtureConfig{})
	stores.character.Characters["c1"]["char-1"] = storage.CharacterRecord{
		ID:         "char-1",
		CampaignID: "c1",
		Name:       "Aria",
	}

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}

	blocker := findReadinessBlocker(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessCharacterControllerRequired)
	if got := blocker.GetMetadata()["character_name"]; got != "Aria" {
		t.Fatalf("blocker metadata character_name = %q, want %q", got, "Aria")
	}
	if got := blocker.GetMetadata()["character_id"]; got != "char-1" {
		t.Fatalf("blocker metadata character_id = %q, want %q", got, "char-1")
	}
	if !strings.Contains(blocker.GetMessage(), "Aria") {
		t.Fatalf("blocker message = %q, want character name", blocker.GetMessage())
	}
	if strings.Contains(blocker.GetMessage(), "char-1") {
		t.Fatalf("blocker message = %q, did not expect character id when name is present", blocker.GetMessage())
	}
}
