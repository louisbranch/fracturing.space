package authorizationtransport

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// AuthorizationService implements the game.v1.AuthorizationService gRPC API.
type AuthorizationService struct {
	campaignv1.UnimplementedAuthorizationServiceServer
	app authorizationApplication
}

// NewService creates an AuthorizationService with default dependencies.
func NewService(deps Deps) *AuthorizationService {
	return &AuthorizationService{
		app: newAuthorizationApplication(deps),
	}
}
