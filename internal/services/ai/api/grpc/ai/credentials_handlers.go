package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateCredential creates one user-owned provider credential.
func (s *Service) CreateCredential(ctx context.Context, in *aiv1.CreateCredentialRequest) (*aiv1.CreateCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create credential request is required")
	}
	if s.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	provider, err := credentialProviderFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	created, err := credential.Create(credential.CreateInput{
		OwnerUserID: userID,
		Provider:    provider,
		Label:       in.GetLabel(),
		Secret:      in.GetSecret(),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Encrypt before persistence so storage only receives ciphertext. The API
	// never returns plaintext secrets after this boundary.
	sealedSecret, err := s.sealer.Seal(created.Secret)
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
	if err := s.credentialStore.PutCredential(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put credential: %v", err)
	}

	return &aiv1.CreateCredentialResponse{
		Credential: credentialToProto(record),
	}, nil
}

// ListCredentials returns a page of credentials owned by the caller.
func (s *Service) ListCredentials(ctx context.Context, in *aiv1.ListCredentialsRequest) (*aiv1.ListCredentialsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list credentials request is required")
	}
	if s.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := s.credentialStore.ListCredentialsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
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
func (s *Service) RevokeCredential(ctx context.Context, in *aiv1.RevokeCredentialRequest) (*aiv1.RevokeCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke credential request is required")
	}
	if s.credentialStore == nil {
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

	revokedAt := s.clock().UTC()
	if err := s.credentialStore.RevokeCredential(ctx, userID, credentialID, revokedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		return nil, status.Errorf(codes.Internal, "revoke credential: %v", err)
	}

	record, err := s.credentialStore.GetCredential(ctx, credentialID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		return nil, status.Errorf(codes.Internal, "get credential: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "credential not found")
	}

	return &aiv1.RevokeCredentialResponse{
		Credential: credentialToProto(record),
	}, nil
}
