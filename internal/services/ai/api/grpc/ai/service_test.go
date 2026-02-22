package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/testkit/aifakes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeSealer = aifakes.Sealer
type fakeStore = aifakes.Store

func newFakeStore() *fakeStore {
	return aifakes.NewStore()
}

type fakeProviderOAuthAdapter struct {
	buildAuthorizationURLErr error
	exchangeErr              error
	exchangeResult           ProviderTokenExchangeResult
	refreshErr               error
	refreshResult            ProviderTokenExchangeResult
	revokeErr                error

	lastAuthorizationInput ProviderAuthorizationURLInput
	lastRefreshToken       string
	lastRevokedToken       string
}

func (f *fakeProviderOAuthAdapter) BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error) {
	f.lastAuthorizationInput = input
	if f.buildAuthorizationURLErr != nil {
		return "", f.buildAuthorizationURLErr
	}
	return "https://provider.example.com/auth", nil
}

func (f *fakeProviderOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, _ ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error) {
	if f.exchangeErr != nil {
		return ProviderTokenExchangeResult{}, f.exchangeErr
	}
	if strings.TrimSpace(f.exchangeResult.TokenPlaintext) == "" {
		return ProviderTokenExchangeResult{TokenPlaintext: `{"access_token":"at-1","refresh_token":"rt-1"}`, RefreshSupported: true}, nil
	}
	return f.exchangeResult, nil
}

func (f *fakeProviderOAuthAdapter) RefreshToken(_ context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error) {
	f.lastRefreshToken = input.RefreshToken
	if f.refreshErr != nil {
		return ProviderTokenExchangeResult{}, f.refreshErr
	}
	return f.refreshResult, nil
}

func (f *fakeProviderOAuthAdapter) RevokeToken(_ context.Context, input ProviderRevokeTokenInput) error {
	f.lastRevokedToken = input.Token
	return f.revokeErr
}

type fakeProviderInvocationAdapter struct {
	invokeErr    error
	invokeResult ProviderInvokeResult
	lastInput    ProviderInvokeInput
}

func (f *fakeProviderInvocationAdapter) Invoke(_ context.Context, input ProviderInvokeInput) (ProviderInvokeResult, error) {
	f.lastInput = input
	if f.invokeErr != nil {
		return ProviderInvokeResult{}, f.invokeErr
	}
	return f.invokeResult, nil
}

func TestCreateCredentialRequiresUserID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.CreateCredential(context.Background(), &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestCreateCredentialEncryptsSecret(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 50, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "cred-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if resp.GetCredential().GetId() != "cred-1" {
		t.Fatalf("credential id = %q, want %q", resp.GetCredential().GetId(), "cred-1")
	}

	stored := store.Credentials["cred-1"]
	if stored.SecretCiphertext != "enc:sk-1" {
		t.Fatalf("stored ciphertext = %q, want %q", stored.SecretCiphertext, "enc:sk-1")
	}
}

func TestListCredentialsReturnsOwnerRecords(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.Credentials["cred-2"] = storage.CredentialRecord{ID: "cred-2", OwnerUserID: "user-2", Provider: "openai", Label: "B", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list Credentials: %v", err)
	}
	if len(resp.GetCredentials()) != 1 {
		t.Fatalf("Credentials len = %d, want 1", len(resp.GetCredentials()))
	}
	if resp.GetCredentials()[0].GetId() != "cred-1" {
		t.Fatalf("credential id = %q, want %q", resp.GetCredentials()[0].GetId(), "cred-1")
	}
}

func TestRevokeCredential(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 55, 0, 0, time.UTC) }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{CredentialId: "cred-1"})
	if err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	if resp.GetCredential().GetStatus() != aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", resp.GetCredential().GetStatus())
	}
}

func TestCreateAgentRequiresActiveOwnedCredential(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "revoked", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Name:         "Narrator",
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        "gpt-4o-mini",
		CredentialId: "cred-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateAgentSuccess(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 57, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "agent-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Name:         "Narrator",
		Provider:     aiv1.Provider_PROVIDER_OPENAI,
		Model:        "gpt-4o-mini",
		CredentialId: "cred-1",
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if resp.GetAgent().GetId() != "agent-1" {
		t.Fatalf("agent id = %q, want %q", resp.GetAgent().GetId(), "agent-1")
	}
}

func TestCreateAgentWithProviderGrantSuccess(t *testing.T) {
	store := newFakeStore()
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 22, 57, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "agent-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Name:            "Narrator",
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		Model:           "gpt-4o-mini",
		ProviderGrantId: "grant-1",
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if resp.GetAgent().GetId() != "agent-1" {
		t.Fatalf("agent id = %q, want %q", resp.GetAgent().GetId(), "agent-1")
	}
	stored := store.Agents["agent-1"]
	if stored.ProviderGrantID != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", stored.ProviderGrantID, "grant-1")
	}
	if stored.CredentialID != "" {
		t.Fatalf("credential_id = %q, want empty", stored.CredentialID)
	}
}

