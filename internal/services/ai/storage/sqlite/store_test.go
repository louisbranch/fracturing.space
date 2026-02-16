package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestPutGetCredentialRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "Main",
		SecretCiphertext: "enc:abc",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.PutCredential(context.Background(), input); err != nil {
		t.Fatalf("put credential: %v", err)
	}

	got, err := store.GetCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.SecretCiphertext != input.SecretCiphertext {
		t.Fatalf("unexpected credential: %+v", got)
	}
}

func TestListCredentialsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	for _, rec := range []storage.CredentialRecord{
		{ID: "cred-1", OwnerUserID: "user-1", Provider: "openai", Label: "A", SecretCiphertext: "enc:1", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "cred-2", OwnerUserID: "user-1", Provider: "openai", Label: "B", SecretCiphertext: "enc:2", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "cred-3", OwnerUserID: "user-2", Provider: "openai", Label: "C", SecretCiphertext: "enc:3", Status: "active", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutCredential(context.Background(), rec); err != nil {
			t.Fatalf("put credential %s: %v", rec.ID, err)
		}
	}

	page, err := store.ListCredentialsByOwner(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list credentials: %v", err)
	}
	if len(page.Credentials) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(page.Credentials))
	}
}

func TestRevokeCredential(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	if err := store.PutCredential(context.Background(), storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "A",
		SecretCiphertext: "enc:1",
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("put credential: %v", err)
	}

	revokedAt := now.Add(time.Minute)
	if err := store.RevokeCredential(context.Background(), "user-1", "cred-1", revokedAt); err != nil {
		t.Fatalf("revoke credential: %v", err)
	}

	got, err := store.GetCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if got.Status != "revoked" {
		t.Fatalf("status = %q, want %q", got.Status, "revoked")
	}
	if got.RevokedAt == nil {
		t.Fatal("expected revoked_at")
	}
}

func TestPutGetProviderGrantRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 25, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	input := storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read", "responses.write"},
		TokenCiphertext:  "enc:grant-token",
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        &expiresAt,
	}
	if err := store.PutProviderGrant(context.Background(), input); err != nil {
		t.Fatalf("put provider grant: %v", err)
	}

	got, err := store.GetProviderGrant(context.Background(), "grant-1")
	if err != nil {
		t.Fatalf("get provider grant: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.TokenCiphertext != input.TokenCiphertext {
		t.Fatalf("unexpected provider grant: %+v", got)
	}
	if len(got.GrantedScopes) != 2 {
		t.Fatalf("granted_scopes len = %d, want 2", len(got.GrantedScopes))
	}
	if got.GrantedScopes[0] != "responses.read" || got.GrantedScopes[1] != "responses.write" {
		t.Fatalf("granted_scopes = %v, want [responses.read responses.write]", got.GrantedScopes)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, expiresAt)
	}
}

func TestListProviderGrantsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 25, 0, 0, time.UTC)

	for _, rec := range []storage.ProviderGrantRecord{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-2", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:3", Status: "active", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), rec); err != nil {
			t.Fatalf("put provider grant %s: %v", rec.ID, err)
		}
	}

	page, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", storage.ProviderGrantFilter{})
	if err != nil {
		t.Fatalf("list provider grants: %v", err)
	}
	if len(page.ProviderGrants) != 2 {
		t.Fatalf("expected 2 provider grants, got %d", len(page.ProviderGrants))
	}
}

