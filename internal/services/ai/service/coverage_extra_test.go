package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type accessRequestStoreWithTransitionErr struct {
	*aifakes.AccessRequestStore
	reviewErr error
	revokeErr error
}

func (s *accessRequestStoreWithTransitionErr) ReviewAccessRequest(ctx context.Context, reviewed accessrequest.AccessRequest) error {
	if s.reviewErr != nil {
		return s.reviewErr
	}
	return s.AccessRequestStore.ReviewAccessRequest(ctx, reviewed)
}

func (s *accessRequestStoreWithTransitionErr) RevokeAccessRequest(ctx context.Context, revoked accessrequest.AccessRequest) error {
	if s.revokeErr != nil {
		return s.revokeErr
	}
	return s.AccessRequestStore.RevokeAccessRequest(ctx, revoked)
}

type pagedAgentStore struct {
	pages map[string]agent.Page
	err   error
}

func (s *pagedAgentStore) PutAgent(context.Context, agent.Agent) error { return nil }
func (s *pagedAgentStore) GetAgent(context.Context, string) (agent.Agent, error) {
	return agent.Agent{}, storage.ErrNotFound
}
func (s *pagedAgentStore) ListAgentsByOwner(_ context.Context, _ string, _ int, pageToken string) (agent.Page, error) {
	if s.err != nil {
		return agent.Page{}, s.err
	}
	return s.pages[pageToken], nil
}
func (s *pagedAgentStore) ListAccessibleAgents(context.Context, string, int, string) (agent.Page, error) {
	return agent.Page{}, nil
}
func (s *pagedAgentStore) DeleteAgent(context.Context, string, string) error { return nil }

