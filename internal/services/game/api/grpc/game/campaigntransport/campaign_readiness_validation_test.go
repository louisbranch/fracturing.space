package campaigntransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetCampaignSessionReadiness_ValidateRequest(t *testing.T) {
	svc := NewCampaignService(Deps{})

	_, err := svc.GetCampaignSessionReadiness(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)

	_, err = svc.GetCampaignSessionReadiness(context.Background(), &statev1.GetCampaignSessionReadinessRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaignSessionReadiness_NotFound(t *testing.T) {
	svc := NewCampaignService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Character:   gametest.NewFakeCharacterStore(),
		Session:     gametest.NewFakeSessionStore(),
	})

	_, err := svc.GetCampaignSessionReadiness(requestctx.WithParticipantID("owner-1"), &statev1.GetCampaignSessionReadinessRequest{CampaignId: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCampaignSessionReadiness_PermissionDeniedWhenActorMissing(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		GmMode: campaign.GmModeHuman,
	}

	svc := NewCampaignService(Deps{
		Campaign:    campaignStore,
		Participant: gametest.NewFakeParticipantStore(),
		Character:   gametest.NewFakeCharacterStore(),
		Session:     gametest.NewFakeSessionStore(),
	})

	_, err := svc.GetCampaignSessionReadiness(requestctx.WithParticipantID("missing"), &statev1.GetCampaignSessionReadinessRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCampaignSessionReadiness_ReadyCampaign(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{})

	resp, err := svc.GetCampaignSessionReadiness(requestctx.WithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	if resp.GetReadiness() == nil {
		t.Fatal("response readiness is nil")
	}
	if !resp.GetReadiness().GetReady() {
		t.Fatalf("readiness.ready = false, want true; blockers=%v", resp.GetReadiness().GetBlockers())
	}
	if len(resp.GetReadiness().GetBlockers()) != 0 {
		t.Fatalf("len(readiness.blockers) = %d, want 0", len(resp.GetReadiness().GetBlockers()))
	}
}

func TestGetCampaignSessionReadiness_BlocksWhenStatusDisallowsStart(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		status: campaign.StatusCompleted,
	})

	resp, err := svc.GetCampaignSessionReadiness(requestctx.WithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	assertReadinessHasBlockerCode(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessCampaignStatusDisallowsStart)
}

func TestGetCampaignSessionReadiness_BlocksWhenActiveSessionExists(t *testing.T) {
	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	svc, stores := newReadinessServiceFixture(readinessServiceFixtureConfig{})
	stores.session.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {
			ID:         "s1",
			CampaignID: "c1",
			Status:     session.StatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
		},
	}
	stores.session.ActiveSession["c1"] = "s1"

	resp, err := svc.GetCampaignSessionReadiness(requestctx.WithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	assertReadinessHasBlockerCode(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessActiveSessionExists)
}