func TestListProviderGrantsByOwnerWithFilters(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 2, 1, 0, 0, time.UTC)
	for _, rec := range []storage.ProviderGrantRecord{
		{ID: "grant-1", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:1", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-2", OwnerUserID: "user-1", Provider: "other", GrantedScopes: []string{"responses.write"}, TokenCiphertext: "enc:2", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-3", OwnerUserID: "user-1", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:3", Status: "revoked", CreatedAt: now, UpdatedAt: now},
		{ID: "grant-4", OwnerUserID: "user-2", Provider: "openai", GrantedScopes: []string{"responses.read"}, TokenCiphertext: "enc:4", Status: "revoked", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutProviderGrant(context.Background(), rec); err != nil {
			t.Fatalf("put provider grant %s: %v", rec.ID, err)
		}
	}

	providerOnly, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", storage.ProviderGrantFilter{
		Provider: "openai",
	})
	if err != nil {
		t.Fatalf("list provider-filtered grants: %v", err)
	}
	if len(providerOnly.ProviderGrants) != 2 {
		t.Fatalf("provider-filtered len = %d, want 2", len(providerOnly.ProviderGrants))
	}

	statusOnly, err := store.ListProviderGrantsByOwner(context.Background(), "user-1", 10, "", storage.ProviderGrantFilter{
		Status: "revoked",
	})
	if err != nil {
		t.Fatalf("list status-filtered grants: %v", err)
	}
	if len(statusOnly.ProviderGrants) != 1 {
		t.Fatalf("status-filtered len = %d, want 1", len(statusOnly.ProviderGrants))
	}
	if got := statusOnly.ProviderGrants[0].ID; got != "grant-3" {
		t.Fatalf("status-filtered id = %q, want %q", got, "grant-3")
	}
}

func TestRevokeProviderGrant(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 25, 0, 0, time.UTC)

	if err := store.PutProviderGrant(context.Background(), storage.ProviderGrantRecord{
		ID:              "grant-1",
		OwnerUserID:     "user-1",
		Provider:        "openai",
		GrantedScopes:   []string{"responses.read"},
		TokenCiphertext: "enc:1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put provider grant: %v", err)
	}

	revokedAt := now.Add(time.Minute)
	if err := store.RevokeProviderGrant(context.Background(), "user-1", "grant-1", revokedAt); err != nil {
		t.Fatalf("revoke provider grant: %v", err)
	}

	got, err := store.GetProviderGrant(context.Background(), "grant-1")
	if err != nil {
		t.Fatalf("get provider grant: %v", err)
	}
	if got.Status != "revoked" {
		t.Fatalf("status = %q, want %q", got.Status, "revoked")
	}
	if got.RevokedAt == nil {
		t.Fatal("expected revoked_at")
	}
}

func TestUpdateProviderGrantToken(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 50, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	if err := store.PutProviderGrant(context.Background(), storage.ProviderGrantRecord{
		ID:               "grant-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		GrantedScopes:    []string{"responses.read"},
		TokenCiphertext:  "enc:old",
		RefreshSupported: true,
		Status:           "active",
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        &expiresAt,
	}); err != nil {
		t.Fatalf("put provider grant: %v", err)
	}

	refreshedAt := now.Add(10 * time.Minute)
	newExpiresAt := refreshedAt.Add(2 * time.Hour)
	if err := store.UpdateProviderGrantToken(context.Background(), "user-1", "grant-1", "enc:new", refreshedAt, &newExpiresAt, "active", ""); err != nil {
		t.Fatalf("update provider grant token: %v", err)
	}

	got, err := store.GetProviderGrant(context.Background(), "grant-1")
	if err != nil {
		t.Fatalf("get provider grant: %v", err)
	}
	if got.TokenCiphertext != "enc:new" {
		t.Fatalf("token_ciphertext = %q, want %q", got.TokenCiphertext, "enc:new")
	}
	if got.LastRefreshedAt == nil || !got.LastRefreshedAt.Equal(refreshedAt) {
		t.Fatalf("last_refreshed_at = %v, want %v", got.LastRefreshedAt, refreshedAt)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(newExpiresAt) {
		t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, newExpiresAt)
	}
}

func TestPutGetProviderConnectSessionRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	input := storage.ProviderConnectSessionRecord{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               "openai",
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 "pending",
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}
	if err := store.PutProviderConnectSession(context.Background(), input); err != nil {
		t.Fatalf("put provider connect session: %v", err)
	}

	got, err := store.GetProviderConnectSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.StateHash != input.StateHash {
		t.Fatalf("unexpected provider connect session: %+v", got)
	}
}

