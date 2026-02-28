package campaigns

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

func (g grpcGateway) CanCampaignAction(
	ctx context.Context,
	campaignID string,
	action campaignAuthorizationAction,
	resource campaignAuthorizationResource,
	target *campaignAuthorizationTarget,
) (campaignAuthorizationDecision, error) {
	if g.authorizationClient == nil {
		return campaignAuthorizationDecision{}, nil
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignAuthorizationDecision{}, nil
	}
	resp, err := g.authorizationClient.Can(ctx, &statev1.CanRequest{
		CampaignId: campaignID,
		Action:     mapCampaignAuthorizationActionToProto(action),
		Resource:   mapCampaignAuthorizationResourceToProto(resource),
		Target:     mapCampaignAuthorizationTargetToProto(target),
	})
	if err != nil {
		return campaignAuthorizationDecision{}, err
	}
	if resp == nil {
		return campaignAuthorizationDecision{}, nil
	}
	return campaignAuthorizationDecision{
		CheckID:    "",
		Evaluated:  true,
		Allowed:    resp.GetAllowed(),
		ReasonCode: strings.TrimSpace(resp.GetReasonCode()),
	}, nil
}

func (g grpcGateway) BatchCanCampaignAction(
	ctx context.Context,
	campaignID string,
	checks []campaignAuthorizationCheck,
) ([]campaignAuthorizationDecision, error) {
	if g.authorizationClient == nil {
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

	resp, err := g.authorizationClient.BatchCan(ctx, &statev1.BatchCanRequest{Checks: protoChecks})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	results := resp.GetResults()
	decisions := make([]campaignAuthorizationDecision, 0, len(results))
	for idx, result := range results {
		if result == nil {
			fallbackCheckID := ""
			if idx < len(checks) {
				fallbackCheckID = strings.TrimSpace(checks[idx].CheckID)
			}
			decisions = append(decisions, campaignAuthorizationDecision{CheckID: fallbackCheckID})
			continue
		}
		checkID := strings.TrimSpace(result.GetCheckId())
		if checkID == "" && idx < len(checks) {
			checkID = strings.TrimSpace(checks[idx].CheckID)
		}
		decisions = append(decisions, campaignAuthorizationDecision{
			CheckID:    checkID,
			Evaluated:  true,
			Allowed:    result.GetAllowed(),
			ReasonCode: strings.TrimSpace(result.GetReasonCode()),
		})
	}

	return decisions, nil
}

func mapCampaignAuthorizationActionToProto(action campaignAuthorizationAction) statev1.AuthorizationAction {
	switch campaignAuthorizationAction(strings.TrimSpace(string(action))) {
	case campaignAuthzActionManage:
		return statev1.AuthorizationAction_AUTHORIZATION_ACTION_MANAGE
	case campaignAuthzActionMutate:
		return statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE
	default:
		return statev1.AuthorizationAction_AUTHORIZATION_ACTION_UNSPECIFIED
	}
}

func mapCampaignAuthorizationResourceToProto(resource campaignAuthorizationResource) statev1.AuthorizationResource {
	switch campaignAuthorizationResource(strings.TrimSpace(string(resource))) {
	case campaignAuthzResourceSession:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_SESSION
	case campaignAuthzResourceParticipant:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_PARTICIPANT
	case campaignAuthzResourceCharacter:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER
	case campaignAuthzResourceInvite:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_INVITE
	default:
		return statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_UNSPECIFIED
	}
}

func mapCampaignAuthorizationTargetToProto(target *campaignAuthorizationTarget) *statev1.AuthorizationTarget {
	if target == nil {
		return nil
	}
	resourceID := strings.TrimSpace(target.ResourceID)
	if resourceID == "" {
		return nil
	}
	return &statev1.AuthorizationTarget{ResourceId: resourceID}
}
