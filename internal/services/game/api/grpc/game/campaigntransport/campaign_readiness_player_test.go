package campaigntransport

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
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
		ID:                 "char-1",
		CampaignID:         "c1",
		OwnerParticipantID: "player-1",
		Name:               "Aria",
	}

	resp, err := svc.GetCampaignSessionReadiness(requestctx.WithParticipantID(context.Background(), "gm-1"), &statev1.GetCampaignSessionReadinessRequest{
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