func TestCompleteProviderConnectSession(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 23, 40, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	if err := store.PutProviderConnectSession(context.Background(), storage.ProviderConnectSessionRecord{
		ID:                     "session-1",
		OwnerUserID:            "user-1",
		Provider:               "openai",
		RequestedScopes:        []string{"responses.read"},
		StateHash:              "hash:state",
		CodeVerifierCiphertext: "enc:verifier",
		Status:                 "pending",
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}); err != nil {
		t.Fatalf("put provider connect session: %v", err)
	}

	completedAt := now.Add(time.Minute)
	if err := store.CompleteProviderConnectSession(context.Background(), "user-1", "session-1", completedAt); err != nil {
		t.Fatalf("complete provider connect session: %v", err)
	}

	got, err := store.GetProviderConnectSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("get provider connect session: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("status = %q, want %q", got.Status, "completed")
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Fatalf("completed_at = %v, want %v", got.CompletedAt, completedAt)
	}
}

func TestPutGetAgentRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := storage.AgentRecord{
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
	if err := store.PutAgent(context.Background(), input); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	got, err := store.GetAgent(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.ID != input.ID || got.OwnerUserID != input.OwnerUserID || got.CredentialID != input.CredentialID {
		t.Fatalf("unexpected agent: %+v", got)
	}
}

func TestPutGetAgentRoundTripWithProviderGrant(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	input := storage.AgentRecord{
		ID:              "agent-grant-1",
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		CredentialID:    "",
		ProviderGrantID: "grant-1",
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.PutAgent(context.Background(), input); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	got, err := store.GetAgent(context.Background(), "agent-grant-1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.ProviderGrantID != "grant-1" {
		t.Fatalf("provider_grant_id = %q, want %q", got.ProviderGrantID, "grant-1")
	}
	if got.CredentialID != "" {
		t.Fatalf("credential_id = %q, want empty", got.CredentialID)
	}
}

func TestPutAgentRequiresExactlyOneAuthReference(t *testing.T) {
	store := openTempStore(t)
	now := time.Now().UTC()

	base := storage.AgentRecord{
		ID:          "agent-1",
		OwnerUserID: "user-1",
		Name:        "Narrator",
		Provider:    "openai",
		Model:       "gpt-4o-mini",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	missing := base
	if err := store.PutAgent(context.Background(), missing); err == nil {
		t.Fatal("expected error for missing auth reference")
	}

	both := base
	both.CredentialID = "cred-1"
	both.ProviderGrantID = "grant-1"
	if err := store.PutAgent(context.Background(), both); err == nil {
		t.Fatal("expected error for multiple auth references")
	}
}

func TestDeleteAgent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 15, 22, 45, 0, 0, time.UTC)

	if err := store.PutAgent(context.Background(), storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Name:         "Narrator",
		Provider:     "openai",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("put agent: %v", err)
	}

	if err := store.DeleteAgent(context.Background(), "user-1", "agent-1"); err != nil {
		t.Fatalf("delete agent: %v", err)
	}

	_, err := store.GetAgent(context.Background(), "agent-1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPutGetAccessRequestRoundTrip(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 55, 0, 0, time.UTC)

	input := storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           "invoke",
		RequestNote:     "please allow",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.PutAccessRequest(context.Background(), input); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	got, err := store.GetAccessRequest(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("get access request: %v", err)
	}
	if got.ID != input.ID || got.RequesterUserID != input.RequesterUserID || got.OwnerUserID != input.OwnerUserID {
		t.Fatalf("unexpected access request: %+v", got)
	}
}

func TestListAccessRequestsByRequesterAndOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 56, 0, 0, time.UTC)

	records := []storage.AccessRequestRecord{
		{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "user-2", AgentID: "agent-1", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "user-3", AgentID: "agent-2", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{ID: "request-3", RequesterUserID: "user-4", OwnerUserID: "user-2", AgentID: "agent-3", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now},
	}
	for _, record := range records {
		if err := store.PutAccessRequest(context.Background(), record); err != nil {
			t.Fatalf("put access request %s: %v", record.ID, err)
		}
	}

	requesterPage, err := store.ListAccessRequestsByRequester(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list by requester: %v", err)
	}
	if len(requesterPage.AccessRequests) != 2 {
		t.Fatalf("requester page len = %d, want 2", len(requesterPage.AccessRequests))
	}

	ownerPage, err := store.ListAccessRequestsByOwner(context.Background(), "user-2", 10, "")
	if err != nil {
		t.Fatalf("list by owner: %v", err)
	}
	if len(ownerPage.AccessRequests) != 2 {
		t.Fatalf("owner page len = %d, want 2", len(ownerPage.AccessRequests))
	}
}

func TestGetApprovedInvokeAccessByRequesterForAgent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 5, 0, 0, time.UTC)

	records := []storage.AccessRequestRecord{
		{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now},
		{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{ID: "request-3", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-2", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now},
		{ID: "request-4", RequesterUserID: "user-1", OwnerUserID: "owner-2", AgentID: "agent-1", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now},
		{ID: "request-5", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: "observe", Status: "approved", CreatedAt: now, UpdatedAt: now},
	}
	for _, record := range records {
		if err := store.PutAccessRequest(context.Background(), record); err != nil {
			t.Fatalf("put access request %s: %v", record.ID, err)
		}
	}

	got, err := store.GetApprovedInvokeAccessByRequesterForAgent(context.Background(), "user-1", "owner-1", "agent-1")
	if err != nil {
		t.Fatalf("get approved invoke access: %v", err)
	}
	if got.ID != "request-1" {
		t.Fatalf("id = %q, want %q", got.ID, "request-1")
	}

	_, err = store.GetApprovedInvokeAccessByRequesterForAgent(context.Background(), "user-1", "owner-1", "agent-missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing access error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestListApprovedInvokeAccessRequestsByRequester(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 1, 6, 0, 0, time.UTC)

	records := []storage.AccessRequestRecord{
		{ID: "request-1", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-1", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now},
		{ID: "request-2", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-2", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now},
		{ID: "request-3", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-3", Scope: "invoke", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{ID: "request-4", RequesterUserID: "user-1", OwnerUserID: "owner-1", AgentID: "agent-4", Scope: "observe", Status: "approved", CreatedAt: now, UpdatedAt: now},
		{ID: "request-5", RequesterUserID: "user-2", OwnerUserID: "owner-1", AgentID: "agent-5", Scope: "invoke", Status: "approved", CreatedAt: now, UpdatedAt: now},
	}
	for _, record := range records {
		if err := store.PutAccessRequest(context.Background(), record); err != nil {
			t.Fatalf("put access request %s: %v", record.ID, err)
		}
	}

	first, err := store.ListApprovedInvokeAccessRequestsByRequester(context.Background(), "user-1", 1, "")
	if err != nil {
		t.Fatalf("list first approved invoke page: %v", err)
	}
	if len(first.AccessRequests) != 1 {
		t.Fatalf("first page len = %d, want 1", len(first.AccessRequests))
	}
	if got := first.AccessRequests[0].ID; got != "request-1" {
		t.Fatalf("first page id = %q, want %q", got, "request-1")
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListApprovedInvokeAccessRequestsByRequester(context.Background(), "user-1", 1, first.NextPageToken)
	if err != nil {
		t.Fatalf("list second approved invoke page: %v", err)
	}
	if len(second.AccessRequests) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.AccessRequests))
	}
	if got := second.AccessRequests[0].ID; got != "request-2" {
		t.Fatalf("second page id = %q, want %q", got, "request-2")
	}
}

func TestReviewAccessRequest(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 57, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           "invoke",
		RequestNote:     "please allow",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	reviewedAt := now.Add(time.Minute)
	if err := store.ReviewAccessRequest(context.Background(), "user-2", "request-1", "approved", "user-2", "approved", reviewedAt); err != nil {
		t.Fatalf("review access request: %v", err)
	}

	got, err := store.GetAccessRequest(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("get access request: %v", err)
	}
	if got.Status != "approved" {
		t.Fatalf("status = %q, want %q", got.Status, "approved")
	}
	if got.ReviewerUserID != "user-2" {
		t.Fatalf("reviewer_user_id = %q, want %q", got.ReviewerUserID, "user-2")
	}
	if got.ReviewedAt == nil || !got.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf("reviewed_at = %v, want %v", got.ReviewedAt, reviewedAt)
	}
}

func TestReviewAccessRequestRejectsNonPending(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 58, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "denied",
		CreatedAt:       now,
		UpdatedAt:       now,
		ReviewerUserID:  "user-2",
		ReviewNote:      "already denied",
		ReviewedAt:      &now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	err := store.ReviewAccessRequest(context.Background(), "user-2", "request-1", "approved", "user-2", "retry", now.Add(time.Minute))
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("review error = %v, want %v", err, storage.ErrConflict)
	}
}

func TestReviewAccessRequestRejectsReviewerMismatch(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 0, 58, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), storage.AccessRequestRecord{
		ID:              "request-1",
		RequesterUserID: "user-1",
		OwnerUserID:     "user-2",
		AgentID:         "agent-1",
		Scope:           "invoke",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	err := store.ReviewAccessRequest(context.Background(), "user-2", "request-1", "approved", "user-3", "retry", now.Add(time.Minute))
	if err == nil {
		t.Fatal("expected review error for reviewer mismatch")
	}
	if !strings.Contains(err.Error(), "reviewer user id must match owner user id") {
		t.Fatalf("review error = %v, want reviewer mismatch", err)
	}
}

func TestRevokeAccessRequestTransition(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), storage.AccessRequestRecord{
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
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	revokedAt := now
	if err := store.RevokeAccessRequest(context.Background(), "owner-1", "request-1", "revoked", "owner-1", "removed", revokedAt); err != nil {
		t.Fatalf("revoke access request: %v", err)
	}

	got, err := store.GetAccessRequest(context.Background(), "request-1")
	if err != nil {
		t.Fatalf("get access request: %v", err)
	}
	if got.Status != "revoked" {
		t.Fatalf("status = %q, want %q", got.Status, "revoked")
	}
	if got.ReviewerUserID != "owner-1" {
		t.Fatalf("reviewer_user_id = %q, want %q", got.ReviewerUserID, "owner-1")
	}
	if got.ReviewNote != "removed" {
		t.Fatalf("review_note = %q, want %q", got.ReviewNote, "removed")
	}
	if !got.UpdatedAt.Equal(revokedAt) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, revokedAt)
	}

	if err := store.RevokeAccessRequest(context.Background(), "owner-1", "request-1", "revoked", "owner-1", "again", revokedAt.Add(time.Minute)); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("second revoke error = %v, want %v", err, storage.ErrConflict)
	}
}