func TestAccessRequestServiceCoverageBranches(t *testing.T) {
	t.Run("constructor requires dependencies", func(t *testing.T) {
		tests := []struct {
			name string
			cfg  AccessRequestServiceConfig
		}{
			{name: "missing agent store"},
			{name: "missing access request store", cfg: AccessRequestServiceConfig{AgentStore: aifakes.NewAgentStore()}},
			{name: "missing audit event store", cfg: AccessRequestServiceConfig{AgentStore: aifakes.NewAgentStore(), AccessRequestStore: aifakes.NewAccessRequestStore()}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if _, err := NewAccessRequestService(tt.cfg); err == nil {
					t.Fatal("expected constructor error")
				}
			})
		}
	})

	now := time.Date(2026, 3, 28, 13, 0, 0, 0, time.UTC)
	baseAgent := agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "owner-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-5-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}

	t.Run("create maps unavailable agent and store failures", func(t *testing.T) {
		t.Run("agent not found", func(t *testing.T) {
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: aifakes.NewAccessRequestStore(),
				AuditEventStore:    aifakes.NewAuditEventStore(),
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			_, err = svc.Create(context.Background(), CreateAccessRequestInput{
				RequesterUserID: "user-1",
				AgentID:         "agent-1",
				Scope:           "invoke",
			})
			if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
			}
		})

		t.Run("inactive agent", func(t *testing.T) {
			agentStore := aifakes.NewAgentStore()
			inactive := baseAgent
			inactive.Status = ""
			agentStore.Agents[inactive.ID] = inactive
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         agentStore,
				AccessRequestStore: aifakes.NewAccessRequestStore(),
				AuditEventStore:    aifakes.NewAuditEventStore(),
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			_, err = svc.Create(context.Background(), CreateAccessRequestInput{
				RequesterUserID: "user-1",
				AgentID:         inactive.ID,
				Scope:           "invoke",
			})
			if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
			}
		})

		t.Run("put access request failure", func(t *testing.T) {
			agentStore := aifakes.NewAgentStore()
			agentStore.Agents[baseAgent.ID] = baseAgent
			accessStore := aifakes.NewAccessRequestStore()
			accessStore.PutErr = errors.New("db down")
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         agentStore,
				AccessRequestStore: accessStore,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
				IDGenerator:        func() (string, error) { return "request-1", nil },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			_, err = svc.Create(context.Background(), CreateAccessRequestInput{
				RequesterUserID: "user-1",
				AgentID:         baseAgent.ID,
				Scope:           "invoke",
			})
			if got := ErrorKindOf(err); got != ErrKindInternal {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
			}
		})

		t.Run("audit write failure", func(t *testing.T) {
			agentStore := aifakes.NewAgentStore()
			agentStore.Agents[baseAgent.ID] = baseAgent
			auditStore := aifakes.NewAuditEventStore()
			auditStore.PutErr = errors.New("audit unavailable")
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         agentStore,
				AccessRequestStore: aifakes.NewAccessRequestStore(),
				AuditEventStore:    auditStore,
				Clock:              func() time.Time { return now },
				IDGenerator:        func() (string, error) { return "request-1", nil },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			_, err = svc.Create(context.Background(), CreateAccessRequestInput{
				RequesterUserID: "user-1",
				AgentID:         baseAgent.ID,
				Scope:           "invoke",
			})
			if got := ErrorKindOf(err); got != ErrKindInternal {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
			}
		})
	})

	t.Run("list maps invalid role and store failures", func(t *testing.T) {
		store := aifakes.NewAccessRequestStore()
		svc, err := NewAccessRequestService(AccessRequestServiceConfig{
			AgentStore:         aifakes.NewAgentStore(),
			AccessRequestStore: store,
			AuditEventStore:    aifakes.NewAuditEventStore(),
		})
		if err != nil {
			t.Fatalf("NewAccessRequestService: %v", err)
		}
		if _, err := svc.List(context.Background(), "user-1", 0, 10, ""); ErrorKindOf(err) != ErrKindInvalidArgument {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInvalidArgument)
		}
		store.ListErr = errors.New("list failed")
		if _, err := svc.List(context.Background(), "user-1", ListAccessRequestRoleRequester, 10, ""); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}
	})

	t.Run("list audit events wraps store failure", func(t *testing.T) {
		auditStore := aifakes.NewAuditEventStore()
		auditStore.ListErr = errors.New("audit list failed")
		svc, err := NewAccessRequestService(AccessRequestServiceConfig{
			AgentStore:         aifakes.NewAgentStore(),
			AccessRequestStore: aifakes.NewAccessRequestStore(),
			AuditEventStore:    auditStore,
		})
		if err != nil {
			t.Fatalf("NewAccessRequestService: %v", err)
		}
		if _, err := svc.ListAuditEvents(context.Background(), ListAuditEventsInput{OwnerUserID: "owner-1", PageSize: 10}); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}
	})

	pending := accessrequest.AccessRequest{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           accessrequest.ScopeInvoke,
		Status:          accessrequest.StatusPending,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	approved := pending
	approved.Status = accessrequest.StatusApproved
	approved.ReviewerUserID = "owner-1"
	approved.ReviewedAt = ptrTime(now.Add(-time.Minute))

	t.Run("review maps get, transition, and audit failures", func(t *testing.T) {
		t.Run("missing id", func(t *testing.T) {
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: aifakes.NewAccessRequestStore(),
				AuditEventStore:    aifakes.NewAuditEventStore(),
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Review(context.Background(), ReviewAccessRequestInput{}); ErrorKindOf(err) != ErrKindInvalidArgument {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInvalidArgument)
			}
		})

		t.Run("get internal", func(t *testing.T) {
			store := aifakes.NewAccessRequestStore()
			store.GetErr = errors.New("db read fail")
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Review(context.Background(), ReviewAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: "request-1", Decision: accessrequest.DecisionApprove}); ErrorKindOf(err) != ErrKindInternal {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
			}
		})

		t.Run("storage conflict", func(t *testing.T) {
			store := &accessRequestStoreWithTransitionErr{AccessRequestStore: aifakes.NewAccessRequestStore(), reviewErr: storage.ErrConflict}
			store.AccessRequests[pending.ID] = pending
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Review(context.Background(), ReviewAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: pending.ID, Decision: accessrequest.DecisionApprove}); ErrorKindOf(err) != ErrKindFailedPrecondition {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindFailedPrecondition)
			}
		})

		t.Run("storage not found", func(t *testing.T) {
			store := &accessRequestStoreWithTransitionErr{AccessRequestStore: aifakes.NewAccessRequestStore(), reviewErr: storage.ErrNotFound}
			store.AccessRequests[pending.ID] = pending
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Review(context.Background(), ReviewAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: pending.ID, Decision: accessrequest.DecisionApprove}); ErrorKindOf(err) != ErrKindNotFound {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindNotFound)
			}
		})

		t.Run("audit failure", func(t *testing.T) {
			store := aifakes.NewAccessRequestStore()
			store.AccessRequests[pending.ID] = pending
			auditStore := aifakes.NewAuditEventStore()
			auditStore.PutErr = errors.New("audit unavailable")
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    auditStore,
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Review(context.Background(), ReviewAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: pending.ID, Decision: accessrequest.DecisionApprove}); ErrorKindOf(err) != ErrKindInternal {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
			}
		})
	})

	t.Run("revoke maps get, transition, and audit failures", func(t *testing.T) {
		t.Run("missing id", func(t *testing.T) {
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: aifakes.NewAccessRequestStore(),
				AuditEventStore:    aifakes.NewAuditEventStore(),
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Revoke(context.Background(), RevokeAccessRequestInput{}); ErrorKindOf(err) != ErrKindInvalidArgument {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInvalidArgument)
			}
		})

		t.Run("get internal", func(t *testing.T) {
			store := aifakes.NewAccessRequestStore()
			store.GetErr = errors.New("db read fail")
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Revoke(context.Background(), RevokeAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: approved.ID}); ErrorKindOf(err) != ErrKindInternal {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
			}
		})

		t.Run("storage conflict", func(t *testing.T) {
			store := &accessRequestStoreWithTransitionErr{AccessRequestStore: aifakes.NewAccessRequestStore(), revokeErr: storage.ErrConflict}
			store.AccessRequests[approved.ID] = approved
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Revoke(context.Background(), RevokeAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: approved.ID}); ErrorKindOf(err) != ErrKindFailedPrecondition {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindFailedPrecondition)
			}
		})

		t.Run("storage not found", func(t *testing.T) {
			store := &accessRequestStoreWithTransitionErr{AccessRequestStore: aifakes.NewAccessRequestStore(), revokeErr: storage.ErrNotFound}
			store.AccessRequests[approved.ID] = approved
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    aifakes.NewAuditEventStore(),
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Revoke(context.Background(), RevokeAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: approved.ID}); ErrorKindOf(err) != ErrKindNotFound {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindNotFound)
			}
		})

		t.Run("audit failure", func(t *testing.T) {
			store := aifakes.NewAccessRequestStore()
			store.AccessRequests[approved.ID] = approved
			auditStore := aifakes.NewAuditEventStore()
			auditStore.PutErr = errors.New("audit unavailable")
			svc, err := NewAccessRequestService(AccessRequestServiceConfig{
				AgentStore:         aifakes.NewAgentStore(),
				AccessRequestStore: store,
				AuditEventStore:    auditStore,
				Clock:              func() time.Time { return now },
			})
			if err != nil {
				t.Fatalf("NewAccessRequestService: %v", err)
			}
			if _, err := svc.Revoke(context.Background(), RevokeAccessRequestInput{OwnerUserID: "owner-1", AccessRequestID: approved.ID}); ErrorKindOf(err) != ErrKindInternal {
				t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
			}
		})
	})
}

