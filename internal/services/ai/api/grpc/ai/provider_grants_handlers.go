package ai

import (
	"context"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProviderGrantHandlers serves provider grant RPCs as thin transport wrappers
// over the provider grant service.
type ProviderGrantHandlers struct {
	aiv1.UnimplementedProviderGrantServiceServer
	svc *service.ProviderGrantService
}

// ProviderGrantHandlersConfig declares the dependencies for provider-grant RPCs.
type ProviderGrantHandlersConfig struct {
	ProviderGrantService *service.ProviderGrantService
}

// NewProviderGrantHandlers builds a provider-grant RPC server from a provider
// grant service.
func NewProviderGrantHandlers(cfg ProviderGrantHandlersConfig) (*ProviderGrantHandlers, error) {
	if cfg.ProviderGrantService == nil {
		return nil, fmt.Errorf("ai: NewProviderGrantHandlers: provider grant service is required")
	}
	return &ProviderGrantHandlers{svc: cfg.ProviderGrantService}, nil
}

// StartProviderConnect starts a provider OAuth grant handshake.
func (h *ProviderGrantHandlers) StartProviderConnect(ctx context.Context, in *aiv1.StartProviderConnectRequest) (*aiv1.StartProviderConnectResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start provider connect request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	out, err := h.svc.StartConnect(ctx, service.StartConnectInput{
		OwnerUserID:     userID,
		Provider:        providerID,
		RequestedScopes: in.GetRequestedScopes(),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.StartProviderConnectResponse{
		ConnectSessionId: out.ConnectSessionID,
		State:            out.State,
		AuthorizationUrl: out.AuthorizationURL,
		ExpiresAt:        timestamppb.New(out.ExpiresAt),
	}, nil
}

// FinishProviderConnect completes a provider OAuth grant handshake.
func (h *ProviderGrantHandlers) FinishProviderConnect(ctx context.Context, in *aiv1.FinishProviderConnectRequest) (*aiv1.FinishProviderConnectResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "finish provider connect request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	record, err := h.svc.FinishConnect(ctx, service.FinishConnectInput{
		OwnerUserID:       userID,
		ConnectSessionID:  strings.TrimSpace(in.GetConnectSessionId()),
		State:             strings.TrimSpace(in.GetState()),
		AuthorizationCode: strings.TrimSpace(in.GetAuthorizationCode()),
	})
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.FinishProviderConnectResponse{ProviderGrant: providerGrantToProto(record)}, nil
}

// ListProviderGrants returns a page of provider grants owned by the caller.
func (h *ProviderGrantHandlers) ListProviderGrants(ctx context.Context, in *aiv1.ListProviderGrantsRequest) (*aiv1.ListProviderGrantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list provider grants request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	filter, err := providerGrantFilterFromRequest(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page, err := h.svc.List(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken(), filter)
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}

	resp := &aiv1.ListProviderGrantsResponse{
		NextPageToken:  page.NextPageToken,
		ProviderGrants: make([]*aiv1.ProviderGrant, 0, len(page.ProviderGrants)),
	}
	for _, rec := range page.ProviderGrants {
		resp.ProviderGrants = append(resp.ProviderGrants, providerGrantToProto(rec))
	}
	return resp, nil
}

// RevokeProviderGrant revokes one provider grant owned by the caller.
func (h *ProviderGrantHandlers) RevokeProviderGrant(ctx context.Context, in *aiv1.RevokeProviderGrantRequest) (*aiv1.RevokeProviderGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke provider grant request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	providerGrantID := strings.TrimSpace(in.GetProviderGrantId())
	if providerGrantID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_grant_id is required")
	}

	record, err := h.svc.Revoke(ctx, userID, providerGrantID)
	if err != nil {
		return nil, serviceErrorToStatus(err)
	}
	return &aiv1.RevokeProviderGrantResponse{ProviderGrant: providerGrantToProto(record)}, nil
}

func providerGrantFilterFromRequest(in *aiv1.ListProviderGrantsRequest) (providergrant.Filter, error) {
	var filter providergrant.Filter
	switch in.GetProvider() {
	case aiv1.Provider_PROVIDER_UNSPECIFIED:
	case aiv1.Provider_PROVIDER_OPENAI:
		filter.Provider = provider.OpenAI
	default:
		return providergrant.Filter{}, fmt.Errorf("provider filter is invalid")
	}

	switch in.GetStatus() {
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED:
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE:
		filter.Status = providergrant.StatusActive
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED:
		filter.Status = providergrant.StatusRevoked
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_EXPIRED:
		filter.Status = providergrant.StatusExpired
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED:
		filter.Status = providergrant.StatusRefreshFailed
	default:
		return providergrant.Filter{}, fmt.Errorf("status filter is invalid")
	}
	return filter, nil
}
