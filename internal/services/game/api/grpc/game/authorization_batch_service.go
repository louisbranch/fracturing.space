package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BatchCan evaluates whether the request actor can perform each batch item
// action/resource in campaign.
func (s *AuthorizationService) BatchCan(ctx context.Context, in *campaignv1.BatchCanRequest) (*campaignv1.BatchCanResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "batch authorization request is required")
	}
	checks := in.GetChecks()
	if len(checks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one batch authorization check is required")
	}

	results := make([]*campaignv1.BatchCanResult, 0, len(checks))
	for idx, check := range checks {
		if check == nil {
			return nil, status.Errorf(codes.InvalidArgument, "batch authorization check at index %d is required", idx)
		}

		resp, err := s.evaluator.Evaluate(ctx, &campaignv1.CanRequest{
			CampaignId: strings.TrimSpace(check.GetCampaignId()),
			Action:     check.GetAction(),
			Resource:   check.GetResource(),
			Target:     check.GetTarget(),
		})
		if err != nil {
			return nil, err
		}

		results = append(results, &campaignv1.BatchCanResult{
			CheckId:             strings.TrimSpace(check.GetCheckId()),
			Allowed:             resp.GetAllowed(),
			ReasonCode:          strings.TrimSpace(resp.GetReasonCode()),
			ActorCampaignAccess: resp.GetActorCampaignAccess(),
			ActorParticipantId:  strings.TrimSpace(resp.GetActorParticipantId()),
		})
	}

	return &campaignv1.BatchCanResponse{Results: results}, nil
}
