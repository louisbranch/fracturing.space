package statmodifiertransport

import (
	"context"

	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// CampaignStore is the campaign-read contract consumed by stat modifier transport.
type CampaignStore = daggerheartguard.CampaignStore

// SessionGateStore is the read-only gate contract used to block writes while a
// session gate is open.
type SessionGateStore = daggerheartguard.SessionGateStore

// DaggerheartStore is the gameplay projection contract needed by stat modifier
// transport.
type DaggerheartStore interface {
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
}

// DomainCommandInput describes one Daggerheart domain command emitted by the
// stat modifier transport slice.
type DomainCommandInput = workflowwrite.DomainCommandInput

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
