package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
	"github.com/louisbranch/fracturing.space/internal/services/ai/secret"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// CredentialService handles credential lifecycle operations.
type CredentialService struct {
	credentialStore  storage.CredentialStore
	providerRegistry *providercatalog.Registry
	sealer           secret.Sealer
	usagePolicy      *UsagePolicy
	clock            Clock
	idGenerator      IDGenerator
}

// CredentialServiceConfig declares dependencies for the credential service.
type CredentialServiceConfig struct {
	CredentialStore  storage.CredentialStore
	ProviderRegistry *providercatalog.Registry
	Sealer           secret.Sealer
	UsagePolicy      *UsagePolicy
	Clock            Clock
	IDGenerator      IDGenerator
}

// NewCredentialService builds a credential service from explicit deps.
func NewCredentialService(cfg CredentialServiceConfig) (*CredentialService, error) {
	if cfg.CredentialStore == nil {
		return nil, fmt.Errorf("ai: NewCredentialService: credential store is required")
	}
	if cfg.Sealer == nil {
		return nil, fmt.Errorf("ai: NewCredentialService: sealer is required")
	}
	if err := RequireProviderRegistry(cfg.ProviderRegistry, "NewCredentialService"); err != nil {
		return nil, err
	}
	return &CredentialService{
		credentialStore:  cfg.CredentialStore,
		providerRegistry: cfg.ProviderRegistry,
		sealer:           cfg.Sealer,
		usagePolicy:      cfg.UsagePolicy,
		clock:            withDefaultClock(cfg.Clock),
		idGenerator:      withDefaultIDGenerator(cfg.IDGenerator),
	}, nil
}

// CreateCredentialInput is the domain input for creating a credential.
type CreateCredentialInput struct {
	OwnerUserID string
	Provider    provider.Provider
	Label       string
	Secret      string
}

// Create creates a user-owned provider credential, encrypts the secret, and
// persists the record.
func (s *CredentialService) Create(ctx context.Context, input CreateCredentialInput) (credential.Credential, error) {
	created, err := credential.Create(credential.CreateInput{
		OwnerUserID: input.OwnerUserID,
		Provider:    input.Provider,
		Label:       input.Label,
		Secret:      input.Secret,
	}, s.clock, s.idGenerator)
	if err != nil {
		return credential.Credential{}, Errorf(ErrKindInvalidArgument, "%s", err)
	}
	if !s.providerRegistry.HasProvider(created.Provider) {
		return credential.Credential{}, Errorf(ErrKindFailedPrecondition, "provider is unavailable")
	}

	sealedSecret, err := s.sealer.Seal(created.Secret)
	if err != nil {
		return credential.Credential{}, Wrapf(ErrKindInternal, err, "seal credential secret")
	}
	created.SecretCiphertext = sealedSecret

	if err := s.credentialStore.PutCredential(ctx, created); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return credential.Credential{}, Errorf(ErrKindAlreadyExists, "credential label already exists")
		}
		return credential.Credential{}, Wrapf(ErrKindInternal, err, "put credential")
	}

	return created, nil
}

// List returns a page of credentials owned by the given user.
func (s *CredentialService) List(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (credential.Page, error) {
	page, err := s.credentialStore.ListCredentialsByOwner(ctx, ownerUserID, pageSize, pageToken)
	if err != nil {
		return credential.Page{}, Wrapf(ErrKindInternal, err, "list credentials")
	}
	return page, nil
}

// Revoke revokes a credential owned by the given user.
func (s *CredentialService) Revoke(ctx context.Context, ownerUserID, credentialID string) (credential.Credential, error) {
	if credentialID == "" {
		return credential.Credential{}, Errorf(ErrKindInvalidArgument, "credential_id is required")
	}
	if s.usagePolicy != nil {
		if err := s.usagePolicy.EnsureCredentialNotBoundToActiveCampaigns(ctx, ownerUserID, credentialID); err != nil {
			return credential.Credential{}, err
		}
	}

	existing, err := s.credentialStore.GetCredential(ctx, credentialID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return credential.Credential{}, Errorf(ErrKindNotFound, "credential not found")
		}
		return credential.Credential{}, Wrapf(ErrKindInternal, err, "get credential")
	}
	if existing.OwnerUserID != ownerUserID {
		return credential.Credential{}, Errorf(ErrKindNotFound, "credential not found")
	}

	revoked, err := credential.Revoke(existing, s.clock)
	if err != nil {
		return credential.Credential{}, Errorf(ErrKindFailedPrecondition, "%s", err)
	}
	if err := s.credentialStore.PutCredential(ctx, revoked); err != nil {
		return credential.Credential{}, Wrapf(ErrKindInternal, err, "put credential")
	}

	return revoked, nil
}