func TestRevokeAccessRequestRejectsReviewerMismatch(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC)

	if err := store.PutAccessRequest(context.Background(), storage.AccessRequestRecord{
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
	}); err != nil {
		t.Fatalf("put access request: %v", err)
	}

	err := store.RevokeAccessRequest(context.Background(), "owner-1", "request-1", "revoked", "owner-2", "removed", now)
	if err == nil {
		t.Fatal("expected revoke error for reviewer mismatch")
	}
	if !strings.Contains(err.Error(), "reviewer user id must match owner user id") {
		t.Fatalf("revoke error = %v, want reviewer mismatch", err)
	}
}

func TestPutAuditEvent(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 2, 30, 0, 0, time.UTC)

	err := store.PutAuditEvent(context.Background(), storage.AuditEventRecord{
		EventName:       "access_request.created",
		ActorUserID:     "",
		OwnerUserID:     "owner-1",
		RequesterUserID: "requester-1",
		AgentID:         "agent-1",
		AccessRequestID: "request-1",
		Outcome:         "pending",
		CreatedAt:       now,
	})
	if err == nil {
		t.Fatal("expected validation error for empty actor_user_id")
	}

	if err := store.PutAuditEvent(context.Background(), storage.AuditEventRecord{
		EventName:       "access_request.created",
		ActorUserID:     "requester-1",
		OwnerUserID:     "owner-1",
		RequesterUserID: "requester-1",
		AgentID:         "agent-1",
		AccessRequestID: "request-1",
		Outcome:         "pending",
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("put audit event: %v", err)
	}

	var (
		eventName       string
		actorUserID     string
		ownerUserID     string
		requesterUserID string
		agentID         string
		accessRequestID string
		outcome         string
		createdAt       int64
	)
	row := store.DB().QueryRowContext(context.Background(), `
SELECT event_name, actor_user_id, owner_user_id, requester_user_id, agent_id, access_request_id, outcome, created_at
FROM ai_audit_events
WHERE actor_user_id = ?
ORDER BY id DESC
LIMIT 1
`, "requester-1")
	if err := row.Scan(&eventName, &actorUserID, &ownerUserID, &requesterUserID, &agentID, &accessRequestID, &outcome, &createdAt); err != nil {
		t.Fatalf("scan audit row: %v", err)
	}
	if eventName != "access_request.created" {
		t.Fatalf("event_name = %q, want %q", eventName, "access_request.created")
	}
	if actorUserID != "requester-1" {
		t.Fatalf("actor_user_id = %q, want %q", actorUserID, "requester-1")
	}
	if ownerUserID != "owner-1" {
		t.Fatalf("owner_user_id = %q, want %q", ownerUserID, "owner-1")
	}
	if requesterUserID != "requester-1" {
		t.Fatalf("requester_user_id = %q, want %q", requesterUserID, "requester-1")
	}
	if agentID != "agent-1" {
		t.Fatalf("agent_id = %q, want %q", agentID, "agent-1")
	}
	if accessRequestID != "request-1" {
		t.Fatalf("access_request_id = %q, want %q", accessRequestID, "request-1")
	}
	if outcome != "pending" {
		t.Fatalf("outcome = %q, want %q", outcome, "pending")
	}
	if createdAt != now.UnixMilli() {
		t.Fatalf("created_at = %d, want %d", createdAt, now.UnixMilli())
	}
}

