package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// Can evaluates whether the request actor can perform action/resource in campaign.
func (s *AuthorizationService) Can(ctx context.Context, in *campaignv1.CanRequest) (*campaignv1.CanResponse, error) {
	return s.app.Can(ctx, in)
}
