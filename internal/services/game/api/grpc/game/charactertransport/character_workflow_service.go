package charactertransport

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	daggerheartcreation "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/creationworkflow"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCharacterCreationProgress returns the current creation workflow progress.
func (s *Service) GetCharacterCreationProgress(ctx context.Context, in *campaignv1.GetCharacterCreationProgressRequest) (*campaignv1.GetCharacterCreationProgressResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get character creation progress request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	progress, err := s.app.GetCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		return nil, daggerheartcreation.HandleWorkflowError(ctx, err)
	}

	return &campaignv1.GetCharacterCreationProgressResponse{
		Progress: creationProgressToProto(campaignID, characterID, progress),
	}, nil
}

// ApplyCharacterCreationStep applies one creation workflow step.
func (s *Service) ApplyCharacterCreationStep(ctx context.Context, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.ApplyCharacterCreationStepResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply character creation step request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	profile, progress, err := s.app.ApplyCharacterCreationStep(ctx, campaignID, in)
	if err != nil {
		return nil, daggerheartcreation.HandleWorkflowError(ctx, err)
	}

	return &campaignv1.ApplyCharacterCreationStepResponse{
		Profile:  profile,
		Progress: creationProgressToProto(campaignID, characterID, progress),
	}, nil
}

// ApplyCharacterCreationWorkflow applies all creation workflow steps atomically.
func (s *Service) ApplyCharacterCreationWorkflow(ctx context.Context, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.ApplyCharacterCreationWorkflowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply character creation workflow request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	profile, progress, err := s.app.ApplyCharacterCreationWorkflow(ctx, campaignID, in)
	if err != nil {
		return nil, daggerheartcreation.HandleWorkflowError(ctx, err)
	}

	return &campaignv1.ApplyCharacterCreationWorkflowResponse{
		Profile:  profile,
		Progress: creationProgressToProto(campaignID, characterID, progress),
	}, nil
}

// ResetCharacterCreationWorkflow resets creation workflow data for a character.
func (s *Service) ResetCharacterCreationWorkflow(ctx context.Context, in *campaignv1.ResetCharacterCreationWorkflowRequest) (*campaignv1.ResetCharacterCreationWorkflowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "reset character creation workflow request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	progress, err := s.app.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
	if err != nil {
		return nil, daggerheartcreation.HandleWorkflowError(ctx, err)
	}

	return &campaignv1.ResetCharacterCreationWorkflowResponse{
		Progress: creationProgressToProto(campaignID, characterID, progress),
	}, nil
}

func creationProgressToProto(campaignID, characterID string, progress characterworkflow.Progress) *campaignv1.CharacterCreationProgress {
	steps := make([]*campaignv1.CharacterCreationStepProgress, 0, len(progress.Steps))
	for _, step := range progress.Steps {
		steps = append(steps, &campaignv1.CharacterCreationStepProgress{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}

	return &campaignv1.CharacterCreationProgress{
		CampaignId:   campaignID,
		CharacterId:  characterID,
		Steps:        steps,
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}
