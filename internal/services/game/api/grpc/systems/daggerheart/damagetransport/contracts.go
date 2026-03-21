package damagetransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by damage transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// SessionGateStore is the read-only gate contract used to block damage writes
// while a session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// DaggerheartStore is the system-owned gameplay projection contract needed by
// damage transport.
type DaggerheartStore interface {
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error)
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
	ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error)
}

// ContentStore loads catalog-backed adversary entries for recurring-rule
// automation during damage application.
type ContentStore interface {
	GetDaggerheartAdversaryEntry(ctx context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error)
	GetDaggerheartArmor(ctx context.Context, id string) (contentstore.DaggerheartArmor, error)
}

// EventStore is the event-read contract used to validate roll-seq references.
type EventStore interface {
	GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error)
}

// SystemCommandInput describes one Daggerheart system command emitted by the
// damage transport slice.
type SystemCommandInput struct {
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

// CharacterDamageResult is the read-model state returned after applying
// character damage.
type CharacterDamageResult struct {
	CharacterID string
	State       projectionstore.DaggerheartCharacterState
}

// AdversaryDamageResult is the read-model state returned after applying
// adversary damage.
type AdversaryDamageResult struct {
	AdversaryID string
	Adversary   projectionstore.DaggerheartAdversary
}

// Dependencies groups the exact read stores and write callbacks the damage
// transport slice consumes.
type Dependencies struct {
	Campaign    CampaignStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore
	Content     ContentStore
	Event       EventStore

	SeedFunc func() (int64, error)

	ExecuteSystemCommand    func(ctx context.Context, in SystemCommandInput) error
	LoadAdversaryForSession func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
}
