package conditiontransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by condition transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// SessionGateStore is the read-only gate contract used to block writes while a
// session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// DaggerheartStore is the gameplay projection contract needed by condition
// transport.
type DaggerheartStore interface {
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
}

// EventStore is the event-read contract used to validate roll-seq references.
type EventStore interface {
	GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error)
}

// DomainCommandInput describes one Daggerheart domain command emitted by the
// condition transport slice.
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

// CharacterConditionsResult is the updated state returned after applying
// character condition or life-state mutations.
type CharacterConditionsResult struct {
	CharacterID string
	State       projectionstore.DaggerheartCharacterState
	Added       []string
	Removed     []string
}

// AdversaryConditionsResult is the updated state returned after applying
// adversary condition mutations.
type AdversaryConditionsResult struct {
	AdversaryID string
	Adversary   projectionstore.DaggerheartAdversary
	Added       []string
	Removed     []string
}

// Dependencies groups the exact read stores and write callbacks the condition
// transport slice consumes.
type Dependencies struct {
	Campaign    CampaignStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore
	Event       EventStore

	ExecuteDomainCommand    func(ctx context.Context, in DomainCommandInput) error
	LoadAdversaryForSession func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
}
