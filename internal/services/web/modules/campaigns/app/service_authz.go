package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// --- Service-level authz helpers ---

// requireManageCampaign enforces campaign-manage access for owner/manager workflows.
func (s authorizationSupport) requireManageCampaign(ctx context.Context, campaignID string) error {
	return s.requirePolicy(ctx, campaignID, policyManageCampaign)
}

// requireManageParticipants enforces participant-manage access for owner/manager workflows.
func (s authorizationSupport) requireManageParticipants(ctx context.Context, campaignID string) error {
	return s.requirePolicy(ctx, campaignID, policyManageParticipant)
}

// requireManageInvites enforces invite-manage access for owner/manager workflows.
func (s authorizationSupport) requireManageInvites(ctx context.Context, campaignID string) error {
	return s.requirePolicy(ctx, campaignID, policyManageInvite)
}

// requireMutateCharacters enforces baseline character-mutation access.
func (s authorizationSupport) requireMutateCharacters(ctx context.Context, campaignID string) error {
	return s.requirePolicy(ctx, campaignID, policyMutateCharacter)
}

// requirePolicy enforces a policy-table authorization check for a mutation.
func (s authorizationSupport) requirePolicy(ctx context.Context, campaignID string, p mutationAuthzPolicy) error {
	return s.requireCampaignActionAccess(ctx, campaignID, p.action, p.resource, nil, p.denyKey, p.denyMsg)
}

// requirePolicyWithTarget enforces a policy-table authorization check scoped to a specific resource.
func (s authorizationSupport) requirePolicyWithTarget(ctx context.Context, campaignID string, p mutationAuthzPolicy, resourceID string) error {
	return s.requireCampaignActionAccess(ctx, campaignID, p.action, p.resource, &AuthorizationTarget{ResourceID: resourceID}, p.denyKey, p.denyMsg)
}

// requireCampaignActionAccess enforces this package invariant before continuing flow.
func (s authorizationSupport) requireCampaignActionAccess(
	ctx context.Context,
	campaignID string,
	action AuthorizationAction,
	resource AuthorizationResource,
	target *AuthorizationTarget,
	denyMessageKey string,
	denyMessage string,
) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	if s.gateway == nil {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	decision, err := s.gateway.CanCampaignAction(
		ctx,
		campaignID,
		action,
		resource,
		target,
	)
	if err != nil {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	if !decision.Evaluated || !decision.Allowed {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	return nil
}
