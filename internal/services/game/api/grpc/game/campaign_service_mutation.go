package game

import (
	"context"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateCampaign updates mutable campaign metadata fields.
func (s *CampaignService) UpdateCampaign(ctx context.Context, in *campaignv1.UpdateCampaignRequest) (*campaignv1.UpdateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update campaign request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	var update campaignUpdateInput
	if name := in.GetName(); name != nil {
		value := name.GetValue()
		if err := validate.MaxLength(value, "name", validate.MaxNameLen); err != nil {
			return nil, err
		}
		update.Name = &value
	}
	if themePrompt := in.GetThemePrompt(); themePrompt != nil {
		value := themePrompt.GetValue()
		if err := validate.MaxLength(value, "theme prompt", validate.MaxPromptLen); err != nil {
			return nil, err
		}
		update.ThemePrompt = &value
	}
	switch locale := in.GetLocale(); locale {
	case commonv1.Locale_LOCALE_UNSPECIFIED:
		// optional field omitted
	case commonv1.Locale_LOCALE_EN_US, commonv1.Locale_LOCALE_PT_BR:
		value := locale
		update.Locale = &value
	default:
		return nil, status.Error(codes.InvalidArgument, "locale is invalid")
	}

	updated, err := s.app.UpdateCampaign(ctx, campaignID, update)
	if err != nil {
		return nil, err
	}

	return &campaignv1.UpdateCampaignResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}

// EndCampaign marks a campaign as completed.
func (s *CampaignService) EndCampaign(ctx context.Context, in *campaignv1.EndCampaignRequest) (*campaignv1.EndCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end campaign request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.EndCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.EndCampaignResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}

// ArchiveCampaign archives a campaign.
func (s *CampaignService) ArchiveCampaign(ctx context.Context, in *campaignv1.ArchiveCampaignRequest) (*campaignv1.ArchiveCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "archive campaign request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.ArchiveCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.ArchiveCampaignResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}

// RestoreCampaign restores an archived campaign to draft state.
func (s *CampaignService) RestoreCampaign(ctx context.Context, in *campaignv1.RestoreCampaignRequest) (*campaignv1.RestoreCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "restore campaign request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.RestoreCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.RestoreCampaignResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}

// SetCampaignCover updates the selected built-in campaign cover.
func (s *CampaignService) SetCampaignCover(ctx context.Context, in *campaignv1.SetCampaignCoverRequest) (*campaignv1.SetCampaignCoverResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set campaign cover request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	coverAssetID, err := validate.RequiredID(in.GetCoverAssetId(), "cover asset id")
	if err != nil {
		return nil, err
	}
	coverSetID := strings.TrimSpace(in.GetCoverSetId())

	updated, err := s.app.SetCampaignCover(ctx, campaignID, coverAssetID, coverSetID)
	if err != nil {
		return nil, err
	}

	return &campaignv1.SetCampaignCoverResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}

// SetCampaignAIBinding binds an AI agent to a campaign.
func (s *CampaignService) SetCampaignAIBinding(ctx context.Context, in *campaignv1.SetCampaignAIBindingRequest) (*campaignv1.SetCampaignAIBindingResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set campaign ai binding request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	aiAgentID, err := validate.RequiredID(in.GetAiAgentId(), "ai agent id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.SetCampaignAIBinding(ctx, campaignID, aiAgentID)
	if err != nil {
		return nil, err
	}
	return &campaignv1.SetCampaignAIBindingResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}

// ClearCampaignAIBinding clears the AI agent binding from a campaign.
func (s *CampaignService) ClearCampaignAIBinding(ctx context.Context, in *campaignv1.ClearCampaignAIBindingRequest) (*campaignv1.ClearCampaignAIBindingResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "clear campaign ai binding request is required")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}

	updated, err := s.app.ClearCampaignAIBinding(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return &campaignv1.ClearCampaignAIBindingResponse{Campaign: campaigntransport.CampaignToProto(updated)}, nil
}
