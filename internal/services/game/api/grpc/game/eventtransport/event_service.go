package eventtransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// Service implements the game.v1.EventService gRPC API.
type Service struct {
	campaignv1.UnimplementedEventServiceServer
	app eventApplication
}

// NewService creates an event Service with the provided dependencies.
func NewService(deps Deps) *Service {
	return &Service{
		app: newEventApplicationWithDependencies(deps, time.Now),
	}
}
