package gmmovetransport

import (
	"context"

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

// DaggerheartStore is the gameplay projection contract needed by GM-move
// transport.
type DaggerheartStore interface {
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error)
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
	Campaign    CampaignStore
	Session     SessionStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore

	ExecuteDomainCommand func(ctx context.Context, in DomainCommandInput) error
}
