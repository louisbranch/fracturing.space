package app

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// campaignAIBindingEditor centralizes owner-only AI binding state for the
// AI participant edit page without making the page fail when the AI service is
// unavailable.
func (s service) campaignAIBindingEditor(ctx context.Context, campaignID string, currentAIAgentID string) (CampaignAIBindingEditor, error) {
	campaignID = strings.TrimSpace(campaignID)
	currentAIAgentID = strings.TrimSpace(currentAIAgentID)
	editor := CampaignAIBindingEditor{
		Visible:   true,
		CurrentID: currentAIAgentID,
		Options:   []CampaignAIAgentOption{},
	}
	if campaignID == "" {
		return editor, nil
	}

	decision, err := s.campaignManageDecision(ctx, campaignID)
	actorAccess, actorAccessErr := s.campaignActorAccess(ctx, campaignID, decision)
	if actorAccessErr == nil && err == nil && decision.Evaluated && decision.Allowed && actorAccess == participantAccessOwner {
		editor.Enabled = true
	}

	options, err := s.readGateway.CampaignAIAgents(ctx)
	if err != nil {
		var appErr apperrors.Error
		if stderrors.As(err, &appErr) && appErr.Kind == apperrors.KindUnavailable {
			editor.Unavailable = true
			editor.Options = ensureCurrentAIAgentOption(nil, currentAIAgentID)
			return editor, nil
		}
		return CampaignAIBindingEditor{}, err
	}

	for _, option := range options {
		agentID := strings.TrimSpace(option.ID)
		if agentID == "" {
			continue
		}
		option.Selected = agentID == currentAIAgentID
		editor.Options = append(editor.Options, option)
	}
	editor.Options = ensureCurrentAIAgentOption(editor.Options, currentAIAgentID)
	return editor, nil
}

// updateCampaignAIBinding applies the owner-only binding mutation.
func (s service) updateCampaignAIBinding(ctx context.Context, campaignID string, input UpdateCampaignAIBindingInput) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	input.ParticipantID = strings.TrimSpace(input.ParticipantID)
	if input.ParticipantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}
	input.AIAgentID = strings.TrimSpace(input.AIAgentID)

	decision, err := s.campaignManageDecision(ctx, campaignID)
	actorAccess, actorAccessErr := s.campaignActorAccess(ctx, campaignID, decision)
	if err != nil || actorAccessErr != nil || !decision.Evaluated || !decision.Allowed || actorAccess != participantAccessOwner {
		return apperrors.EK(
			apperrors.KindForbidden,
			"error.web.message.owner_access_required_for_campaign_ai_binding",
			"owner access required for campaign AI binding",
		)
	}

	return s.mutationGateway.UpdateCampaignAIBinding(ctx, campaignID, input)
}

// campaignManageDecision resolves the caller's campaign-manage authz decision,
// including actor campaign access when available.
func (s service) campaignManageDecision(ctx context.Context, campaignID string) (AuthorizationDecision, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || s.authzGateway == nil {
		return AuthorizationDecision{}, nil
	}
	return s.authzGateway.CanCampaignAction(ctx, campaignID, campaignAuthzActionManage, campaignAuthzResourceCampaign, nil)
}

// campaignActorAccess resolves the caller's campaign access. It prefers authz
// response metadata and falls back to matching the current user against campaign
// participants when the authz transport omits actor access.
func (s service) campaignActorAccess(ctx context.Context, campaignID string, decision AuthorizationDecision) (string, error) {
	if access := participantAccessCanonical(decision.ActorCampaignAccess); access != "" {
		return access, nil
	}

	userID := strings.TrimSpace(grpcauthctx.UserIDFromOutgoingContext(ctx))
	if userID == "" {
		return "", nil
	}

	participants, err := s.readGateway.CampaignParticipants(ctx, campaignID)
	if err != nil {
		return "", err
	}
	for _, participant := range participants {
		if strings.TrimSpace(participant.UserID) != userID {
			continue
		}
		return participantAccessCanonical(participant.CampaignAccess), nil
	}
	return "", nil
}

// ensureCurrentAIAgentOption preserves the currently bound agent in the UI
// even when it is inactive or missing from the active option list.
func ensureCurrentAIAgentOption(options []CampaignAIAgentOption, currentAIAgentID string) []CampaignAIAgentOption {
	currentAIAgentID = strings.TrimSpace(currentAIAgentID)
	if currentAIAgentID == "" {
		return options
	}
	for idx := range options {
		if strings.TrimSpace(options[idx].ID) != currentAIAgentID {
			continue
		}
		options[idx].Selected = true
		return options
	}
	return append(options, CampaignAIAgentOption{
		ID:       currentAIAgentID,
		Label:    currentAIAgentID,
		Enabled:  false,
		Selected: true,
	})
}
