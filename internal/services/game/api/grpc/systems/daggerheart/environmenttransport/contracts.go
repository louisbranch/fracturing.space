package environmenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type CampaignStore interface {
	Get(ctx context.Context, campaignID string) (storage.CampaignRecord, error)
}

type SessionStore interface {
	GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error)
}

type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

type DaggerheartStore interface {
	GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error)
	ListDaggerheartEnvironmentEntities(ctx context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error)
}

type ContentStore interface {
	GetDaggerheartEnvironment(ctx context.Context, id string) (contentstore.DaggerheartEnvironment, error)
}

type DomainCommandInput struct {
	CampaignID      string
	CommandType     command.Type
	SessionID       string
	SceneID         string
	RequestID       string
	InvocationID    string
	EntityType      string
	EntityID        string
	PayloadJSON     []byte
	MissingEventMsg string
	ApplyErrMessage string
}

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
