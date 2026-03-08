package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// CanCampaignAction centralizes this web behavior in one helper seam.
func (g GRPCGateway) CanCampaignAction(
	ctx context.Context,
	campaignID string,
	action campaignapp.AuthorizationAction,
	resource campaignapp.AuthorizationResource,
	target *campaignapp.AuthorizationTarget,
) (campaignapp.AuthorizationDecision, error) {
	if g.AuthorizationClient == nil {
		return campaignapp.AuthorizationDecision{}, nil
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignapp.AuthorizationDecision{}, nil
	}
	resp, err := g.AuthorizationClient.Can(ctx, &statev1.CanRequest{
		CampaignId: campaignID,
		Action:     mapCampaignAuthorizationActionToProto(action),
		Resource:   mapCampaignAuthorizationResourceToProto(resource),
		Target:     mapCampaignAuthorizationTargetToProto(target),
	})
	if err != nil {
		return campaignapp.AuthorizationDecision{}, err
	}
	if resp == nil {
		return campaignapp.AuthorizationDecision{}, nil
	}
	return campaignapp.AuthorizationDecision{
		CheckID:             "",
		Evaluated:           true,
		Allowed:             resp.GetAllowed(),
		ReasonCode:          strings.TrimSpace(resp.GetReasonCode()),
		ActorCampaignAccess: participantCampaignAccessLabel(resp.GetActorCampaignAccess()),
	}, nil
}

// BatchCanCampaignAction centralizes this web behavior in one helper seam.
func (g GRPCGateway) BatchCanCampaignAction(
	ctx context.Context,
	campaignID string,
	checks []campaignapp.AuthorizationCheck,
) ([]campaignapp.AuthorizationDecision, error) {
	if g.AuthorizationClient == nil {
		return nil, nil
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || len(checks) == 0 {
		return nil, nil
	}

	protoChecks := make([]*statev1.BatchCanCheck, 0, len(checks))
	for _, check := range checks {
		target := mapCampaignAuthorizationTargetToProto(check.Target)
		protoChecks = append(protoChecks, &statev1.BatchCanCheck{
			CheckId:    strings.TrimSpace(check.CheckID),
			CampaignId: campaignID,
			Action:     mapCampaignAuthorizationActionToProto(check.Action),
			Resource:   mapCampaignAuthorizationResourceToProto(check.Resource),
			Target:     target,
		})
	}

	resp, err := g.AuthorizationClient.BatchCan(ctx, &statev1.BatchCanRequest{Checks: protoChecks})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	results := resp.GetResults()
	decisions := make([]campaignapp.AuthorizationDecision, 0, len(results))
	for idx, result := range results {
		if result == nil {
			fallbackCheckID := ""
			if idx < len(checks) {
				fallbackCheckID = strings.TrimSpace(checks[idx].CheckID)
			}
			decisions = append(decisions, campaignapp.AuthorizationDecision{CheckID: fallbackCheckID})
			continue
		}
		checkID := strings.TrimSpace(result.GetCheckId())
		if checkID == "" && idx < len(checks) {
			checkID = strings.TrimSpace(checks[idx].CheckID)
		}
		decisions = append(decisions, campaignapp.AuthorizationDecision{
			CheckID:             checkID,
			Evaluated:           true,
			Allowed:             result.GetAllowed(),
			ReasonCode:          strings.TrimSpace(result.GetReasonCode()),
			ActorCampaignAccess: participantCampaignAccessLabel(result.GetActorCampaignAccess()),
		})
	}

	return decisions, nil
}

// mapCampaignAuthorizationActionToProto maps values across transport and domain boundaries.
func mapCampaignAuthorizationActionToProto(action campaignapp.AuthorizationAction) statev1.AuthorizationAction {
	switch campaignapp.AuthorizationAction(strings.TrimSpace(string(action))) {
	case campaignapp.AuthorizationActionManage:
		return statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE
	case campaignapp.AuthorizationActionMutate:
		return statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE
	default:
		return statev1.AuthorizationAction_AUTHORIZATION_ACTION_UNSPECIFIED
	}
}

// mapCampaignAuthorizationResourceToProto maps values across transport and domain boundaries.
func mapCampaignAuthorizationResourceToProto(resource campaignapp.AuthorizationResource) statev1.AuthorizationResource {
	switch campaignapp.AuthorizationResource(strings.TrimSpace(string(resource))) {
	case campaignapp.AuthorizationResourceCampaign:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN
	case campaignapp.AuthorizationResourceSession:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION
	case campaignapp.AuthorizationResourceParticipant:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT
	case campaignapp.AuthorizationResourceCharacter:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER
	case campaignapp.AuthorizationResourceInvite:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE
	default:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_UNSPECIFIED
	}
}

// mapCampaignAuthorizationTargetToProto maps values across transport and domain boundaries.
func mapCampaignAuthorizationTargetToProto(target *campaignapp.AuthorizationTarget) *statev1.AuthorizationTarget {
	if target == nil {
		return nil
	}
	resourceID := strings.TrimSpace(target.ResourceID)
	ownerParticipantID := strings.TrimSpace(target.OwnerParticipantID)
	targetParticipantID := strings.TrimSpace(target.TargetParticipantID)
	targetCampaignAccess := mapParticipantAccessToProto(target.TargetCampaignAccess)
	requestedCampaignAccess := mapParticipantAccessToProto(target.RequestedCampaignAccess)
	participantOperation := mapParticipantGovernanceOperationToProto(target.ParticipantOperation)
	if resourceID == "" &&
		ownerParticipantID == "" &&
		targetParticipantID == "" &&
		targetCampaignAccess == statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED &&
		requestedCampaignAccess == statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED &&
		participantOperation == statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_UNSPECIFIED {
		return nil
	}
	protoTarget := &statev1.AuthorizationTarget{
		ResourceId:           resourceID,
		OwnerParticipantId:   ownerParticipantID,
		TargetParticipantId:  targetParticipantID,
		ParticipantOperation: participantOperation,
	}
	if targetCampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		protoTarget.TargetCampaignAccess = targetCampaignAccess
	}
	if requestedCampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		protoTarget.RequestedCampaignAccess = requestedCampaignAccess
	}
	return protoTarget
}

// mapParticipantGovernanceOperationToProto maps participant-governance operation labels to proto enums.
func mapParticipantGovernanceOperationToProto(operation campaignapp.ParticipantGovernanceOperation) statev1.ParticipantGovernanceOperation {
	switch strings.ToLower(strings.TrimSpace(string(operation))) {
	case "mutate":
		return statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_MUTATE
	case "access_change":
		return statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_ACCESS_CHANGE
	case "remove":
		return statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_REMOVE
	default:
		return statev1.ParticipantGovernanceOperation_PARTICIPANT_GOVERNANCE_OPERATION_UNSPECIFIED
	}
}
