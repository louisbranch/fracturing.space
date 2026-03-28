package ai

import (
	"context"
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
)

// CampaignArtifactHandlers serves campaign artifact RPCs with explicit deps.
type CampaignArtifactHandlers struct {
	aiv1.UnimplementedCampaignArtifactServiceServer

	campaignArtifactManager  *campaigncontext.Manager
	campaignContextValidator campaignContextValidator
}

// CampaignArtifactHandlersConfig declares the dependencies for campaign artifact RPCs.
type CampaignArtifactHandlersConfig struct {
	Manager            *campaigncontext.Manager
	CampaignAuthorizer CampaignAccessAuthorizer
}

// NewCampaignArtifactHandlers builds a campaign-artifact RPC server.
func NewCampaignArtifactHandlers(cfg CampaignArtifactHandlersConfig) (*CampaignArtifactHandlers, error) {
	if cfg.Manager == nil {
		return nil, fmt.Errorf("ai: NewCampaignArtifactHandlers: campaign artifact manager is required")
	}
	return &CampaignArtifactHandlers{
		campaignArtifactManager:  cfg.Manager,
		campaignContextValidator: newCampaignContextValidator(cfg.CampaignAuthorizer),
	}, nil
}

// EnsureCampaignArtifacts creates default GM campaign artifacts when missing.
func (h *CampaignArtifactHandlers) EnsureCampaignArtifacts(ctx context.Context, in *aiv1.EnsureCampaignArtifactsRequest) (*aiv1.EnsureCampaignArtifactsResponse, error) {
	if err := requireUnaryRequest(in, "ensure campaign artifacts request is required"); err != nil {
		return nil, err
	}

	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE); err != nil {
		return nil, err
	}
	records, err := h.campaignArtifactManager.EnsureDefaultArtifacts(ctx, in.GetCampaignId(), in.GetStorySeedMarkdown())
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{Operation: "ensure campaign artifacts"})
	}
	resp := &aiv1.EnsureCampaignArtifactsResponse{Artifacts: make([]*aiv1.CampaignArtifact, 0, len(records))}
	for _, record := range records {
		resp.Artifacts = append(resp.Artifacts, campaignArtifactToProto(record))
	}
	return resp, nil
}

// ListCampaignArtifacts returns all persisted artifacts for one campaign.
func (h *CampaignArtifactHandlers) ListCampaignArtifacts(ctx context.Context, in *aiv1.ListCampaignArtifactsRequest) (*aiv1.ListCampaignArtifactsResponse, error) {
	if err := requireUnaryRequest(in, "list campaign artifacts request is required"); err != nil {
		return nil, err
	}

	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return nil, err
	}
	records, err := h.campaignArtifactManager.ListArtifacts(ctx, in.GetCampaignId())
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{Operation: "list campaign artifacts"})
	}
	resp := &aiv1.ListCampaignArtifactsResponse{Artifacts: make([]*aiv1.CampaignArtifact, 0, len(records))}
	for _, record := range records {
		resp.Artifacts = append(resp.Artifacts, campaignArtifactToProto(record))
	}
	return resp, nil
}

// GetCampaignArtifact returns one persisted campaign artifact.
func (h *CampaignArtifactHandlers) GetCampaignArtifact(ctx context.Context, in *aiv1.GetCampaignArtifactRequest) (*aiv1.GetCampaignArtifactResponse, error) {
	if err := requireUnaryRequest(in, "get campaign artifact request is required"); err != nil {
		return nil, err
	}

	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ); err != nil {
		return nil, err
	}
	record, err := h.campaignArtifactManager.GetArtifact(ctx, in.GetCampaignId(), in.GetPath())
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{Operation: "get campaign artifact"})
	}
	return &aiv1.GetCampaignArtifactResponse{Artifact: campaignArtifactToProto(record)}, nil
}

// UpsertCampaignArtifact replaces one mutable campaign artifact body.
func (h *CampaignArtifactHandlers) UpsertCampaignArtifact(ctx context.Context, in *aiv1.UpsertCampaignArtifactRequest) (*aiv1.UpsertCampaignArtifactResponse, error) {
	if err := requireUnaryRequest(in, "upsert campaign artifact request is required"); err != nil {
		return nil, err
	}

	if err := h.campaignContextValidator.validateCampaignContext(ctx, in.GetCampaignId(), gamev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE); err != nil {
		return nil, err
	}
	record, err := h.campaignArtifactManager.UpsertArtifact(ctx, in.GetCampaignId(), in.GetPath(), in.GetContent())
	if err != nil {
		return nil, transportErrorToStatus(err, transportErrorConfig{Operation: "upsert campaign artifact"})
	}
	return &aiv1.UpsertCampaignArtifactResponse{Artifact: campaignArtifactToProto(record)}, nil
}
