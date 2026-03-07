package game

import (
	"context"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetCampaignSessionReadiness_ValidateRequest(t *testing.T) {
	svc := NewCampaignService(Stores{})

	_, err := svc.GetCampaignSessionReadiness(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)

	_, err = svc.GetCampaignSessionReadiness(context.Background(), &statev1.GetCampaignSessionReadinessRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetCampaignSessionReadiness_NotFound(t *testing.T) {
	svc := NewCampaignService(Stores{
		Campaign:    newFakeCampaignStore(),
		Participant: newFakeParticipantStore(),
		Character:   newFakeCharacterStore(),
		Session:     newFakeSessionStore(),
	})

	_, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("owner-1"), &statev1.GetCampaignSessionReadinessRequest{CampaignId: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetCampaignSessionReadiness_PermissionDeniedWhenActorMissing(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		GmMode: campaign.GmModeHuman,
	}

	svc := NewCampaignService(Stores{
		Campaign:    campaignStore,
		Participant: newFakeParticipantStore(),
		Character:   newFakeCharacterStore(),
		Session:     newFakeSessionStore(),
	})

	_, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("missing"), &statev1.GetCampaignSessionReadinessRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetCampaignSessionReadiness_ReadyCampaign(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{})

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
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

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
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
	stores.session.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {
			ID:         "s1",
			CampaignID: "c1",
			Status:     session.StatusActive,
			StartedAt:  now,
			UpdatedAt:  now,
		},
	}
	stores.session.activeSession["c1"] = "s1"

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	assertReadinessHasBlockerCode(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessActiveSessionExists)
}

func TestGetCampaignSessionReadiness_BlocksWhenAIAgentMissing(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:            campaign.GmModeAI,
		aiAgentID:         "",
		includeAIGM:       true,
		includeHumanGM:    false,
		includePlayerSeat: true,
	})

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("ai-gm-1"), &statev1.GetCampaignSessionReadinessRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("GetCampaignSessionReadiness() error = %v", err)
	}
	assertReadinessHasBlockerCode(t, resp.GetReadiness(), readiness.RejectionCodeSessionReadinessAIAgentRequired)
}

