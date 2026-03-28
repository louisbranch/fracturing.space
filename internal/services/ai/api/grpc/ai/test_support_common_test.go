package ai

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"sort"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providerconnect"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSealer = aifakes.Sealer

type fakeStore struct {
	*aifakes.CredentialStore
	*aifakes.AgentStore
	*aifakes.AccessRequestStore
	*aifakes.ProviderGrantStore
	*aifakes.ProviderConnectSessionStore
	*aifakes.CampaignArtifactStore
	*aifakes.AuditEventStore
}

// ListAccessibleAgents overrides the embedded AgentStore method to include
// agents reachable via approved invoke access requests.
func (s *fakeStore) ListAccessibleAgents(_ context.Context, userID string, pageSize int, pageToken string) (agent.Page, error) {
	seen := make(map[string]struct{})
	items := make([]agent.Agent, 0)

	for _, rec := range s.Agents {
		if rec.OwnerUserID == userID {
			items = append(items, rec)
			seen[rec.ID] = struct{}{}
		}
	}

	for _, ar := range s.AccessRequests {
		if ar.RequesterUserID != userID || ar.Scope != "invoke" || ar.Status != "approved" {
			continue
		}
		if _, ok := seen[ar.AgentID]; ok {
			continue
		}
		a, ok := s.Agents[ar.AgentID]
		if !ok || a.OwnerUserID != ar.OwnerUserID {
			continue
		}
		items = append(items, a)
		seen[ar.AgentID] = struct{}{}
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	start := 0
	if pageToken != "" {
		for i, rec := range items {
			if rec.ID > pageToken {
				start = i
				break
			}
			if i == len(items)-1 {
				start = len(items)
			}
		}
	}
	items = items[start:]

	if pageSize > 0 && len(items) > pageSize {
		nextToken := items[pageSize-1].ID
		return agent.Page{Agents: items[:pageSize], NextPageToken: nextToken}, nil
	}
	return agent.Page{Agents: items}, nil
}

type fakeCampaignAIAuthStateClient struct {
	usageByAgent map[string]int32
	usageErr     error
	authState    *gamev1.GetCampaignAIAuthStateResponse
	authStateErr error
}

func (f *fakeCampaignAIAuthStateClient) ActiveCampaignCount(_ context.Context, agentID string) (int32, error) {
	if f.usageErr != nil {
		return 0, f.usageErr
	}
	return f.usageByAgent[agentID], nil
}

func (f *fakeCampaignAIAuthStateClient) CampaignAuthState(context.Context, string) (*gamev1.GetCampaignAIAuthStateResponse, error) {
	if f.authStateErr != nil {
		return nil, f.authStateErr
	}
	if f.authState == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	return f.authState, nil
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		CredentialStore:             aifakes.NewCredentialStore(),
		AgentStore:                  aifakes.NewAgentStore(),
		AccessRequestStore:          aifakes.NewAccessRequestStore(),
		ProviderGrantStore:          aifakes.NewProviderGrantStore(),
		ProviderConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
		CampaignArtifactStore:       aifakes.NewCampaignArtifactStore(),
		AuditEventStore:             aifakes.NewAuditEventStore(),
	}
}

func (s *fakeStore) FinishProviderConnect(_ context.Context, grant providergrant.ProviderGrant, completedSession providerconnect.Session) error {
	session, ok := s.ConnectSessions[completedSession.ID]
	if !ok || session.OwnerUserID != completedSession.OwnerUserID || session.Status != providerconnect.StatusPending || completedSession.Status != providerconnect.StatusCompleted {
		return storage.ErrNotFound
	}
	s.ProviderGrants[grant.ID] = grant
	session.Status = completedSession.Status
	session.CompletedAt = completedSession.CompletedAt
	session.UpdatedAt = completedSession.UpdatedAt
	s.ConnectSessions[completedSession.ID] = session
	return nil
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func testAISessionGrantConfig() aisessiongrant.Config {
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	return aisessiongrant.Config{
		Issuer:   "fracturing-space-game",
		Audience: "fracturing-space-ai",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      10 * time.Minute,
		Now: func() time.Time {
			return now
		},
	}
}

func mustIssueAISessionGrant(t *testing.T, cfg aisessiongrant.Config, input aisessiongrant.IssueInput) string {
	t.Helper()
	token, _, err := aisessiongrant.Issue(cfg, input)
	if err != nil {
		t.Fatalf("issue ai session grant: %v", err)
	}
	return token
}

// pkceCodeChallengeS256 computes the S256 code challenge for test assertions.
func pkceCodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func assertStatusReason(t *testing.T, err error, want apperrors.Code) {
	t.Helper()
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			if info.Reason != string(want) {
				t.Fatalf("expected reason %q, got %q", want, info.Reason)
			}
			return
		}
	}
	t.Fatalf("missing ErrorInfo detail in %v", err)
}

var (
	_ storage.AgentStore = (*fakeStore)(nil)
)