func TestCreateAgentRejectsMultipleAuthReferences(t *testing.T) {
	store := newFakeStore()
	store.Credentials["cred-1"] = storage.CredentialRecord{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAgent(ctx, &aiv1.CreateAgentRequest{
		Name:            "Narrator",
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		Model:           "gpt-4o-mini",
		CredentialId:    "cred-1",
		ProviderGrantId: "grant-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAgentsReturnsOwnerRecords(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-1",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.Agents["agent-2"] = storage.AgentRecord{
		ID:              "agent-2",
		OwnerUserID:     "user-2",
		Name:            "Planner",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-2",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListAgents(ctx, &aiv1.ListAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list Agents: %v", err)
	}
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("Agents len = %d, want 1", len(resp.GetAgents()))
	}
	if got := resp.GetAgents()[0].GetId(); got != "agent-1" {
		t.Fatalf("agent id = %q, want %q", got, "agent-1")
	}
}

func TestListAccessibleAgentsIncludesOwnedAndApprovedShared(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-own-1"] = storage.AgentRecord{
		ID:           "agent-own-1",
		OwnerUserID:  "user-1",
		Name:         "Owner Agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-own-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-shared-1"] = storage.AgentRecord{
		ID:           "agent-shared-1",
		OwnerUserID:  "user-owner",
		Name:         "Shared Agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-shared-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-approved"] = storage.AccessRequestRecord{
		ID:              "request-approved",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-shared-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list accessible Agents: %v", err)
	}
	if len(resp.GetAgents()) != 2 {
		t.Fatalf("Agents len = %d, want 2", len(resp.GetAgents()))
	}
	got := []string{resp.GetAgents()[0].GetId(), resp.GetAgents()[1].GetId()}
	want := []string{"agent-own-1", "agent-shared-1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("agent[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestListAccessibleAgentsExcludesPendingDeniedAndStale(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-approved"] = storage.AgentRecord{
		ID:           "agent-approved",
		OwnerUserID:  "owner-1",
		Name:         "Approved",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-pending"] = storage.AgentRecord{
		ID:           "agent-pending",
		OwnerUserID:  "owner-1",
		Name:         "Pending",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-2",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-denied"] = storage.AgentRecord{
		ID:           "agent-denied",
		OwnerUserID:  "owner-1",
		Name:         "Denied",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-3",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-approved"] = storage.AccessRequestRecord{
		ID:              "request-approved",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-approved",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-pending"] = storage.AccessRequestRecord{
		ID:              "request-pending",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-pending",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-denied"] = storage.AccessRequestRecord{
		ID:              "request-denied",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-denied",
		Scope:           "invoke",
		Status:          "denied",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-stale-agent"] = storage.AccessRequestRecord{
		ID:              "request-stale-agent",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-missing",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-wrong-owner"] = storage.AccessRequestRecord{
		ID:              "request-wrong-owner",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-other",
		AgentID:         "agent-approved",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list accessible Agents: %v", err)
	}
	if len(resp.GetAgents()) != 1 {
		t.Fatalf("Agents len = %d, want 1", len(resp.GetAgents()))
	}
	if got := resp.GetAgents()[0].GetId(); got != "agent-approved" {
		t.Fatalf("agent id = %q, want %q", got, "agent-approved")
	}
}

func TestListAccessibleAgentsRequiresUserID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.ListAccessibleAgents(context.Background(), &aiv1.ListAccessibleAgentsRequest{PageSize: 10})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListAccessibleAgentsPaginatesByAgentID(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-a"] = storage.AgentRecord{
		ID:           "agent-a",
		OwnerUserID:  "user-1",
		Name:         "A",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-a",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-b"] = storage.AgentRecord{
		ID:           "agent-b",
		OwnerUserID:  "owner-1",
		Name:         "B",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-b",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Agents["agent-c"] = storage.AgentRecord{
		ID:           "agent-c",
		OwnerUserID:  "owner-2",
		Name:         "C",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-c",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-b"] = storage.AccessRequestRecord{
		ID:              "request-b",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-b",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.AccessRequests["request-c"] = storage.AccessRequestRecord{
		ID:              "request-c",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-2",
		AgentID:         "agent-c",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	first, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("list accessible Agents first page: %v", err)
	}
	if len(first.GetAgents()) != 2 {
		t.Fatalf("first page Agents len = %d, want 2", len(first.GetAgents()))
	}
	if got := first.GetAgents()[0].GetId(); got != "agent-a" {
		t.Fatalf("first page agent[0] = %q, want %q", got, "agent-a")
	}
	if got := first.GetAgents()[1].GetId(); got != "agent-b" {
		t.Fatalf("first page agent[1] = %q, want %q", got, "agent-b")
	}
	if got := first.GetNextPageToken(); got != "agent-b" {
		t.Fatalf("first page next token = %q, want %q", got, "agent-b")
	}

	second, err := svc.ListAccessibleAgents(ctx, &aiv1.ListAccessibleAgentsRequest{
		PageSize:  2,
		PageToken: first.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("list accessible Agents second page: %v", err)
	}
	if len(second.GetAgents()) != 1 {
		t.Fatalf("second page Agents len = %d, want 1", len(second.GetAgents()))
	}
	if got := second.GetAgents()[0].GetId(); got != "agent-c" {
		t.Fatalf("second page agent[0] = %q, want %q", got, "agent-c")
	}
	if got := second.GetNextPageToken(); got != "" {
		t.Fatalf("second page next token = %q, want empty", got)
	}
}

func TestGetAccessibleAgentRequiresUserID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.GetAccessibleAgent(context.Background(), &aiv1.GetAccessibleAgentRequest{
		AgentId: "agent-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetAccessibleAgentRequiresAgentID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAccessibleAgentMissingAgent(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetAccessibleAgentOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Owner Agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-1"})
	if err != nil {
		t.Fatalf("get accessible agent: %v", err)
	}
	if got := resp.GetAgent().GetId(); got != "agent-1" {
		t.Fatalf("agent id = %q, want %q", got, "agent-1")
	}
}

func TestGetAccessibleAgentApprovedRequester(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)
	store.Agents["agent-shared"] = storage.AgentRecord{
		ID:           "agent-shared",
		OwnerUserID:  "owner-1",
		Name:         "Shared Agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-owner",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-shared",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-shared"})
	if err != nil {
		t.Fatalf("get accessible agent: %v", err)
	}
	if got := resp.GetAgent().GetId(); got != "agent-shared" {
		t.Fatalf("agent id = %q, want %q", got, "agent-shared")
	}
}

