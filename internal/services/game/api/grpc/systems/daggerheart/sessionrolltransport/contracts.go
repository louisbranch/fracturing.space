package sessionrolltransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

type SessionStore interface {
	GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error)
}

type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

type DaggerheartStore interface {
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
}

type EventStore interface {
	GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error)
}

type RollResolveInput struct {
	CampaignID      string
	SessionID       string
	SceneID         string
	RequestID       string
	InvocationID    string
	EntityType      string
	EntityID        string
	PayloadJSON     []byte
	MissingEventMsg string
}

type HopeSpendInput struct {
	CampaignID   string
	SessionID    string
	SceneID      string
	RequestID    string
	InvocationID string
	CharacterID  string
	Source       string
	Amount       int
	HopeBefore   int
	HopeAfter    int
	RollSeq      uint64
}

type Dependencies struct {
	Campaign    CampaignStore
	Session     SessionStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore
	Event       EventStore

	SeedFunc func() (int64, error)

	ExecuteActionRollResolve    func(ctx context.Context, in RollResolveInput) (uint64, error)
	ExecuteDamageRollResolve    func(ctx context.Context, in RollResolveInput) (uint64, error)
	ExecuteAdversaryRollResolve func(ctx context.Context, in RollResolveInput) (uint64, error)
	ExecuteHopeSpend            func(ctx context.Context, in HopeSpendInput) error

	AdvanceBreathCountdown  func(ctx context.Context, campaignID, sessionID, countdownID string, failed bool) error
	LoadAdversaryForSession func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
}
