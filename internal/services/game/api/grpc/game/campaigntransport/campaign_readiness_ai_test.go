package campaigntransport

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
)

func TestGetCampaignSessionReadiness_BlocksWhenAIAgentMissing(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:            campaign.GmModeAI,
		aiAgentID:         "",
		includeAIGM:       true,
		includeHumanGM:    false,
		includePlayerSeat: true,
	})

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("ai-gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	assertReadinessHasBlockerCode(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessAIAgentRequired)
}

func TestGetCampaignSessionReadiness_BlocksWhenAIAgentMissingIncludesAction(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:            campaign.GmModeAI,
		aiAgentID:         "",
		includeAIGM:       true,
		includeHumanGM:    true,
		includePlayerSeat: true,
	})

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}

	blocker := findReadinessBlocker(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessAIAgentRequired)
	action := blocker.GetAction()
	if action == nil {
		t.Fatal("blocker action is nil")
	}
	if got := action.GetResolutionKind(); got != statev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_CONFIGURE_AI_AGENT {
		t.Fatalf("resolution kind = %v, want %v", got, statev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_CONFIGURE_AI_AGENT)
	}
	assertStringSliceEqual(t, action.GetResponsibleUserIds(), []string{"user-gm-1"})
	assertStringSliceEqual(t, action.GetResponsibleParticipantIds(), []string{"gm-1"})
	if got := action.GetTargetParticipantId(); got != "ai-gm-1" {
		t.Fatalf("target participant id = %q, want %q", got, "ai-gm-1")
	}
	if got := action.GetTargetCharacterId(); got != "" {
		t.Fatalf("target character id = %q, want empty", got)
	}
}

func TestGetCampaignSessionReadiness_BlocksWhenAIGMParticipantMissing(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:      campaign.GmModeAI,
		aiAgentID:   "agent-1",
		includeAIGM: false,
	})

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	assertReadinessHasBlockerCode(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessAIGMParticipantRequired)
}

func TestGetCampaignSessionReadiness_UsesRequestedLocale(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:      campaign.GmModeAI,
		aiAgentID:   "agent-1",
		includeAIGM: false,
	})

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
		Locale:     commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	blocker := findReadinessBlocker(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessAIGMParticipantRequired)
	if !strings.Contains(strings.ToLower(blocker.GetMessage()), "prontid") {
		t.Fatalf("blocker message = %q, want portuguese localized message", blocker.GetMessage())
	}
}

func TestGetCampaignSessionReadiness_FallsBackToCampaignLocale(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:      campaign.GmModeAI,
		aiAgentID:   "agent-1",
		includeAIGM: false,
		locale:      "pt-BR",
	})

	resp, err := svc.GetCampaignSessionReadiness(gametest.ContextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	blocker := findReadinessBlocker(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessAIGMParticipantRequired)
	if !strings.Contains(strings.ToLower(blocker.GetMessage()), "prontid") {
		t.Fatalf("blocker message = %q, want portuguese localized message via campaign locale fallback", blocker.GetMessage())
	}
}