func TestGetAccessibleAgentPendingRequesterHidden(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)
	store.Agents["agent-shared"] = storage.AgentRecord{
		ID:           "agent-shared",
		OwnerUserID:  "owner-1",
		Name:         "Shared Agent",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-owner",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-shared",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.GetAccessibleAgent(ctx, &aiv1.GetAccessibleAgentRequest{AgentId: "agent-shared"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestUpdateAgentSwitchesCredentialToProviderGrant(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 10, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-1",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.UpdateAgent(ctx, &aiv1.UpdateAgentRequest{
		AgentId:         "agent-1",
		ProviderGrantId: "grant-1",
		Model:           "gpt-4o",
	})
	if err != nil {
		t.Fatalf("update agent: %v", err)
	}

	if got := resp.GetAgent().GetCredentialId(); got != "" {
		t.Fatalf("credential_id = %q, want empty", got)
	}
	if got := resp.GetAgent().GetProviderGrantId(); got != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", got, "grant-1")
	}
	if got := resp.GetAgent().GetModel(); got != "gpt-4o" {
		t.Fatalf("model = %q, want %q", got, "gpt-4o")
	}
}

func TestDeleteAgentRemovesOwnedRecord(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "cred-1",
		ProviderGrantID: "",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	if _, err := svc.DeleteAgent(ctx, &aiv1.DeleteAgentRequest{AgentId: "agent-1"}); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	if _, ok := store.Agents["agent-1"]; ok {
		t.Fatal("agent should be deleted")
	}
}

func TestCreateCredentialSealError(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{SealErr: errors.New("seal fail")})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    "Main",
		Secret:   "sk-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentRequiresUserID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.InvokeAgent(context.Background(), &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestInvokeAgentRequiresRequest(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresAgentID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: " ",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresInput(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   " ",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestInvokeAgentRequiresActiveOwnedCredential(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "revoked",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentProviderGrantPathWithoutCredentialStore(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := NewService(nil, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
}

