package app

// mutationAuthzPolicy declares the authorization requirement for a single
// mutation gateway method.
type mutationAuthzPolicy struct {
	action   AuthorizationAction
	resource AuthorizationResource
	denyKey  string
	denyMsg  string
}

const (
	campaignAuthzActionManage AuthorizationAction = AuthorizationActionManage
	campaignAuthzActionMutate AuthorizationAction = AuthorizationActionMutate

	campaignAuthzResourceSession     AuthorizationResource = AuthorizationResourceSession
	campaignAuthzResourceCampaign    AuthorizationResource = AuthorizationResourceCampaign
	campaignAuthzResourceParticipant AuthorizationResource = AuthorizationResourceParticipant
	campaignAuthzResourceCharacter   AuthorizationResource = AuthorizationResourceCharacter
	campaignAuthzResourceInvite      AuthorizationResource = AuthorizationResourceInvite
)

var (
	policyManageSession = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceSession,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_session_action",
		denyMsg:  "manager or owner access required for session action",
	}
	policyManageCampaign = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceCampaign,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_campaign_action",
		denyMsg:  "manager or owner access required for campaign action",
	}
	policyMutateCharacter = mutationAuthzPolicy{
		action:   campaignAuthzActionMutate,
		resource: campaignAuthzResourceCharacter,
		denyKey:  "error.web.message.campaign_membership_required_for_character_action",
		denyMsg:  "campaign membership required for character action",
	}
	policyManageInvite = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceInvite,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_invite_action",
		denyMsg:  "manager or owner access required for invite action",
	}
	policyManageParticipant = mutationAuthzPolicy{
		action:   campaignAuthzActionManage,
		resource: campaignAuthzResourceParticipant,
		denyKey:  "error.web.message.manager_or_owner_access_required_for_participant_action",
		denyMsg:  "manager or owner access required for participant action",
	}
)