func TestListAuditEventsByOwner(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 3, 15, 0, 0, time.UTC)

	records := []storage.AuditEventRecord{
		{EventName: "access_request.created", ActorUserID: "user-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "pending", CreatedAt: now},
		{EventName: "access_request.reviewed", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "approved", CreatedAt: now.Add(time.Minute)},
		{EventName: "access_request.created", ActorUserID: "user-2", OwnerUserID: "owner-2", RequesterUserID: "user-2", AgentID: "agent-2", AccessRequestID: "request-2", Outcome: "pending", CreatedAt: now},
		{EventName: "access_request.revoked", ActorUserID: "owner-1", OwnerUserID: "owner-1", RequesterUserID: "user-1", AgentID: "agent-1", AccessRequestID: "request-1", Outcome: "revoked", CreatedAt: now.Add(2 * time.Minute)},
	}
	for _, record := range records {
		if err := store.PutAuditEvent(context.Background(), record); err != nil {
			t.Fatalf("put audit event: %v", err)
		}
	}

	first, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 2, "", storage.AuditEventFilter{})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.AuditEvents) != 2 {
		t.Fatalf("first page len = %d, want 2", len(first.AuditEvents))
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}
	if first.AuditEvents[0].OwnerUserID != "owner-1" || first.AuditEvents[1].OwnerUserID != "owner-1" {
		t.Fatalf("unexpected owner ids: %+v", first.AuditEvents)
	}

	second, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 2, first.NextPageToken, storage.AuditEventFilter{})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.AuditEvents) != 1 {
		t.Fatalf("second page len = %d, want 1", len(second.AuditEvents))
	}
	if second.AuditEvents[0].Outcome != "revoked" {
		t.Fatalf("second page outcome = %q, want %q", second.AuditEvents[0].Outcome, "revoked")
	}
	if second.NextPageToken != "" {
		t.Fatalf("second next page token = %q, want empty", second.NextPageToken)
	}
}

