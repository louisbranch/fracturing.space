package aitransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// Service implements the game.v1.CampaignAIService gRPC API for internal
// Game<=>AI authorization contracts.
type Service struct {
	campaignv1.UnimplementedCampaignAIServiceServer
	app application
}

// NewService creates a Service with configured grant signing.
func NewService(deps Deps) *Service {
	return newServiceWithDependencies(deps, time.Now, id.NewID)
}

func newServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *Service {
	return &Service{
		app: newApplicationWithDependencies(deps, clock, idGenerator),
	}
}