func TestInvokeAgentCredentialPathWithoutCredentialStore(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	svc := NewService(nil, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentSuccess(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	adapter := &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{
			OutputText: "Hello from AI",
		},
	}
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = adapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if adapter.lastInput.CredentialSecret != "sk-1" {
		t.Fatalf("credential secret = %q, want %q", adapter.lastInput.CredentialSecret, "sk-1")
	}
	if adapter.lastInput.Model != "gpt-4o-mini" {
		t.Fatalf("model = %q, want %q", adapter.lastInput.Model, "gpt-4o-mini")
	}
	if adapter.lastInput.Input != "Say hello" {
		t.Fatalf("input = %q, want %q", adapter.lastInput.Input, "Say hello")
	}
	if resp.GetOutputText() != "Hello from AI" {
		t.Fatalf("output_text = %q, want %q", resp.GetOutputText(), "Hello from AI")
	}
}

func TestInvokeAgentWithProviderGrantRefreshesNearExpiry(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Second)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "active",
		ExpiresAt:        &expiresAt,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	oauthAdapter := &fakeProviderOAuthAdapter{
		refreshResult: ProviderTokenExchangeResult{
			TokenPlaintext:   `{"access_token":"at-2","refresh_token":"rt-2"}`,
			RefreshSupported: true,
			ExpiresAt:        ptrTime(now.Add(time.Hour)),
		},
	}
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{
			OutputText: "Hello from refreshed grant",
		},
	}
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = oauthAdapter
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if oauthAdapter.lastRefreshToken != "rt-1" {
		t.Fatalf("refresh token = %q, want %q", oauthAdapter.lastRefreshToken, "rt-1")
	}
	if invokeAdapter.lastInput.CredentialSecret != "at-2" {
		t.Fatalf("invoke auth token = %q, want %q", invokeAdapter.lastInput.CredentialSecret, "at-2")
	}
	if resp.GetOutputText() != "Hello from refreshed grant" {
		t.Fatalf("output_text = %q, want %q", resp.GetOutputText(), "Hello from refreshed grant")
	}
}

func TestInvokeAgentWithRefreshFailedGrantWithoutRefreshSupport(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: false,
		Status:           "refresh_failed",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentWithExpiredGrantWithoutRefreshSupport(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Minute)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: false,
		Status:           "active",
		ExpiresAt:        &expiresAt,
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello"},
	}
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if got := strings.TrimSpace(invokeAdapter.lastInput.CredentialSecret); got != "" {
		t.Fatalf("invoke adapter should not be called, got credential secret %q", got)
	}
}

func TestInvokeAgentWithRefreshFailedGrantRefreshesAndInvokes(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "refresh_failed",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = &fakeProviderOAuthAdapter{
		refreshResult: ProviderTokenExchangeResult{
			TokenPlaintext:   `{"access_token":"at-2","refresh_token":"rt-2"}`,
			RefreshSupported: true,
			ExpiresAt:        ptrTime(now.Add(time.Hour)),
		},
	}
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello"},
	}
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	if _, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	}); err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if got := invokeAdapter.lastInput.CredentialSecret; got != "at-2" {
		t.Fatalf("credential secret = %q, want %q", got, "at-2")
	}
}

func TestInvokeAgentWithProviderGrantMissingAccessToken(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:              "agent-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		TokenCiphertext:  `enc:{"refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentMissingAgentIsNotFound(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "missing",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentHiddenForNonOwner(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-2",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentApprovedRequesterCanInvokeOwnerAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	invokeAdapter := &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Shared response"},
	}
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = invokeAdapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if got := resp.GetOutputText(); got != "Shared response" {
		t.Fatalf("output_text = %q, want %q", got, "Shared response")
	}
	if got := invokeAdapter.lastInput.CredentialSecret; got != "sk-owner" {
		t.Fatalf("credential secret = %q, want %q", got, "sk-owner")
	}
}

func TestInvokeAgentApprovedRequesterWritesAuditEvent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Shared response"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	if _, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	}); err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "agent.invoke.shared" {
		t.Fatalf("audit event = %q, want %q", got, "agent.invoke.shared")
	}
}

func TestInvokeAgentSharedAccessUsesTargetedLookup(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.ListAccessRequestsByRequesterErr = errors.New("unexpected requester-wide list call")
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "owner-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "owner-1",
		Provider:         "openai",
		Label:            "Primary",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	adapter := &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Hello from shared access"},
	}
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = adapter

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "hello",
	})
	if err != nil {
		t.Fatalf("invoke agent: %v", err)
	}
	if got := resp.GetOutputText(); got != "Hello from shared access" {
		t.Fatalf("output_text = %q, want %q", got, "Hello from shared access")
	}
	if store.ListAccessRequestsByRequesterCalls != 0 {
		t.Fatalf("requester-wide list calls = %d, want 0", store.ListAccessRequestsByRequesterCalls)
	}
	if store.GetApprovedInvokeAccessCalls == 0 {
		t.Fatal("expected targeted approved invoke lookup to be used")
	}
}

