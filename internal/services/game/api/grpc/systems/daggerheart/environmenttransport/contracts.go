package environmenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

type CampaignStore = daggerheartguard.CampaignStore

type SessionStore = daggerheartguard.SessionStore

type SessionGateStore = daggerheartguard.SessionGateStore

type DaggerheartStore interface {
	GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error)
	ListDaggerheartEnvironmentEntities(ctx context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error)
}

type ContentStore interface {
	GetDaggerheartEnvironment(ctx context.Context, id string) (contentstore.DaggerheartEnvironment, error)
}

type DomainCommandInput = workflowwrite.DomainCommandInput

type Dependencies struct {
	Campaign CampaignStore
	Session  SessionStore
	Gate     SessionGateStore

	Daggerheart DaggerheartStore
	Content     ContentStore

	GenerateID           func() (string, error)
	ExecuteDomainCommand func(ctx context.Context, in DomainCommandInput) error
}

type Handler struct {
	deps Dependencies
}

func EnvironmentEntityToProto(environmentEntity projectionstore.DaggerheartEnvironmentEntity) *pb.DaggerheartEnvironmentEntity {
	return environmentEntityToProto(environmentEntity)
}