func TestListAuditEventsByOwnerWithFilters(t *testing.T) {
	store := openTempStore(t)
	now := time.Date(2026, 2, 16, 4, 0, 0, 0, time.UTC)
	records := []storage.AuditEventRecord{
		{
			EventName:       "access_request.created",
			ActorUserID:     "requester-1",
			OwnerUserID:     "owner-1",
			RequesterUserID: "requester-1",
			AgentID:         "agent-1",
			AccessRequestID: "request-1",
			Outcome:         "pending",
			CreatedAt:       now,
		},
		{
			EventName:       "access_request.reviewed",
			ActorUserID:     "owner-1",
			OwnerUserID:     "owner-1",
			RequesterUserID: "requester-1",
			AgentID:         "agent-1",
			AccessRequestID: "request-1",
			Outcome:         "approved",
			CreatedAt:       now.Add(2 * time.Minute),
		},
		{
			EventName:       "access_request.reviewed",
			ActorUserID:     "owner-1",
			OwnerUserID:     "owner-1",
			RequesterUserID: "requester-2",
			AgentID:         "agent-2",
			AccessRequestID: "request-2",
			Outcome:         "approved",
			CreatedAt:       now.Add(4 * time.Minute),
		},
		{
			EventName:       "access_request.reviewed",
			ActorUserID:     "owner-2",
			OwnerUserID:     "owner-2",
			RequesterUserID: "requester-3",
			AgentID:         "agent-3",
			AccessRequestID: "request-3",
			Outcome:         "approved",
			CreatedAt:       now.Add(5 * time.Minute),
		},
	}
	for _, record := range records {
		if err := store.PutAuditEvent(context.Background(), record); err != nil {
			t.Fatalf("put audit event: %v", err)
		}
	}

	eventNameOnly, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 10, "", storage.AuditEventFilter{
		EventName: "access_request.reviewed",
	})
	if err != nil {
		t.Fatalf("list by event name: %v", err)
	}
	if len(eventNameOnly.AuditEvents) != 2 {
		t.Fatalf("event name len = %d, want 2", len(eventNameOnly.AuditEvents))
	}
	for _, event := range eventNameOnly.AuditEvents {
		if event.EventName != "access_request.reviewed" {
			t.Fatalf("event_name = %q, want %q", event.EventName, "access_request.reviewed")
		}
	}

	agentOnly, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 10, "", storage.AuditEventFilter{
		AgentID: "agent-2",
	})
	if err != nil {
		t.Fatalf("list by agent id: %v", err)
	}
	if len(agentOnly.AuditEvents) != 1 {
		t.Fatalf("agent filter len = %d, want 1", len(agentOnly.AuditEvents))
	}
	if got := agentOnly.AuditEvents[0].AgentID; got != "agent-2" {
		t.Fatalf("agent id = %q, want %q", got, "agent-2")
	}

	createdAfter := now.Add(time.Minute)
	createdBefore := now.Add(3 * time.Minute)
	timeWindow, err := store.ListAuditEventsByOwner(context.Background(), "owner-1", 10, "", storage.AuditEventFilter{
		CreatedAfter:  &createdAfter,
		CreatedBefore: &createdBefore,
	})
	if err != nil {
		t.Fatalf("list by time window: %v", err)
	}
	if len(timeWindow.AuditEvents) != 1 {
		t.Fatalf("time window len = %d, want 1", len(timeWindow.AuditEvents))
	}
	if got := timeWindow.AuditEvents[0].AccessRequestID; got != "request-1" {
		t.Fatalf("time window access_request_id = %q, want %q", got, "request-1")
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ai.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}