func TestInvokeAgentApprovedRequesterDeniedAfterRevocation(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "Shared response"},
	}
	ownerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-owner"))
	if _, err := svc.RevokeAccessRequest(ownerCtx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "removed",
	}); err != nil {
		t.Fatalf("revoke access request: %v", err)
	}

	requesterCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	_, err := svc.InvokeAgent(requesterCtx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentPendingRequesterCannotInvokeOwnerAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentDeniedRequesterCannotInvokeOwnerAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-owner",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-owner",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-owner",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-requester",
		OwnerUserID:     "user-owner",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "denied",
		CreatedAt:       now.Add(-time.Minute),
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-requester"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestInvokeAgentMissingCredentialIsFailedPrecondition(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-missing",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentProviderInvokeError(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{invokeErr: errors.New("provider unavailable")}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentEmptyProviderOutputIsInternal(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.providerInvocationAdapters[providergrant.ProviderOpenAI] = &fakeProviderInvocationAdapter{
		invokeResult: ProviderInvokeResult{OutputText: "   "},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestInvokeAgentAdapterUnavailable(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	delete(svc.providerInvocationAdapters, providergrant.ProviderOpenAI)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestInvokeAgentSecretOpenError(t *testing.T) {
	store := newFakeStore()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:sk-1",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{OpenErr: errors.New("open fail")})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.InvokeAgent(ctx, &aiv1.InvokeAgentRequest{
		AgentId: "agent-1",
		Input:   "Say hello",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestStartProviderConnectRequiresUserID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.StartProviderConnect(context.Background(), &aiv1.StartProviderConnectRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestStartProviderConnectUsesS256CodeChallenge(t *testing.T) {
	store := newFakeStore()
	oauthAdapter := &fakeProviderOAuthAdapter{}
	svc := NewService(store, store, &fakeSealer{})
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = oauthAdapter

	idValues := []string{"session-1", "state-1"}
	svc.idGenerator = func() (string, error) {
		if len(idValues) == 0 {
			return "", errors.New("unexpected id call")
		}
		value := idValues[0]
		idValues = idValues[1:]
		return value, nil
	}

	codeVerifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	svc.codeVerifierGenerator = func() (string, error) {
		return codeVerifier, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	expectedChallenge := pkceCodeChallengeS256(codeVerifier)
	if got := oauthAdapter.lastAuthorizationInput.CodeChallenge; got != expectedChallenge {
		t.Fatalf("code challenge = %q, want %q", got, expectedChallenge)
	}
	if got := oauthAdapter.lastAuthorizationInput.CodeChallenge; got == codeVerifier {
		t.Fatalf("code challenge must not equal verifier: %q", got)
	}

	stored := store.ConnectSessions[resp.GetConnectSessionId()]
	if got := stored.CodeVerifierCiphertext; got != "enc:"+codeVerifier {
		t.Fatalf("stored code verifier ciphertext = %q, want %q", got, "enc:"+codeVerifier)
	}
}

func TestFinishProviderConnectCreatesProviderGrant(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 23, 30, 0, 0, time.UTC) }

	idValues := []string{"session-1", "state-1", "grant-1"}
	svc.idGenerator = func() (string, error) {
		if len(idValues) == 0 {
			return "", errors.New("unexpected id call")
		}
		value := idValues[0]
		idValues = idValues[1:]
		return value, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	finishResp, err := svc.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}
	if finishResp.GetProviderGrant().GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", finishResp.GetProviderGrant().GetId(), "grant-1")
	}
	stored := store.ProviderGrants["grant-1"]
	if stored.TokenCiphertext != "enc:token:auth-code-1" {
		t.Fatalf("stored token ciphertext = %q, want %q", stored.TokenCiphertext, "enc:token:auth-code-1")
	}
}

func TestFinishProviderConnectDoesNotSealRawAuthorizationCode(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 23, 30, 0, 0, time.UTC) }

	idValues := []string{"session-1", "state-1", "grant-1"}
	svc.idGenerator = func() (string, error) {
		if len(idValues) == 0 {
			return "", errors.New("unexpected id call")
		}
		value := idValues[0]
		idValues = idValues[1:]
		return value, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	_, err = svc.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}
	stored := store.ProviderGrants["grant-1"]
	if stored.TokenCiphertext == "enc:auth-code-1" {
		t.Fatalf("stored ciphertext should not seal raw authorization code: %q", stored.TokenCiphertext)
	}
}

func TestFinishProviderConnectKeepsSessionPendingOnExchangeFailure(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, store, &fakeSealer{})
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = &fakeProviderOAuthAdapter{
		exchangeErr: errors.New("exchange failed"),
	}

	idValues := []string{"session-1", "state-1"}
	svc.idGenerator = func() (string, error) {
		if len(idValues) == 0 {
			return "", errors.New("unexpected id call")
		}
		value := idValues[0]
		idValues = idValues[1:]
		return value, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svc.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	_, err = svc.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	assertStatusCode(t, err, codes.Internal)

	stored := store.ConnectSessions[startResp.GetConnectSessionId()]
	if stored.Status != "pending" {
		t.Fatalf("connect session status = %q, want %q", stored.Status, "pending")
	}
}

func TestListProviderGrantsReturnsOwnerRecords(t *testing.T) {
	store := newFakeStore()
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	store.ProviderGrants["grant-2"] = storage.ProviderGrantRecord{
		ID:              "grant-2",
		OwnerUserID:     "user-2",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if resp.GetProviderGrants()[0].GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", resp.GetProviderGrants()[0].GetId(), "grant-1")
	}
}

func TestListProviderGrantsFiltersByProvider(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-2"] = storage.ProviderGrantRecord{
		ID:              "grant-2",
		OwnerUserID:     "user-1",
		Provider:        "other",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-3"] = storage.ProviderGrantRecord{
		ID:              "grant-3",
		OwnerUserID:     "user-2",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:3",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Provider: aiv1.Provider_PROVIDER_OPENAI,
	})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if got := resp.GetProviderGrants()[0].GetId(); got != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", got, "grant-1")
	}
}

