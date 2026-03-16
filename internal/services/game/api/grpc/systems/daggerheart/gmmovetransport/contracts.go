package gmmovetransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gmconsequence"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by GM-move transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// SessionStore is the session-read contract consumed by GM-move transport.
type SessionStore interface {
	GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error)
}

// SessionGateStore blocks writes while a session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// SessionSpotlightStore reads the current session spotlight state.
type SessionSpotlightStore interface {
	GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error)
}

// DaggerheartStore is the gameplay projection contract needed by GM-move
// transport.
type DaggerheartStore interface {
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error)
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
	GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error)
}

// ContentStore is the subset of Daggerheart catalog content needed by typed
// GM Fear spend validation.
type ContentStore interface {
	GetDaggerheartAdversaryEntry(ctx context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error)
	GetDaggerheartEnvironment(ctx context.Context, id string) (contentstore.DaggerheartEnvironment, error)
}

// DomainCommandInput describes one Daggerheart domain command emitted by the
// GM-move transport slice.
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

// Result is the GM-fear state returned after applying a GM move.
type Result struct {
	CampaignID   string
	GMFearBefore int
	GMFearAfter  int
}

// Dependencies groups the exact read stores and write callbacks the GM-move
// transport slice consumes.
type Dependencies struct {
	Campaign         CampaignStore
	Session          SessionStore
	SessionGate      SessionGateStore
	SessionSpotlight SessionSpotlightStore
	Daggerheart      DaggerheartStore
	Content          ContentStore

	ExecuteDomainCommand func(ctx context.Context, in DomainCommandInput) error
	ExecuteCoreCommand   func(ctx context.Context, in gmconsequence.CoreCommandInput) error
}