func TestUsagePolicyCoverageBranches(t *testing.T) {
	t.Run("anyMatchingAgentHasActiveCampaignUsage validates dependencies", func(t *testing.T) {
		reader := (*AuthReferenceUsageReader)(nil)
		if _, err := reader.anyMatchingAgentHasActiveCampaignUsage(context.Background(), "owner-1", func(agent.Agent) bool { return true }); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}

		reader = NewAuthReferenceUsageReader(&pagedAgentStore{}, nil)
		if _, err := reader.anyMatchingAgentHasActiveCampaignUsage(context.Background(), "owner-1", func(agent.Agent) bool { return true }); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}

		reader = NewAuthReferenceUsageReader(&pagedAgentStore{}, NewAgentBindingUsageReader(&fakeCampaignUsageReader{}))
		inUse, err := reader.anyMatchingAgentHasActiveCampaignUsage(context.Background(), "owner-1", nil)
		if err != nil {
			t.Fatalf("anyMatchingAgentHasActiveCampaignUsage(nil match): %v", err)
		}
		if inUse {
			t.Fatal("expected nil match to report false")
		}
	})

	t.Run("anyMatchingAgentHasActiveCampaignUsage handles pagination and list errors", func(t *testing.T) {
		reader := NewAuthReferenceUsageReader(&pagedAgentStore{err: errors.New("list failed")}, NewAgentBindingUsageReader(&fakeCampaignUsageReader{}))
		if _, err := reader.anyMatchingAgentHasActiveCampaignUsage(context.Background(), "owner-1", func(agent.Agent) bool { return true }); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}

		repeating := &pagedAgentStore{
			pages: map[string]agent.Page{
				"": {Agents: []agent.Agent{{ID: "agent-1"}}, NextPageToken: ""},
			},
		}
		reader = NewAuthReferenceUsageReader(repeating, NewAgentBindingUsageReader(&fakeCampaignUsageReader{}))
		inUse, err := reader.anyMatchingAgentHasActiveCampaignUsage(context.Background(), "owner-1", func(a agent.Agent) bool { return a.ID == "agent-1" })
		if err != nil {
			t.Fatalf("anyMatchingAgentHasActiveCampaignUsage(repeating): %v", err)
		}
		if inUse {
			t.Fatal("expected false when matching agent has no usage")
		}

		paged := &pagedAgentStore{
			pages: map[string]agent.Page{
				"":       {Agents: []agent.Agent{{ID: "agent-1"}}, NextPageToken: "page-2"},
				"page-2": {Agents: []agent.Agent{{ID: "agent-2"}}},
			},
		}
		reader = NewAuthReferenceUsageReader(paged, NewAgentBindingUsageReader(&fakeCampaignUsageReader{
			usageByAgent: map[string]int32{"agent-2": 2},
		}))
		inUse, err = reader.anyMatchingAgentHasActiveCampaignUsage(context.Background(), "owner-1", func(a agent.Agent) bool {
			return strings.HasSuffix(a.ID, "2")
		})
		if err != nil {
			t.Fatalf("anyMatchingAgentHasActiveCampaignUsage(paged): %v", err)
		}
		if !inUse {
			t.Fatal("expected second-page usage to be detected")
		}
	})

	t.Run("policy guards nil readers and active usage", func(t *testing.T) {
		now := time.Date(2026, 3, 28, 14, 0, 0, 0, time.UTC)
		if err := (*UsagePolicy)(nil).EnsureCredentialNotBoundToActiveCampaigns(context.Background(), "owner-1", "cred-1"); err != nil {
			t.Fatalf("nil policy credential guard: %v", err)
		}
		if err := (*UsagePolicy)(nil).EnsureProviderGrantNotBoundToActiveCampaigns(context.Background(), "owner-1", "grant-1"); err != nil {
			t.Fatalf("nil policy provider grant guard: %v", err)
		}

		policy := NewUsagePolicy(UsagePolicyConfig{})
		if got := ErrorKindOf(policy.EnsureCredentialNotBoundToActiveCampaigns(context.Background(), "owner-1", "cred-1")); got != ErrKindInternal {
			t.Fatalf("credential guard kind = %v, want %v", got, ErrKindInternal)
		}
		if got := ErrorKindOf(policy.EnsureProviderGrantNotBoundToActiveCampaigns(context.Background(), "owner-1", "grant-1")); got != ErrKindInternal {
			t.Fatalf("provider grant guard kind = %v, want %v", got, ErrKindInternal)
		}

		agentStore := aifakes.NewAgentStore()
		agentStore.Agents["agent-1"] = agent.Agent{
			ID:            "agent-1",
			OwnerUserID:   "owner-1",
			Label:         "narrator",
			Provider:      provider.OpenAI,
			Model:         "gpt-5-mini",
			AuthReference: agent.ProviderGrantAuthReference("grant-1"),
			Status:        agent.StatusActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		usageReader := NewAgentBindingUsageReader(&fakeCampaignUsageReader{usageByAgent: map[string]int32{"agent-1": 1}})
		policy = NewUsagePolicy(UsagePolicyConfig{
			AgentBindingUsageReader:  usageReader,
			AuthReferenceUsageReader: NewAuthReferenceUsageReader(agentStore, usageReader),
		})
		if got := ErrorKindOf(policy.EnsureProviderGrantNotBoundToActiveCampaigns(context.Background(), "owner-1", "grant-1")); got != ErrKindFailedPrecondition {
			t.Fatalf("provider grant guard kind = %v, want %v", got, ErrKindFailedPrecondition)
		}
	})
}

