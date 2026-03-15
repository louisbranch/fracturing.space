package app

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CampaignAIBindingSummary loads campaign overview AI-binding state without
// requiring the dedicated AI settings page to render.
func (s automationReadService) CampaignAIBindingSummary(
	ctx context.Context,
	campaignID string,
	currentAIAgentID string,
	gmMode string,
) (CampaignAIBindingSummary, error) {
	return s.campaignAIBindingSummary(ctx, campaignID, currentAIAgentID, gmMode)
}

// CampaignAIBindingSettings loads dedicated campaign AI-binding page state.
func (s automationReadService) CampaignAIBindingSettings(
	ctx context.Context,
	campaignID string,
	currentAIAgentID string,
) (CampaignAIBindingSettings, error) {
	return s.campaignAIBindingSettings(ctx, campaignID, currentAIAgentID)
}

// UpdateCampaignAIBinding applies this package workflow transition.
func (s automationMutationService) UpdateCampaignAIBinding(ctx context.Context, campaignID string, input UpdateCampaignAIBindingInput) error {
	return s.updateCampaignAIBinding(ctx, campaignID, input)
}

// campaignAIBindingSummary derives overview status and manage visibility for one campaign.
func (s automationReadService) campaignAIBindingSummary(
	ctx context.Context,
	campaignID string,
	currentAIAgentID string,
	gmMode string,
) (CampaignAIBindingSummary, error) {
	campaignID = strings.TrimSpace(campaignID)
	summary := CampaignAIBindingSummary{
		Status: campaignAIBindingStatus(gmMode, currentAIAgentID),
	}
	if campaignID == "" {
		return summary, nil
	}

	owner, _ := resolveCampaignAIBindingOwner(ctx, s.auth, s.participants, campaignID)
	summary.CanManage = owner
	return summary, nil
}

// campaignAIBindingSettings loads owner-only binding settings for the dedicated campaign page.
func (s automationReadService) campaignAIBindingSettings(
	ctx context.Context,
	campaignID string,
	currentAIAgentID string,
) (CampaignAIBindingSettings, error) {
	campaignID = strings.TrimSpace(campaignID)
	currentAIAgentID = strings.TrimSpace(currentAIAgentID)
	if campaignID == "" {
		return CampaignAIBindingSettings{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	owner, err := resolveCampaignAIBindingOwner(ctx, s.auth, s.participants, campaignID)
	if err != nil || !owner {
		return CampaignAIBindingSettings{}, apperrors.EK(
			apperrors.KindForbidden,
			"error.web.message.owner_access_required_for_campaign_ai_binding",
			"owner access required for campaign AI binding",
		)
	}

	settings := CampaignAIBindingSettings{
		CurrentID: currentAIAgentID,
		Options:   []CampaignAIAgentOption{},
	}

	options, err := s.read.CampaignAIAgents(ctx)
	if err != nil {
		var appErr apperrors.Error
		if stderrors.As(err, &appErr) && appErr.Kind == apperrors.KindUnavailable {
			settings.Unavailable = true
			settings.Options = ensureCurrentAIAgentOption(nil, currentAIAgentID)
			return settings, nil
		}
		return CampaignAIBindingSettings{}, err
	}

	for _, option := range options {
		agentID := strings.TrimSpace(option.ID)
		if agentID == "" {
			continue
		}
		option.Selected = agentID == currentAIAgentID
		settings.Options = append(settings.Options, option)
	}
	settings.Options = ensureCurrentAIAgentOption(settings.Options, currentAIAgentID)
	return settings, nil
}

// updateCampaignAIBinding applies the owner-only binding mutation.
func (s automationMutationService) updateCampaignAIBinding(ctx context.Context, campaignID string, input UpdateCampaignAIBindingInput) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}

	input.AIAgentID = strings.TrimSpace(input.AIAgentID)

	owner, err := resolveCampaignAIBindingOwner(ctx, s.auth, s.participants, campaignID)
	if err != nil || !owner {
		return apperrors.EK(
			apperrors.KindForbidden,
			"error.web.message.owner_access_required_for_campaign_ai_binding",
			"owner access required for campaign AI binding",
		)
	}

	return s.mutation.UpdateCampaignAIBinding(ctx, campaignID, input)
}

// resolveCampaignAIBindingOwner resolves whether the current actor is the campaign owner.
func resolveCampaignAIBindingOwner(
	ctx context.Context,
	auth authorizationSupport,
	participants CampaignParticipantReadGateway,
	campaignID string,
) (bool, error) {
	decision, err := automationCampaignManageDecision(ctx, auth, campaignID)
	if err != nil {
		return false, err
	}
	actorAccess, actorAccessErr := automationCampaignActorAccess(ctx, participants, campaignID, decision)
	if actorAccessErr != nil {
		return false, actorAccessErr
	}
	return decision.Evaluated && decision.Allowed && actorAccess == participantAccessOwner, nil
}

// campaignAIBindingStatus maps workspace GM mode and binding state into overview status.
func campaignAIBindingStatus(gmMode string, currentAIAgentID string) CampaignAIBindingStatus {
	if strings.TrimSpace(currentAIAgentID) != "" {
		return CampaignAIBindingStatusConfigured
	}

	switch strings.ToLower(strings.TrimSpace(gmMode)) {
	case "ai", "hybrid":
		return CampaignAIBindingStatusPending
	default:
		return CampaignAIBindingStatusNotRequired
	}
}

// campaignManageDecision resolves the caller's campaign-manage authz decision,
// including actor campaign access when available.
func automationCampaignManageDecision(ctx context.Context, auth authorizationSupport, campaignID string) (AuthorizationDecision, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || auth.gateway == nil {
		return AuthorizationDecision{}, nil
	}
	return auth.gateway.CanCampaignAction(ctx, campaignID, campaignAuthzActionManage, campaignAuthzResourceCampaign, nil)
}

// campaignActorAccess resolves the caller's campaign access. It prefers authz
// response metadata and falls back to matching the current user against campaign
// participants when the authz transport omits actor access.
func automationCampaignActorAccess(ctx context.Context, participants CampaignParticipantReadGateway, campaignID string, decision AuthorizationDecision) (string, error) {
	if access := participantAccessCanonical(decision.ActorCampaignAccess); access != "" {
		return access, nil
	}

	userID := strings.TrimSpace(grpcauthctx.UserIDFromOutgoingContext(ctx))
	if userID == "" {
		return "", nil
	}

	items, err := participants.CampaignParticipants(ctx, campaignID)
	if err != nil {
		return "", err
	}
	for _, participant := range items {
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
