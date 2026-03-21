package ai

import (
	"context"
	"errors"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CredentialHandlers serves credential RPCs with explicit dependencies.
type CredentialHandlers struct {
	aiv1.UnimplementedCredentialServiceServer

	credentialStore storage.CredentialStore
	sealer          SecretSealer
	clock           func() time.Time
	idGenerator     func() (string, error)
	usageGuard      authReferenceUsageGuard
}

// CredentialHandlersConfig declares the dependencies for credential RPCs.
type CredentialHandlersConfig struct {
	CredentialStore      storage.CredentialStore
	AgentStore           storage.AgentStore
	GameCampaignAIClient gamev1.CampaignAIServiceClient
	Sealer               SecretSealer
	Clock                func() time.Time
	IDGenerator          func() (string, error)
}

// NewCredentialHandlers builds a credential RPC server from explicit deps.
func NewCredentialHandlers(cfg CredentialHandlersConfig) *CredentialHandlers {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := cfg.IDGenerator
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	return &CredentialHandlers{
		credentialStore: cfg.CredentialStore,
		sealer:          cfg.Sealer,
		clock:           clock,
		idGenerator:     idGenerator,
		usageGuard:      newAuthReferenceUsageGuard(cfg.AgentStore, cfg.GameCampaignAIClient),
	}
}

// CreateCredential creates one user-owned provider credential.
func (h *CredentialHandlers) CreateCredential(ctx context.Context, in *aiv1.CreateCredentialRequest) (*aiv1.CreateCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create credential request is required")
	}
	if h == nil || h.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}
	if h.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	created, err := credential.Create(credential.CreateInput{
		OwnerUserID: userID,
		Provider:    providerID,
		Label:       in.GetLabel(),
		Secret:      in.GetSecret(),
	}, h.clock, h.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Encrypt before persistence so storage only receives ciphertext.
	sealedSecret, err := h.sealer.Seal(created.Secret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "seal credential secret: %v", err)
	}

	record := storage.CredentialRecord{
		ID:               created.ID,
		OwnerUserID:      created.OwnerUserID,
		Provider:         string(created.Provider),
		Label:            created.Label,
		SecretCiphertext: sealedSecret,
		Status:           string(created.Status),
		CreatedAt:        created.CreatedAt,
		UpdatedAt:        created.UpdatedAt,
	}
	if err := h.credentialStore.PutCredential(ctx, record); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.AlreadyExists, "credential label already exists")
		}
		return nil, status.Errorf(codes.Internal, "put credential: %v", err)
	}

	return &aiv1.CreateCredentialResponse{Credential: credentialToProto(record)}, nil
}

// ListCredentials returns a page of credentials owned by the caller.
func (h *CredentialHandlers) ListCredentials(ctx context.Context, in *aiv1.ListCredentialsRequest) (*aiv1.ListCredentialsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list credentials request is required")
	}
	if h == nil || h.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := h.credentialStore.ListCredentialsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list credentials: %v", err)
	}

	resp := &aiv1.ListCredentialsResponse{
		NextPageToken: page.NextPageToken,
		Credentials:   make([]*aiv1.Credential, 0, len(page.Credentials)),
	}
	for _, rec := range page.Credentials {
		resp.Credentials = append(resp.Credentials, credentialToProto(rec))
	}
	return resp, nil
}

// RevokeCredential revokes one credential owned by the caller.
func (h *CredentialHandlers) RevokeCredential(ctx context.Context, in *aiv1.RevokeCredentialRequest) (*aiv1.RevokeCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke credential request is required")
	}
	if h == nil || h.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	credentialID := strings.TrimSpace(in.GetCredentialId())
	if credentialID == "" {
		return nil, status.Error(codes.InvalidArgument, "credential_id is required")
	}
	if err := h.usageGuard.ensureCredentialNotBoundToActiveCampaigns(ctx, userID, credentialID); err != nil {
		return nil, err
	}
	record, err := h.credentialStore.GetCredential(ctx, credentialID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		return nil, status.Errorf(codes.Internal, "get credential: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "credential not found")
	}
	revoked, err := credential.Revoke(credentialFromRecord(record), h.clock)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	applyCredentialLifecycle(&record, revoked)
	if err := h.credentialStore.PutCredential(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put credential: %v", err)
	}

	return &aiv1.RevokeCredentialResponse{Credential: credentialToProto(record)}, nil
}
