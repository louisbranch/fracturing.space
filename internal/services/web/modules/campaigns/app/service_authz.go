package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// AuthorizationDecision captures one authorization check result.
type AuthorizationDecision struct {
	CheckID    string
	Evaluated  bool
	Allowed    bool
	ReasonCode string
}

// AuthorizationTarget scopes a check to a specific resource instance.
type AuthorizationTarget struct {
	ResourceID string
}

// AuthorizationCheck describes one authz request in batch evaluation.
type AuthorizationCheck struct {
	CheckID  string
	Action   AuthorizationAction
	Resource AuthorizationResource
	Target   *AuthorizationTarget
}

// AuthorizationGateway performs unary campaign action checks.
type AuthorizationGateway interface {
	CanCampaignAction(context.Context, string, AuthorizationAction, AuthorizationResource, *AuthorizationTarget) (AuthorizationDecision, error)
}

// BatchAuthorizationGateway performs batched campaign action checks.
type BatchAuthorizationGateway interface {
	BatchCanCampaignAction(context.Context, string, []AuthorizationCheck) ([]AuthorizationDecision, error)
}

// AuthzGateway combines unary and batch authorization checks.
type AuthzGateway interface {
	AuthorizationGateway
	BatchAuthorizationGateway
}

// --- Authorization policy table ---

// AuthorizationAction defines an action dimension for campaign authz checks.
type AuthorizationAction string

// AuthorizationResource defines a resource dimension for campaign authz checks.
type AuthorizationResource string

const (
	AuthorizationActionManage AuthorizationAction = "manage"
	AuthorizationActionMutate AuthorizationAction = "mutate"

	AuthorizationResourceSession     AuthorizationResource = "session"
	AuthorizationResourceParticipant AuthorizationResource = "participant"
	AuthorizationResourceCharacter   AuthorizationResource = "character"
	AuthorizationResourceInvite      AuthorizationResource = "invite"
)

// Compatibility aliases keep package-local service/test names stable while the
// exported contracts are consumed by sibling packages.
type campaignAuthorizationDecision = AuthorizationDecision
type campaignAuthorizationTarget = AuthorizationTarget
type campaignAuthorizationCheck = AuthorizationCheck
type campaignAuthorizationGateway = AuthorizationGateway
type campaignBatchAuthorizationGateway = BatchAuthorizationGateway
type campaignAuthzGateway = AuthzGateway
type campaignAuthorizationAction = AuthorizationAction
type campaignAuthorizationResource = AuthorizationResource

// mutationAuthzPolicy declares the authorization requirement for a single
// mutation gateway method.
type mutationAuthzPolicy struct {
	action   campaignAuthorizationAction
	resource campaignAuthorizationResource
	denyKey  string
	denyMsg  string
}

const (
	campaignAuthzActionManage campaignAuthorizationAction = AuthorizationActionManage
	campaignAuthzActionMutate campaignAuthorizationAction = AuthorizationActionMutate

	campaignAuthzResourceSession     campaignAuthorizationResource = AuthorizationResourceSession
	campaignAuthzResourceParticipant campaignAuthorizationResource = AuthorizationResourceParticipant
	campaignAuthzResourceCharacter   campaignAuthorizationResource = AuthorizationResourceCharacter
	campaignAuthzResourceInvite      campaignAuthorizationResource = AuthorizationResourceInvite
)

var (
	policyManageSession = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceSession,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_session_action",
		denyMsg:  "manager or owner access required for session action",
	}
	policyManageParticipant = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceParticipant,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_participant_action",
		denyMsg:  "manager or owner access required for participant action",
	}
	policyMutateCharacter = mutationAuthzPolicy{
		action:   campaignAuthzActionMutate,
		resource: campaignAuthzResourceCharacter,
		denyKey:  "error.web.message.campaign_membership_required_for_character_action",
		denyMsg:  "campaign membership required for character action",
	}
	policyManageCharacter = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceCharacter,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_character_action",
		denyMsg:  "manager or owner access required for character action",
	}
	policyManageInvite = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceInvite,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_invite_action",
		denyMsg:  "manager or owner access required for invite action",
	}
)

// --- Service-level authz helpers ---

// requirePolicy enforces a policy-table authorization check for a mutation.
func (s service) requirePolicy(ctx context.Context, campaignID string, p mutationAuthzPolicy) error {
	return s.requireCampaignActionAccess(ctx, campaignID, p.action, p.resource, nil, p.denyKey, p.denyMsg)
}

// requirePolicyWithTarget enforces a policy-table authorization check scoped to a specific resource.
func (s service) requirePolicyWithTarget(ctx context.Context, campaignID string, p mutationAuthzPolicy, resourceID string) error {
	return s.requireCampaignActionAccess(ctx, campaignID, p.action, p.resource, &campaignAuthorizationTarget{ResourceID: resourceID}, p.denyKey, p.denyMsg)
}

func (s service) requireCampaignActionAccess(
	ctx context.Context,
	campaignID string,
	action campaignAuthorizationAction,
	resource campaignAuthorizationResource,
	target *campaignAuthorizationTarget,
	denyMessageKey string,
	denyMessage string,
) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	if s.authzGateway == nil {
		return apperrors.EK(apperrors.KindForbidden, denyMessageKey, denyMessage)
	}
	decision, err := s.authzGateway.CanCampaignAction(
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

func (s service) hydrateCharacterEditability(ctx context.Context, campaignID string, characters []CampaignCharacter) {
	if len(characters) == 0 {
		return
	}
	if s.authzGateway == nil {
		return
	}

	checks := make([]campaignAuthorizationCheck, 0, len(characters))
	indexesByCheckID := make(map[string][]int, len(characters))
	for idx := range characters {
		characterID := strings.TrimSpace(characters[idx].ID)
		if characterID == "" {
			continue
		}
		indexesByCheckID[characterID] = append(indexesByCheckID[characterID], idx)
		if len(indexesByCheckID[characterID]) > 1 {
			continue
		}
		checks = append(checks, campaignAuthorizationCheck{
			CheckID:  characterID,
			Action:   campaignAuthzActionMutate,
			Resource: campaignAuthzResourceCharacter,
			Target: &campaignAuthorizationTarget{
				ResourceID: characterID,
			},
		})
	}
	if len(checks) == 0 {
		return
	}

	decisions, err := s.authzGateway.BatchCanCampaignAction(ctx, campaignID, checks)
	if err != nil {
		return
	}

	for idx, decision := range decisions {
		checkID := strings.TrimSpace(decision.CheckID)
		if checkID == "" && idx < len(checks) {
			checkID = strings.TrimSpace(checks[idx].CheckID)
		}
		if checkID == "" {
			continue
		}
		characterIndexes, found := indexesByCheckID[checkID]
		if !found {
			continue
		}
		for _, characterIndex := range characterIndexes {
			characters[characterIndex].EditReasonCode = strings.TrimSpace(decision.ReasonCode)
			if decision.Evaluated && decision.Allowed {
				characters[characterIndex].CanEdit = true
			}
		}
	}
}