func TestListProviderGrantsFiltersByStatus(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 2, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-2"] = storage.ProviderGrantRecord{
		ID:              "grant-2",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:2",
		Status:          "revoked",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	store.ProviderGrants["grant-3"] = storage.ProviderGrantRecord{
		ID:              "grant-3",
		OwnerUserID:     "user-2",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:3",
		Status:          "revoked",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.ListProviderGrants(ctx, &aiv1.ListProviderGrantsRequest{
		PageSize: 10,
		Status:   aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED,
	})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(resp.GetProviderGrants()) != 1 {
		t.Fatalf("provider grants len = %d, want 1", len(resp.GetProviderGrants()))
	}
	if got := resp.GetProviderGrants()[0].GetId(); got != "grant-2" {
		t.Fatalf("provider grant id = %q, want %q", got, "grant-2")
	}
}

func TestRevokeProviderGrant(t *testing.T) {
	store := newFakeStore()
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	svc := NewService(store, store, &fakeSealer{})
	adapter := &fakeProviderOAuthAdapter{}
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = adapter
	svc.clock = func() time.Time { return time.Date(2026, 2, 15, 23, 31, 0, 0, time.UTC) }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	resp, err := svc.RevokeProviderGrant(ctx, &aiv1.RevokeProviderGrantRequest{ProviderGrantId: "grant-1"})
	if err != nil {
		t.Fatalf("revoke provider grant: %v", err)
	}
	if resp.GetProviderGrant().GetStatus() != aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", resp.GetProviderGrant().GetStatus())
	}
	if adapter.lastRevokedToken != "rt-1" {
		t.Fatalf("revoked token = %q, want %q", adapter.lastRevokedToken, "rt-1")
	}
}

func TestRefreshProviderGrantSuccessUpdatesStoredToken(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	adapter := &fakeProviderOAuthAdapter{
		refreshResult: ProviderTokenExchangeResult{
			TokenPlaintext:   `{"access_token":"at-2","refresh_token":"rt-2"}`,
			RefreshSupported: true,
			ExpiresAt:        ptrTime(now.Add(time.Hour)),
		},
	}
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = adapter

	got, err := svc.refreshProviderGrant(context.Background(), "user-1", "grant-1")
	if err != nil {
		t.Fatalf("refresh provider grant: %v", err)
	}
	if adapter.lastRefreshToken != "rt-1" {
		t.Fatalf("refresh token = %q, want %q", adapter.lastRefreshToken, "rt-1")
	}
	if got.TokenCiphertext != `enc:{"access_token":"at-2","refresh_token":"rt-2"}` {
		t.Fatalf("token ciphertext = %q", got.TokenCiphertext)
	}
	if got.LastRefreshedAt == nil || !got.LastRefreshedAt.Equal(now) {
		t.Fatalf("last_refreshed_at = %v, want %v", got.LastRefreshedAt, now)
	}
}

func TestRefreshProviderGrantMarksRefreshFailed(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	store.ProviderGrants["grant-1"] = storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  `enc:{"access_token":"at-1","refresh_token":"rt-1"}`,
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	adapter := &fakeProviderOAuthAdapter{refreshErr: errors.New("provider timeout")}
	svc.providerOAuthAdapters[providergrant.ProviderOpenAI] = adapter

	if _, err := svc.refreshProviderGrant(context.Background(), "user-1", "grant-1"); err == nil {
		t.Fatal("expected refresh error")
	}
	updated := store.ProviderGrants["grant-1"]
	if updated.Status != "refresh_failed" {
		t.Fatalf("status = %q, want %q", updated.Status, "refresh_failed")
	}
	if strings.TrimSpace(updated.LastRefreshError) == "" {
		t.Fatal("expected last_refresh_error to be set")
	}
}

func TestCreateAccessRequestSuccess(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "owner-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "request-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	resp, err := svc.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId:     "agent-1",
		Scope:       "invoke",
		RequestNote: "please allow",
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}
	if got := resp.GetAccessRequest().GetId(); got != "request-1" {
		t.Fatalf("id = %q, want %q", got, "request-1")
	}
	if got := resp.GetAccessRequest().GetStatus(); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_PENDING {
		t.Fatalf("status = %v, want pending", got)
	}
}

func TestCreateAccessRequestWritesAuditEvent(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "owner-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "request-1", nil }

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	if _, err := svc.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId:     "agent-1",
		Scope:       "invoke",
		RequestNote: "please allow",
	}); err != nil {
		t.Fatalf("create access request: %v", err)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "access_request.created" {
		t.Fatalf("audit event = %q, want %q", got, "access_request.created")
	}
}