func TestProviderGrantAndErrorHelpersCoverage(t *testing.T) {
	t.Run("provider grant constructor requires dependencies", func(t *testing.T) {
		tests := []struct {
			name string
			cfg  ProviderGrantServiceConfig
		}{
			{name: "missing provider grant store"},
			{name: "missing connect session store", cfg: ProviderGrantServiceConfig{ProviderGrantStore: aifakes.NewProviderGrantStore()}},
			{name: "missing connect finisher", cfg: ProviderGrantServiceConfig{ProviderGrantStore: aifakes.NewProviderGrantStore(), ConnectSessionStore: aifakes.NewProviderConnectSessionStore()}},
			{name: "missing sealer", cfg: ProviderGrantServiceConfig{ProviderGrantStore: aifakes.NewProviderGrantStore(), ConnectSessionStore: aifakes.NewProviderConnectSessionStore(), ConnectFinisher: newProviderGrantTestConnectFinisher(aifakes.NewProviderGrantStore(), aifakes.NewProviderConnectSessionStore())}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if _, err := NewProviderGrantService(tt.cfg); err == nil {
					t.Fatal("expected constructor error")
				}
			})
		}
	})

	t.Run("pkce helper functions", func(t *testing.T) {
		verifier, err := generatePKCECodeVerifier()
		if err != nil {
			t.Fatalf("generatePKCECodeVerifier: %v", err)
		}
		if !isValidPKCECodeVerifier(verifier) {
			t.Fatalf("generated verifier is invalid: %q", verifier)
		}
		if len(verifier) < 43 || len(verifier) > 128 {
			t.Fatalf("generated verifier length = %d", len(verifier))
		}
		if isValidPKCECodeVerifier("short") {
			t.Fatal("expected short verifier to be invalid")
		}
		if isValidPKCECodeVerifier(strings.Repeat("a", 129)) {
			t.Fatal("expected long verifier to be invalid")
		}
		if isValidPKCECodeVerifier("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN$012345") {
			t.Fatal("expected invalid character verifier to be invalid")
		}
		if got := pkceCodeChallengeS256("abc"); got != "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0" {
			t.Fatalf("pkceCodeChallengeS256(abc) = %q", got)
		}
		if got := hashState("state-1"); len(got) != 64 {
			t.Fatalf("hashState length = %d, want 64", len(got))
		}
	})

	t.Run("provider grant revoke maps store and upstream failures", func(t *testing.T) {
		now := time.Date(2026, 3, 28, 13, 30, 0, 0, time.UTC)
		providerGrantStore := aifakes.NewProviderGrantStore()
		providerGrantStore.ProviderGrants["grant-1"] = providergrant.ProviderGrant{
			ID:              "grant-1",
			OwnerUserID:     "owner-1",
			Provider:        provider.OpenAI,
			GrantedScopes:   []string{"responses.read"},
			TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
			Status:          providergrant.StatusActive,
			CreatedAt:       now.Add(-time.Hour),
			UpdatedAt:       now.Add(-time.Hour),
		}
		oauthAdapter := &fakeOAuthAdapter{revokeErr: errors.New("revoke failed")}
		svc := mustNewProviderGrantService(t, ProviderGrantServiceConfig{
			ProviderGrantStore:  providerGrantStore,
			ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
			ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, aifakes.NewProviderConnectSessionStore()),
			Sealer:              &aifakes.Sealer{},
			ProviderRegistry: mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{
				provider.OpenAI: oauthAdapter,
			}, nil, nil, nil),
			Clock: func() time.Time { return now },
		})
		if _, err := svc.Revoke(context.Background(), "owner-1", "grant-1"); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}

		providerGrantStore = aifakes.NewProviderGrantStore()
		providerGrantStore.GetErr = errors.New("db get failed")
		svc = mustNewProviderGrantService(t, ProviderGrantServiceConfig{
			ProviderGrantStore:  providerGrantStore,
			ConnectSessionStore: aifakes.NewProviderConnectSessionStore(),
			ConnectFinisher:     newProviderGrantTestConnectFinisher(providerGrantStore, aifakes.NewProviderConnectSessionStore()),
			Sealer:              &aifakes.Sealer{},
			ProviderRegistry:    mustProviderRegistryForTests(t, map[provider.Provider]provideroauth.Adapter{provider.OpenAI: nil}, nil, nil, nil),
		})
		if _, err := svc.Revoke(context.Background(), "owner-1", "grant-1"); ErrorKindOf(err) != ErrKindInternal {
			t.Fatalf("ErrorKindOf(err) = %v, want %v", ErrorKindOf(err), ErrKindInternal)
		}
	})

	t.Run("wrapped service errors unwrap", func(t *testing.T) {
		cause := errors.New("boom")
		err := Wrapf(ErrKindInternal, cause, "wrapped")
		if !errors.Is(err, cause) {
			t.Fatal("expected wrapped error to unwrap to cause")
		}
		if err.Unwrap() != cause {
			t.Fatalf("Unwrap() = %v, want %v", err.Unwrap(), cause)
		}
	})
}
