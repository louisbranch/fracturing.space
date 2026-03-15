package campaigntransport

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// CampaignAIService implements internal Game<=>AI authorization contracts.
type CampaignAIService struct {
	campaignv1.UnimplementedCampaignAIServiceServer
	app campaignAIApplication
}

// NewCampaignAIService creates a CampaignAIService with configured grant signing.
func NewCampaignAIService(deps Deps) *CampaignAIService {
	return newCampaignAIServiceWithDependencies(deps, time.Now, id.NewID)
}

func newCampaignAIServiceWithDependencies(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *CampaignAIService {
	return &CampaignAIService{
		app: newCampaignAIApplicationWithDependencies(deps, clock, idGenerator),
	}
}
