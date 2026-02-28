package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

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
		CheckID:    "",
		Evaluated:  true,
		Allowed:    resp.GetAllowed(),
		ReasonCode: strings.TrimSpace(resp.GetReasonCode()),
	}, nil
}

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
			CheckID:    checkID,
			Evaluated:  true,
			Allowed:    result.GetAllowed(),
			ReasonCode: strings.TrimSpace(result.GetReasonCode()),
		})
	}

	return decisions, nil
}

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

func mapCampaignAuthorizationResourceToProto(resource campaignapp.AuthorizationResource) statev1.AuthorizationResource {
	switch campaignapp.AuthorizationResource(strings.TrimSpace(string(resource))) {
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

func mapCampaignAuthorizationTargetToProto(target *campaignapp.AuthorizationTarget) *statev1.AuthorizationTarget {
	if target == nil {
		return nil
	}
	resourceID := strings.TrimSpace(target.ResourceID)
	if resourceID == "" {
		return nil
	}
	return &statev1.AuthorizationTarget{ResourceId: resourceID}
}
