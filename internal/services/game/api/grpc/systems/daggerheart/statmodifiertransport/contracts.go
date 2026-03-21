package statmodifiertransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by stat modifier transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// SessionGateStore is the read-only gate contract used to block writes while a
// session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// DaggerheartStore is the gameplay projection contract needed by stat modifier
// transport.
type DaggerheartStore interface {
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
}

// DomainCommandInput describes one Daggerheart domain command emitted by the
// stat modifier transport slice.
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

// StatModifiersResult is the updated state returned after applying stat
// modifier mutations.
type StatModifiersResult struct {
	CharacterID     string
	ActiveModifiers []*StatModifierView
	Added           []*StatModifierView
	Removed         []*StatModifierView
}

// Dependencies groups the exact read stores and write callbacks the stat
// modifier transport slice consumes.
type Dependencies struct {
	Campaign    CampaignStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore

	ExecuteDomainCommand func(ctx context.Context, in DomainCommandInput) error
}
