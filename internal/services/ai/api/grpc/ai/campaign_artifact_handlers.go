package ai

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CampaignArtifactHandlers serves campaign artifact RPCs with explicit deps.
type CampaignArtifactHandlers struct {
	aiv1.UnimplementedCampaignArtifactServiceServer

	campaignArtifactManager  *campaigncontext.Manager
	campaignContextValidator campaignContextValidator
}

// CampaignArtifactHandlersConfig declares the dependencies for campaign artifact RPCs.
type CampaignArtifactHandlersConfig struct {
	Manager                  *campaigncontext.Manager
	AuthorizationClient      gamev1.AuthorizationServiceClient
	InternalServiceAllowlist map[string]struct{}
}

// NewCampaignArtifactHandlers builds a campaign-artifact RPC server.
func NewCampaignArtifactHandlers(cfg CampaignArtifactHandlersConfig) *CampaignArtifactHandlers {
	return &CampaignArtifactHandlers{
		campaignArtifactManager:  cfg.Manager,
		campaignContextValidator: newCampaignContextValidator(cfg.AuthorizationClient, cfg.InternalServiceAllowlist),
	}
}

// EnsureCampaignArtifacts creates default GM campaign artifacts when missing.
func (h *CampaignArtifactHandlers) EnsureCampaignArtifacts(ctx context.Context, in *aiv1.EnsureCampaignArtifactsRequest) (*aiv1.EnsureCampaignArtifactsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "ensure campaign artifacts request is required")
	}
	if h == nil || h.campaignArtifactManager == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign artifact manager is unavailable")
	}
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE); err != nil {
		return nil, err
	}
	records, err := h.campaignArtifactManager.EnsureDefaultArtifacts(ctx, in.GetCampaignId(), in.GetStorySeedMarkdown())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ensure campaign artifacts: %v", err)
	}
	resp := &aiv1.EnsureCampaignArtifactsResponse{Artifacts: make([]*aiv1.CampaignArtifact, 0, len(records))}
	for _, record := range records {
		resp.Artifacts = append(resp.Artifacts, campaignArtifactToProto(record))
	}
	return resp, nil
}

// ListCampaignArtifacts returns all persisted artifacts for one campaign.
func (h *CampaignArtifactHandlers) ListCampaignArtifacts(ctx context.Context, in *aiv1.ListCampaignArtifactsRequest) (*aiv1.ListCampaignArtifactsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaign artifacts request is required")
	}
	if h == nil || h.campaignArtifactManager == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign artifact manager is unavailable")
	}
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return nil, err
	}
	records, err := h.campaignArtifactManager.ListArtifacts(ctx, in.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaign artifacts: %v", err)
	}
	resp := &aiv1.ListCampaignArtifactsResponse{Artifacts: make([]*aiv1.CampaignArtifact, 0, len(records))}
	for _, record := range records {
		resp.Artifacts = append(resp.Artifacts, campaignArtifactToProto(record))
	}
	return resp, nil
}

// GetCampaignArtifact returns one persisted campaign artifact.
func (h *CampaignArtifactHandlers) GetCampaignArtifact(ctx context.Context, in *aiv1.GetCampaignArtifactRequest) (*aiv1.GetCampaignArtifactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign artifact request is required")
	}
	if h == nil || h.campaignArtifactManager == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign artifact manager is unavailable")
	}
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return nil, err
	}
	record, err := h.campaignArtifactManager.GetArtifact(ctx, in.GetCampaignId(), in.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get campaign artifact: %v", err)
	}
	return &aiv1.GetCampaignArtifactResponse{Artifact: campaignArtifactToProto(record)}, nil
}

// UpsertCampaignArtifact replaces one mutable campaign artifact body.
func (h *CampaignArtifactHandlers) UpsertCampaignArtifact(ctx context.Context, in *aiv1.UpsertCampaignArtifactRequest) (*aiv1.UpsertCampaignArtifactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "upsert campaign artifact request is required")
	}
	if h == nil || h.campaignArtifactManager == nil {
		return nil, status.Error(codes.FailedPrecondition, "campaign artifact manager is unavailable")
	}
	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE); err != nil {
		return nil, err
	}
	record, err := h.campaignArtifactManager.UpsertArtifact(ctx, in.GetCampaignId(), in.GetPath(), in.GetContent())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert campaign artifact: %v", err)
	}
	return &aiv1.UpsertCampaignArtifactResponse{Artifact: campaignArtifactToProto(record)}, nil
}
