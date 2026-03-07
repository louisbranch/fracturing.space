package game

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

const (
	defaultListCampaignsPageSize = 10
	maxListCampaignsPageSize     = 10
)

// CampaignService implements the game.v1.CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
	aiClient    aiv1.AgentServiceClient
}

// NewCampaignService creates a CampaignService. The authClient and aiClient
// are optional — pass nil when the dependency is not needed (e.g. in tests).
func NewCampaignService(stores Stores, authClient authv1.AuthServiceClient, aiClient aiv1.AgentServiceClient) *CampaignService {
	return &CampaignService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
		authClient:  authClient,
		aiClient:    aiClient,
	}
}
