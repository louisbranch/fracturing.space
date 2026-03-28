package sessionrolltransport

import (
	"context"

	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

type CampaignStore = daggerheartguard.CampaignStore

type SessionStore = daggerheartguard.SessionStore

type SessionGateStore = daggerheartguard.SessionGateStore

type DaggerheartStore interface {
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error)
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
}

type EventStore interface {
	GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error)
}

type ContentStore interface {
	GetDaggerheartArmor(ctx context.Context, id string) (contentstore.DaggerheartArmor, error)
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

type ArmorBackedHopeSpendInput struct {
	CampaignID   string
	SessionID    string
	SceneID      string
	RequestID    string
	InvocationID string
	CharacterID  string
	Source       string
	ArmorBefore  int
	ArmorAfter   int
}

type AdversaryFeatureApplyInput struct {
	CampaignID              string
	SessionID               string
	SceneID                 string
	RequestID               string
	InvocationID            string
	Adversary               projectionstore.DaggerheartAdversary
	FeatureID               string
	PendingExperienceBefore *projectionstore.DaggerheartAdversaryPendingExperience
	PendingExperienceAfter  *projectionstore.DaggerheartAdversaryPendingExperience
}

type Dependencies struct {
	Campaign    CampaignStore
	Session     SessionStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore
	Content     ContentStore
	Event       EventStore

	SeedFunc func() (int64, error)

	ExecuteActionRollResolve     func(ctx context.Context, in RollResolveInput) (uint64, error)
	ExecuteDamageRollResolve     func(ctx context.Context, in RollResolveInput) (uint64, error)
	ExecuteAdversaryRollResolve  func(ctx context.Context, in RollResolveInput) (uint64, error)
	ExecuteHopeSpend             func(ctx context.Context, in HopeSpendInput) error
	ExecuteArmorBackedHopeSpend  func(ctx context.Context, in ArmorBackedHopeSpendInput) error
	ExecuteAdversaryFeatureApply func(ctx context.Context, in AdversaryFeatureApplyInput) error

	AdvanceBreathCountdown  func(ctx context.Context, campaignID, sessionID, countdownID string, failed bool) error
	LoadAdversaryForSession func(ctx context.Context, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error)
}