func TestCreateAccessRequestRejectsOwnAgent(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId: "agent-1",
		Scope:   "invoke",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAccessRequestRejectsUnsupportedScope(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "owner-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	_, err := svc.CreateAccessRequest(ctx, &aiv1.CreateAccessRequestRequest{
		AgentId: "agent-1",
		Scope:   "admin",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAccessRequestsByRole(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now}
	store.AccessRequests["request-2"] = storage.AccessRequestRecord{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-2", AgentID: "agent-2", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now}
	store.AccessRequests["request-3"] = storage.AccessRequestRecord{ID: "request-3", RequesterUserID: "user-3", OwnerUserID: "owner-1", AgentID: "agent-3", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now}

	svc := NewService(store, store, &fakeSealer{})

	requesterCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	requesterResp, err := svc.ListAccessRequests(requesterCtx, &aiv1.ListAccessRequestsRequest{
		Role: aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_REQUESTER,
	})
	if err != nil {
		t.Fatalf("list requester access requests: %v", err)
	}
	if len(requesterResp.GetAccessRequests()) != 2 {
		t.Fatalf("requester len = %d, want 2", len(requesterResp.GetAccessRequests()))
	}

	ownerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	ownerResp, err := svc.ListAccessRequests(ownerCtx, &aiv1.ListAccessRequestsRequest{
		Role: aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_OWNER,
	})
	if err != nil {
		t.Fatalf("list owner access requests: %v", err)
	}
	if len(ownerResp.GetAccessRequests()) != 2 {
		t.Fatalf("owner len = %d, want 2", len(ownerResp.GetAccessRequests()))
	}
}

func TestListAuditEventsRequiresUserID(t *testing.T) {
	svc := NewService(newFakeStore(), newFakeStore(), &fakeSealer{})
	_, err := svc.ListAuditEvents(context.Background(), &aiv1.ListAuditEventsRequest{PageSize: 10})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListAuditEventsOwnerScoped(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 10, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", ActorUserID: "user-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "pending", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "approved", CreatedAt: now.Add(time.Minute)},
		{ID: "3", EventName: "access_request.created", ActorUserID: "user-2", OwnerUserID: "owner-2", RequesterUserID: "user-2", AgentID: "agent-2", AccessRequestID: "request-2", Outcome: "pending", CreatedAt: now},
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 2 {
		t.Fatalf("events len = %d, want 2", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "1" {
		t.Fatalf("event[0].id = %q, want %q", got, "1")
	}
	if got := resp.GetAuditEvents()[1].GetId(); got != "2" {
		t.Fatalf("event[1].id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsPaginates(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 10, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", ActorUserID: "user-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "pending", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "approved", CreatedAt: now.Add(time.Minute)},
		{ID: "3", EventName: "access_request.revoked", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "revoked", CreatedAt: now.Add(2 * time.Minute)},
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))

	first, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{PageSize: 2})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.GetAuditEvents()) != 2 {
		t.Fatalf("first page len = %d, want 2", len(first.GetAuditEvents()))
	}
	if got := first.GetNextPageToken(); got != "2" {
		t.Fatalf("first next token = %q, want %q", got, "2")
	}

	second, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:  2,
		PageToken: first.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.GetAuditEvents()) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.GetAuditEvents()))
	}
	if got := second.GetAuditEvents()[0].GetId(); got != "3" {
		t.Fatalf("second page id = %q, want %q", got, "3")
	}
	if got := second.GetNextPageToken(); got != "" {
		t.Fatalf("second next token = %q, want empty", got)
	}
}

func TestListAuditEventsFiltersByEventName(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now.Add(time.Minute)},
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:  10,
		EventName: "access_request.reviewed",
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "2" {
		t.Fatalf("event id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsFiltersByAgentID(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", OwnerUserID: "owner-1", AgentID: "agent-2", CreatedAt: now.Add(time.Minute)},
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize: 10,
		AgentId:  "agent-2",
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "2" {
		t.Fatalf("event id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsFiltersByTimeWindow(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	store.AuditEvents = []storage.AuditEventRecord{
		{ID: "1", EventName: "access_request.created", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now},
		{ID: "2", EventName: "access_request.reviewed", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now.Add(2 * time.Minute)},
		{ID: "3", EventName: "access_request.revoked", OwnerUserID: "owner-1", AgentID: "agent-1", CreatedAt: now.Add(4 * time.Minute)},
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:      10,
		CreatedAfter:  timestamppb.New(now.Add(time.Minute)),
		CreatedBefore: timestamppb.New(now.Add(3 * time.Minute)),
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(resp.GetAuditEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetAuditEvents()))
	}
	if got := resp.GetAuditEvents()[0].GetId(); got != "2" {
		t.Fatalf("event id = %q, want %q", got, "2")
	}
}

