package campaigntransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
)

const (
	defaultListCampaignsPageSize = handler.PageSmall
	maxListCampaignsPageSize     = handler.PageSmall
)

// CampaignService implements the game.v1.CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	app       campaignApplication
	readiness campaignReadinessApplication
}

// NewCampaignService creates a CampaignService. The AuthClient and AIClient
// fields in deps are optional — pass nil when the dependency is not needed
// (e.g. in tests).
func NewCampaignService(deps Deps) *CampaignService {
	return newCampaignServiceWithDependencies(deps, time.Now, id.NewID)
}

func newCampaignServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *CampaignService {
	return &CampaignService{
		app:       newCampaignApplicationWithDependencies(deps, clock, idGenerator),
		readiness: newCampaignReadinessApplication(deps),
	}
}