func TestGetCampaignSessionReadiness_BlocksWhenAIGMParticipantMissing(t *testing.T) {
	svc, _ := newReadinessServiceFixture(readinessServiceFixtureConfig{
		gmMode:      campaign.GmModeAI,
		aiAgentID:   "agent-1",
		includeAIGM: false,
	})

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
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

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
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
		locale:      commonv1.Locale_LOCALE_PT_BR,
	})

	resp, err := svc.GetCampaignSessionReadiness(contextWithParticipantID("gm-1"), &statev1.GetCampaignSessionReadinessRequest{
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

func TestLocalizeReadinessBlockerMessage_CharacterSystemWithoutReason(t *testing.T) {
	message := localizeReadinessBlockerMessage(commonv1.Locale_LOCALE_EN_US, readiness.Blocker{
		Code: readiness.RejectionCodeSessionReadinessCharacterSystemRequired,
		Metadata: map[string]string{
			"character_id": "char-1",
		},
	})
	if strings.Contains(strings.ToLower(message), "unspecified") {
		t.Fatalf("message = %q, did not expect unspecified fallback reason", message)
	}
	if !strings.Contains(message, "char-1") {
		t.Fatalf("message = %q, want character id in message", message)
	}
}

func TestLocalizeReadinessBlockerMessage_PlayerCharacterUsesParticipantName(t *testing.T) {
	message := localizeReadinessBlockerMessage(commonv1.Locale_LOCALE_EN_US, readiness.Blocker{
		Code: readiness.RejectionCodeSessionReadinessPlayerCharacterRequired,
		Metadata: map[string]string{
			"participant_id":   "player-2",
			"participant_name": "Player Two",
		},
	})
	if !strings.Contains(message, "Player Two") {
		t.Fatalf("message = %q, want participant name in localized message", message)
	}
	if strings.Contains(message, "player-2") {
		t.Fatalf("message = %q, did not expect participant id when name is present", message)
	}
}

func TestLocalizeReadinessBlockerMessage_PlayerCharacterFallsBackToParticipantID(t *testing.T) {
	message := localizeReadinessBlockerMessage(commonv1.Locale_LOCALE_EN_US, readiness.Blocker{
		Code: readiness.RejectionCodeSessionReadinessPlayerCharacterRequired,
		Metadata: map[string]string{
			"participant_id": "player-2",
		},
	})
	if !strings.Contains(message, "player-2") {
		t.Fatalf("message = %q, want participant id fallback in localized message", message)
	}
}

type readinessServiceFixtureStores struct {
	campaign    *fakeCampaignStore
	participant *fakeParticipantStore
	character   *fakeCharacterStore
	session     *fakeSessionStore
}

type readinessServiceFixtureConfig struct {
	status            campaign.Status
	gmMode            campaign.GmMode
	aiAgentID         string
	locale            commonv1.Locale
	includeHumanGM    bool
	includeAIGM       bool
	includePlayerSeat bool
}

func newReadinessServiceFixture(config readinessServiceFixtureConfig) (*CampaignService, readinessServiceFixtureStores) {
	stores := readinessServiceFixtureStores{
		campaign:    newFakeCampaignStore(),
		participant: newFakeParticipantStore(),
		character:   newFakeCharacterStore(),
		session:     newFakeSessionStore(),
	}

	status := config.status
	if status == "" {
		status = campaign.StatusActive
	}
	gmMode := config.gmMode
	if gmMode == "" {
		gmMode = campaign.GmModeHuman
	}
	locale := config.locale
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}
	stores.campaign.campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Campaign One",
		Locale:    locale,
		Status:    status,
		GmMode:    gmMode,
		AIAgentID: strings.TrimSpace(config.aiAgentID),
	}

	includeHumanGM := config.includeHumanGM
	includePlayerSeat := config.includePlayerSeat
	if !includeHumanGM && !config.includeAIGM {
		includeHumanGM = true
	}
	if !includePlayerSeat {
		includePlayerSeat = true
	}

	participants := map[string]storage.ParticipantRecord{}
	if includeHumanGM {
		participants["gm-1"] = storage.ParticipantRecord{
			ID:             "gm-1",
			CampaignID:     "c1",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessOwner,
		}
	}
	if config.includeAIGM {
		participants["ai-gm-1"] = storage.ParticipantRecord{
			ID:             "ai-gm-1",
			CampaignID:     "c1",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessOwner,
		}
	}
	if includePlayerSeat {
		participants["player-1"] = storage.ParticipantRecord{
			ID:             "player-1",
			CampaignID:     "c1",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
		}
	}
	stores.participant.participants["c1"] = participants

	stores.character.characters["c1"] = map[string]storage.CharacterRecord{
		"char-1": {
			ID:            "char-1",
			CampaignID:    "c1",
			ParticipantID: "player-1",
		},
	}

	service := NewCampaignService(Stores{
		Campaign:    stores.campaign,
		Participant: stores.participant,
		Character:   stores.character,
		Session:     stores.session,
	})
	return service, stores
}

func assertReadinessHasBlockerCode(t *testing.T, report *statev1.CampaignSessionReadiness, code string) {
	t.Helper()
	if report == nil {
		t.Fatal("readiness report is nil")
	}
	if report.GetReady() {
		t.Fatalf("readiness.ready = true, want false with blocker %s", code)
	}
	for _, blocker := range report.GetBlockers() {
		if strings.TrimSpace(blocker.GetCode()) == code {
			return
		}
	}
	t.Fatalf("expected blocker code %q, got %v", code, readinessBlockerCodes(report.GetBlockers()))
}

func findReadinessBlocker(t *testing.T, report *statev1.CampaignSessionReadiness, code string) *statev1.CampaignSessionReadinessBlocker {
	t.Helper()
	if report == nil {
		t.Fatal("readiness report is nil")
	}
	for _, blocker := range report.GetBlockers() {
		if strings.TrimSpace(blocker.GetCode()) == code {
			return blocker
		}
	}
	t.Fatalf("expected blocker code %q, got %v", code, readinessBlockerCodes(report.GetBlockers()))
	return nil
}

func readinessBlockerCodes(blockers []*statev1.CampaignSessionReadinessBlocker) []string {
	codes := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		codes = append(codes, strings.TrimSpace(blocker.GetCode()))
	}
	return codes
}
