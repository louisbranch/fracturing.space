package game

import (
	"context"
	"errors"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
)

type fakeCampaignStoreWithAIBindingReader struct {
	*fakeCampaignStore
	campaignIDsByAgent map[string][]string
	listByAgentErr     error
}

func (s *fakeCampaignStoreWithAIBindingReader) ListCampaignIDsByAIAgent(_ context.Context, aiAgentID string) ([]string, error) {
	if s.listByAgentErr != nil {
		return nil, s.listByAgentErr
	}
	ids := s.campaignIDsByAgent[aiAgentID]
	copied := make([]string, len(ids))
	copy(copied, ids)
	return copied, nil
}

func campaignAIGrantConfig(now time.Time) aisessiongrant.Config {
	return aisessiongrant.Config{
		Issuer:   "test-issuer",
		Audience: "test-audience",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      10 * time.Minute,
		Now:      fixedClock(now),
	}
}

func newCampaignAIServiceForTest(stores Stores, now time.Time) *CampaignAIService {
	return newCampaignAIServiceWithDependencies(stores, fixedClock(now), fixedIDGenerator("grant-1"), campaignAIGrantConfig(now))
}

func TestIssueCampaignAISessionGrantRequiresRequestAndStores(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	svc := newCampaignAIServiceForTest(Stores{}, now)

	_, err := svc.IssueCampaignAISessionGrant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)

	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestIssueCampaignAISessionGrantValidationFailures(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	svc := newCampaignAIServiceForTest(Stores{Campaign: campaignStore, Session: sessionStore}, now)

	_, err := svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: ""})
	assertStatusCode(t, err, codes.InvalidArgument)

	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeHuman, AIAgentID: "agent-1"}
	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)

	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI}
	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)

	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-2"}
	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestIssueCampaignAISessionGrantRequiresMatchingActiveSession(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-1"}

	svc := newCampaignAIServiceForTest(Stores{Campaign: campaignStore, Session: sessionStore}, now)
	_, err := svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)

	sessionStore.sessions["camp-1"] = map[string]storage.SessionRecord{
		"session-2": {ID: "session-2", CampaignID: "camp-1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["camp-1"] = "session-2"
	_, err = svc.IssueCampaignAISessionGrant(context.Background(), &statev1.IssueCampaignAISessionGrantRequest{CampaignId: "camp-1", SessionId: "session-1", AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestIssueCampaignAISessionGrantSuccess(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", GmMode: campaign.GmModeAI, AIAgentID: "agent-1", AIAuthEpoch: 13}
	sessionStore.sessions["camp-1"] = map[string]storage.SessionRecord{
		"session-1": {ID: "session-1", CampaignID: "camp-1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["camp-1"] = "session-1"

	svc := newCampaignAIServiceForTest(Stores{Campaign: campaignStore, Session: sessionStore}, now)
	resp, err := svc.IssueCampaignAISessionGrant(contextWithUserID("user-7"), &statev1.IssueCampaignAISessionGrantRequest{
		CampaignId: "camp-1",
		SessionId:  "session-1",
		AiAgentId:  "agent-1",
	})
	if err != nil {
		t.Fatalf("issue campaign ai session grant: %v", err)
	}
	if resp.GetGrant() == nil {
		t.Fatal("expected grant in response")
	}
	if resp.GetGrant().GetGrantId() != "grant-1" {
		t.Fatalf("grant id = %q, want %q", resp.GetGrant().GetGrantId(), "grant-1")
	}
	if resp.GetGrant().GetIssuedForUserId() != "user-7" {
		t.Fatalf("issued for user id = %q, want %q", resp.GetGrant().GetIssuedForUserId(), "user-7")
	}
	if resp.GetGrant().GetAuthEpoch() != 13 {
		t.Fatalf("auth epoch = %d, want %d", resp.GetGrant().GetAuthEpoch(), 13)
	}
	claims, err := aisessiongrant.Validate(campaignAIGrantConfig(now), resp.GetGrant().GetToken())
	if err != nil {
		t.Fatalf("validate issued token: %v", err)
	}
	if claims.CampaignID != "camp-1" || claims.SessionID != "session-1" || claims.AIAgentID != "agent-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestGetCampaignAIBindingUsage(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	svc := newCampaignAIServiceForTest(Stores{}, now)
	_, err := svc.GetCampaignAIBindingUsage(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)

	_, err = svc.GetCampaignAIBindingUsage(context.Background(), &statev1.GetCampaignAIBindingUsageRequest{AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.Internal)

	svc = newCampaignAIServiceForTest(Stores{Campaign: newFakeCampaignStore()}, now)
	_, err = svc.GetCampaignAIBindingUsage(context.Background(), &statev1.GetCampaignAIBindingUsageRequest{AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.Internal)

	campaignStore := &fakeCampaignStoreWithAIBindingReader{
		fakeCampaignStore:  newFakeCampaignStore(),
		campaignIDsByAgent: map[string][]string{"agent-1": {"camp-1", "camp-2"}},
	}
	svc = newCampaignAIServiceForTest(Stores{Campaign: campaignStore}, now)
	resp, err := svc.GetCampaignAIBindingUsage(context.Background(), &statev1.GetCampaignAIBindingUsageRequest{AiAgentId: "agent-1"})
	if err != nil {
		t.Fatalf("get campaign ai binding usage: %v", err)
	}
	if resp.GetActiveCampaignCount() != 2 {
		t.Fatalf("active campaign count = %d, want %d", resp.GetActiveCampaignCount(), 2)
	}
	if len(resp.GetCampaignIds()) != 2 || resp.GetCampaignIds()[0] != "camp-1" || resp.GetCampaignIds()[1] != "camp-2" {
		t.Fatalf("campaign ids = %v, want [camp-1 camp-2]", resp.GetCampaignIds())
	}

	campaignStore.listByAgentErr = errors.New("boom")
	_, err = svc.GetCampaignAIBindingUsage(context.Background(), &statev1.GetCampaignAIBindingUsageRequest{AiAgentId: "agent-1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetCampaignAIAuthState(t *testing.T) {
	now := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)
	svc := newCampaignAIServiceForTest(Stores{}, now)
	_, err := svc.GetCampaignAIAuthState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
	_, err = svc.GetCampaignAIAuthState(context.Background(), &statev1.GetCampaignAIAuthStateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)

	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", AIAgentID: "agent-1", AIAuthEpoch: 8}
	sessionStore := newFakeSessionStore()
	svc = newCampaignAIServiceForTest(Stores{Campaign: campaignStore, Session: sessionStore}, now)

	resp, err := svc.GetCampaignAIAuthState(context.Background(), &statev1.GetCampaignAIAuthStateRequest{CampaignId: "camp-1"})
	if err != nil {
		t.Fatalf("get campaign ai auth state: %v", err)
	}
	if resp.GetCampaignId() != "camp-1" || resp.GetAiAgentId() != "agent-1" || resp.GetAuthEpoch() != 8 {
		t.Fatalf("unexpected auth state response: %+v", resp)
	}
	if resp.GetActiveSessionId() != "" {
		t.Fatalf("active session id = %q, want empty", resp.GetActiveSessionId())
	}

	sessionStore.sessions["camp-1"] = map[string]storage.SessionRecord{
		"session-1": {ID: "session-1", CampaignID: "camp-1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["camp-1"] = "session-1"
	resp, err = svc.GetCampaignAIAuthState(context.Background(), &statev1.GetCampaignAIAuthStateRequest{CampaignId: "camp-1"})
	if err != nil {
		t.Fatalf("get campaign ai auth state with active session: %v", err)
	}
	if resp.GetActiveSessionId() != "session-1" {
		t.Fatalf("active session id = %q, want %q", resp.GetActiveSessionId(), "session-1")
	}

	sessionStore.activeErr = errors.New("boom")
	_, err = svc.GetCampaignAIAuthState(context.Background(), &statev1.GetCampaignAIAuthStateRequest{CampaignId: "camp-1"})
	assertStatusCode(t, err, codes.Internal)
}
