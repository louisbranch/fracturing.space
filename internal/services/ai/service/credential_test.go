package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type fakeCampaignUsageReader struct {
	usageByAgent map[string]int32
	err          error
}

func (f *fakeCampaignUsageReader) ActiveCampaignCount(_ context.Context, agentID string) (int32, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.usageByAgent[agentID], nil
}

func TestCredentialServiceCreateEncryptsAndPersists(t *testing.T) {
	store := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 15, 0, 0, 0, time.UTC)

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{},
		Clock:            func() time.Time { return now },
		IDGenerator:      func() (string, error) { return "cred-1", nil },
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	record, err := svc.Create(context.Background(), CreateCredentialInput{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "Main",
		Secret:      "sk-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.ID != "cred-1" {
		t.Fatalf("record.ID = %q, want %q", record.ID, "cred-1")
	}
	if record.SecretCiphertext != "enc:sk-1" {
		t.Fatalf("record.SecretCiphertext = %q, want %q", record.SecretCiphertext, "enc:sk-1")
	}

	stored := store.Credentials["cred-1"]
	if stored.SecretCiphertext != "enc:sk-1" {
		t.Fatalf("stored.SecretCiphertext = %q, want %q", stored.SecretCiphertext, "enc:sk-1")
	}
	if stored.Secret != "sk-1" {
		t.Fatalf("stored.Secret = %q, want %q", stored.Secret, "sk-1")
	}
}

func TestCredentialServiceCreateMapsConflict(t *testing.T) {
	store := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 15, 5, 0, 0, time.UTC)
	store.Credentials["cred-existing"] = credential.Credential{
		ID:               "cred-existing",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		Status:           credential.StatusActive,
		SecretCiphertext: "enc:sk-old",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{},
		Clock:            func() time.Time { return now },
		IDGenerator:      func() (string, error) { return "cred-1", nil },
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateCredentialInput{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       " main ",
		Secret:      "sk-1",
	})
	if got := ErrorKindOf(err); got != ErrKindAlreadyExists {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindAlreadyExists)
	}
}

func TestCredentialServiceCreateRejectsUnavailableProvider(t *testing.T) {
	store := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 15, 6, 0, 0, time.UTC)

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{},
		Clock:            func() time.Time { return now },
		IDGenerator:      func() (string, error) { return "cred-1", nil },
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateCredentialInput{
		OwnerUserID: "user-1",
		Provider:    provider.Anthropic,
		Label:       "Main",
		Secret:      "sk-1",
	})
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
	if len(store.Credentials) != 0 {
		t.Fatalf("len(store.Credentials) = %d, want 0", len(store.Credentials))
	}
}

func TestCredentialServiceCreateAcceptsAnthropicProviderWhenRegistered(t *testing.T) {
	store := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 15, 7, 0, 0, time.UTC)

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore: store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{
			provider.OpenAI:    nil,
			provider.Anthropic: nil,
		}, nil),
		Sealer:      &aifakes.Sealer{},
		Clock:       func() time.Time { return now },
		IDGenerator: func() (string, error) { return "cred-anthropic", nil },
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	record, err := svc.Create(context.Background(), CreateCredentialInput{
		OwnerUserID: "user-1",
		Provider:    provider.Anthropic,
		Label:       "Claude",
		Secret:      "sk-ant-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if record.Provider != provider.Anthropic {
		t.Fatalf("record.Provider = %q, want %q", record.Provider, provider.Anthropic)
	}
	if store.Credentials["cred-anthropic"].Provider != provider.Anthropic {
		t.Fatalf("stored.Provider = %q, want %q", store.Credentials["cred-anthropic"].Provider, provider.Anthropic)
	}
}

func TestCredentialServiceListByOwner(t *testing.T) {
	store := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 15, 8, 0, 0, time.UTC)
	store.Credentials["cred-1"] = credential.Credential{
		ID:          "cred-1",
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "Main",
		Status:      credential.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	store.Credentials["cred-2"] = credential.Credential{
		ID:          "cred-2",
		OwnerUserID: "user-2",
		Provider:    provider.Anthropic,
		Label:       "Claude",
		Status:      credential.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{},
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	page, err := svc.List(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Credentials) != 1 || page.Credentials[0].ID != "cred-1" {
		t.Fatalf("page.Credentials = %+v, want cred-1 only", page.Credentials)
	}
}

func TestCredentialServiceRevokeRevokesOwnedCredential(t *testing.T) {
	store := aifakes.NewCredentialStore()
	now := time.Date(2026, 3, 23, 15, 10, 0, 0, time.UTC)
	store.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		Status:           credential.StatusActive,
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{},
		Clock:            func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	record, err := svc.Revoke(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if record.Status != credential.StatusRevoked {
		t.Fatalf("record.Status = %q, want %q", record.Status, credential.StatusRevoked)
	}
	if record.RevokedAt == nil || !record.RevokedAt.Equal(now) {
		t.Fatalf("record.RevokedAt = %v, want %v", record.RevokedAt, now)
	}
	if got := store.Credentials["cred-1"].Status; got != credential.StatusRevoked {
		t.Fatalf("stored.Status = %q, want %q", got, credential.StatusRevoked)
	}
}

func TestCredentialServiceRevokeRejectsActiveCampaignBinding(t *testing.T) {
	credentialStore := aifakes.NewCredentialStore()
	agentStore := aifakes.NewAgentStore()
	now := time.Date(2026, 3, 23, 15, 15, 0, 0, time.UTC)
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "Main",
		Status:           credential.StatusActive,
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now.Add(-time.Hour),
		UpdatedAt:        now.Add(-time.Hour),
	}
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}

	usageReader := NewAgentBindingUsageReader(&fakeCampaignUsageReader{
		usageByAgent: map[string]int32{"agent-1": 1},
	})
	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  credentialStore,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{},
		UsagePolicy: NewUsagePolicy(UsagePolicyConfig{
			AgentBindingUsageReader:  usageReader,
			AuthReferenceUsageReader: NewAuthReferenceUsageReader(agentStore, usageReader),
		}),
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	_, err = svc.Revoke(context.Background(), "user-1", "cred-1")
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
	if got := credentialStore.Credentials["cred-1"].Status; got != credential.StatusActive {
		t.Fatalf("stored.Status = %q, want %q", got, credential.StatusActive)
	}
}

func TestCredentialServiceCreateMapsSealFailure(t *testing.T) {
	store := aifakes.NewCredentialStore()

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, map[provider.Provider]provider.ModelAdapter{provider.OpenAI: nil}, nil),
		Sealer:           &aifakes.Sealer{SealErr: errors.New("seal fail")},
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateCredentialInput{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Label:       "Main",
		Secret:      "sk-1",
	})
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}

func TestCredentialServiceListReturnsInternalOnStoreError(t *testing.T) {
	t.Parallel()
	store := aifakes.NewCredentialStore()
	store.ListErr = errors.New("db read fail")

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, nil, nil),
		Sealer:           &aifakes.Sealer{},
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	_, err = svc.List(context.Background(), "user-1", 10, "")
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}

func TestCredentialServiceRevokeReturnsInternalOnGetStoreError(t *testing.T) {
	t.Parallel()
	store := aifakes.NewCredentialStore()
	store.GetErr = errors.New("db read fail")

	svc, err := NewCredentialService(CredentialServiceConfig{
		CredentialStore:  store,
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, nil, nil),
		Sealer:           &aifakes.Sealer{},
	})
	if err != nil {
		t.Fatalf("NewCredentialService: %v", err)
	}

	_, err = svc.Revoke(context.Background(), "user-1", "cred-1")
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}