func TestListAuditEventsRejectsInvalidTimeWindow(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 3, 20, 0, 0, time.UTC)
	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	_, err := svc.ListAuditEvents(ctx, &aiv1.ListAuditEventsRequest{
		PageSize:      10,
		CreatedAfter:  timestamppb.New(now.Add(2 * time.Minute)),
		CreatedBefore: timestamppb.New(now),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestReviewAccessRequestByOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 7, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := svc.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
		Decision:        aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_APPROVE,
		ReviewNote:      "approved",
	})
	if err != nil {
		t.Fatalf("review access request: %v", err)
	}
	if got := resp.GetAccessRequest().GetStatus(); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_APPROVED {
		t.Fatalf("status = %v, want approved", got)
	}
}

func TestReviewAccessRequestWritesAuditEvent(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 7, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	if _, err := svc.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
		Decision:        aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_APPROVE,
		ReviewNote:      "approved",
	}); err != nil {
		t.Fatalf("review access request: %v", err)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "access_request.reviewed" {
		t.Fatalf("audit event = %q, want %q", got, "access_request.reviewed")
	}
}

func TestReviewAccessRequestRejectsNonOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-2"))
	_, err := svc.ReviewAccessRequest(ctx, &aiv1.ReviewAccessRequestRequest{
		AccessRequestId: "request-1",
		Decision:        aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_DENY,
		ReviewNote:      "no",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRevokeAccessRequestByOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 2, 16, 1, 9, 0, 0, time.UTC)
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Minute),
		ReviewerUserID:  "owner-1",
		ReviewNote:      "approved",
		ReviewedAt:      ptrTime(now.Add(-time.Minute)),
	}

	svc := NewService(store, store, &fakeSealer{})
	svc.clock = func() time.Time { return now }
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	resp, err := svc.RevokeAccessRequest(ctx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "removed",
	})
	if err != nil {
		t.Fatalf("revoke access request: %v", err)
	}
	if got := resp.GetAccessRequest().GetStatus(); got != aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_REVOKED {
		t.Fatalf("status = %v, want revoked", got)
	}
	if len(store.AuditEventNames) != 1 {
		t.Fatalf("audit events len = %d, want 1", len(store.AuditEventNames))
	}
	if got := store.AuditEventNames[0]; got != "access_request.revoked" {
		t.Fatalf("audit event = %q, want %q", got, "access_request.revoked")
	}
}

func TestRevokeAccessRequestRejectsNonOwner(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "approved",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-2"))
	_, err := svc.RevokeAccessRequest(ctx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "no",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestRevokeAccessRequestRejectsNonApproved(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.AccessRequests["request-1"] = storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "owner-1",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	svc := NewService(store, store, &fakeSealer{})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "owner-1"))
	_, err := svc.RevokeAccessRequest(ctx, &aiv1.RevokeAccessRequestRequest{
		AccessRequestId: "request-1",
		RevokeNote:      "not approved",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func TestFinishProviderConnectAcrossServiceInstances(t *testing.T) {
	store := newFakeStore()
	sealer := &fakeSealer{}

	svcStart := NewService(store, store, sealer)
	svcStart.clock = func() time.Time { return time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC) }
	idValuesStart := []string{"session-1", "state-1"}
	svcStart.idGenerator = func() (string, error) {
		if len(idValuesStart) == 0 {
			return "", errors.New("unexpected id call")
		}
		value := idValuesStart[0]
		idValuesStart = idValuesStart[1:]
		return value, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))
	startResp, err := svcStart.StartProviderConnect(ctx, &aiv1.StartProviderConnectRequest{
		Provider:        aiv1.Provider_PROVIDER_OPENAI,
		RequestedScopes: []string{"responses.read"},
	})
	if err != nil {
		t.Fatalf("start provider connect: %v", err)
	}

	svcFinish := NewService(store, store, sealer)
	svcFinish.clock = func() time.Time { return time.Date(2026, 2, 15, 23, 41, 0, 0, time.UTC) }
	svcFinish.idGenerator = func() (string, error) { return "grant-1", nil }

	finishResp, err := svcFinish.FinishProviderConnect(ctx, &aiv1.FinishProviderConnectRequest{
		ConnectSessionId:  startResp.GetConnectSessionId(),
		State:             startResp.GetState(),
		AuthorizationCode: "auth-code-1",
	})
	if err != nil {
		t.Fatalf("finish provider connect: %v", err)
	}
	if finishResp.GetProviderGrant().GetId() != "grant-1" {
		t.Fatalf("provider grant id = %q, want %q", finishResp.GetProviderGrant().GetId(), "grant-1")
	}
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected status %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != want {
		t.Fatalf("expected status %v, got %v", want, st.Code())
	}
}
