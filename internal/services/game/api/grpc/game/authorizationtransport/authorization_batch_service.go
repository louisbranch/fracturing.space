package authorizationtransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// BatchCan evaluates whether the request actor can perform each batch item
// action/resource in campaign.
func (s *AuthorizationService) BatchCan(ctx context.Context, in *campaignv1.BatchCanRequest) (*campaignv1.BatchCanResponse, error) {
	return s.app.BatchCan(ctx, in)
}
