package game

import (
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
)

// CampaignAIService implements internal Game<=>AI/Game<=>Chat contracts.
type CampaignAIService struct {
	campaignv1.UnimplementedCampaignAIServiceServer
	stores             Stores
	clock              func() time.Time
	idGenerator        func() (string, error)
	sessionGrantConfig aisessiongrant.Config
}

// NewCampaignAIService creates a CampaignAIService with configured grant signing.
func NewCampaignAIService(stores Stores, sessionGrantConfig aisessiongrant.Config) *CampaignAIService {
	return &CampaignAIService{
		stores:             stores,
		clock:              time.Now,
		idGenerator:        id.NewID,
		sessionGrantConfig: sessionGrantConfig,
	}
}
