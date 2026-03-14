package game

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// AuthorizationService implements the game.v1.AuthorizationService gRPC API.
type AuthorizationService struct {
	campaignv1.UnimplementedAuthorizationServiceServer
	app authorizationApplication
}

// NewAuthorizationService creates an AuthorizationService with default dependencies.
func NewAuthorizationService(stores Stores) *AuthorizationService {
	return &AuthorizationService{
		app: newAuthorizationApplication(stores),
	}
}
