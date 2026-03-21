package ai

import (
	"context"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CredentialHandlers serves credential RPCs as thin transport wrappers over
// the credential service.
type CredentialHandlers struct {
	aiv1.UnimplementedCredentialServiceServer
	svc *service.CredentialService
}

// CredentialHandlersConfig declares the dependencies for credential RPCs.
type CredentialHandlersConfig struct {
	CredentialService *service.CredentialService
}

// NewCredentialHandlers builds a credential RPC server from a credential service.
func NewCredentialHandlers(cfg CredentialHandlersConfig) (*CredentialHandlers, error) {
	if cfg.CredentialService == nil {
		return nil, fmt.Errorf("ai: NewCredentialHandlers: credential service is required")
	}
	return &CredentialHandlers{svc: cfg.CredentialService}, nil
}

// CreateCredential creates one user-owned provider credential.
func (h *CredentialHandlers) CreateCredential(ctx context.Context, in *aiv1.CreateCredentialRequest) (*aiv1.CreateCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create credential request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	record, err := h.svc.Create(ctx, service.CreateCredentialInput{
		OwnerUserID: userID,
		Provider:    providerID,
		Label:       in.GetLabel(),
		Secret:      in.GetSecret(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.CreateCredentialResponse{Credential: credentialToProto(record)}, nil
}

// ListCredentials returns a page of credentials owned by the caller.
func (h *CredentialHandlers) ListCredentials(ctx context.Context, in *aiv1.ListCredentialsRequest) (*aiv1.ListCredentialsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list credentials request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := h.svc.List(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, serviceErrorToStatus(err)
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
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	credentialID := strings.TrimSpace(in.GetCredentialId())
	if credentialID == "" {
		return nil, status.Error(codes.InvalidArgument, "credential_id is required")
	}

	record, err := h.svc.Revoke(ctx, userID, credentialID)
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.RevokeCredentialResponse{Credential: credentialToProto(record)}, nil
}
