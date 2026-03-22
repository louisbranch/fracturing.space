package recoverytransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by recovery transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// SessionGateStore blocks writes while a session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// DaggerheartStore is the gameplay projection contract consumed by recovery
// transport.
type DaggerheartStore interface {
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error)
	GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error)
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error)
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
}

// SystemCommandInput describes one Daggerheart system command emitted by the
// recovery transport slice.
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

// StressConditionInput describes one stress/vulnerable repair callback request.
type StressConditionInput struct {
	CampaignID    string
	SessionID     string
	CharacterID   string
	Conditions    []projectionstore.DaggerheartConditionState
	StressBefore  int
	StressAfter   int
	StressMax     int
	RollSeq       *uint64
	RequestID     string
	CorrelationID string
}

// CharacterDeleteInput describes one character deletion callback request.
type CharacterDeleteInput struct {
	CampaignID  string
	CharacterID string
	Reason      string
}

// RestResult is the canonical output for ApplyRest.
type RestResult struct {
	Snapshot                  projectionstore.DaggerheartSnapshot
	CharacterStates           []CharacterStateEntry
	Countdowns                []projectionstore.DaggerheartCountdown
	CampaignCountdownAdvances []daggerheartpayload.CampaignCountdownAdvancePayload
}

// CharacterStateEntry couples a character ID with its updated gameplay state.
type CharacterStateEntry struct {
	CharacterID string
	State       projectionstore.DaggerheartCharacterState
}

// CharacterStateResult is the canonical output for recovery mutations that
// return one updated character state.
type CharacterStateResult struct {
	CharacterID string
	State       projectionstore.DaggerheartCharacterState
}

// DeathMoveResult is the canonical output for ApplyDeathMove.
type DeathMoveResult struct {
	CharacterID string
	State       projectionstore.DaggerheartCharacterState
	Outcome     DeathOutcome
}

// DeathOutcome captures the transport-visible death move result details.
type DeathOutcome struct {
	Move          string
	LifeState     string
	HopeDie       *int
	FearDie       *int
	HPCleared     int
	StressCleared int
	ScarGained    bool
}

// BlazeResult is the canonical output for ResolveBlazeOfGlory.
type BlazeResult struct {
	CharacterID string
	State       projectionstore.DaggerheartCharacterState
	LifeState   string
}

// Dependencies groups the exact reads and callbacks consumed by the recovery
// transport slice.
type Dependencies struct {
	Campaign    CampaignStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore

	SeedGenerator               func() (int64, error)
	ExecuteSystemCommand        func(ctx context.Context, in SystemCommandInput) error
	ApplyStressConditionChange  func(ctx context.Context, in StressConditionInput) error
	AppendCharacterDeletedEvent func(ctx context.Context, in CharacterDeleteInput) error

	ResolveSeed func(rng *commonv1.RngRequest, seedFunc func() (int64, error), replayMode func(commonv1.RollMode) bool) (seed int64, source string, mode commonv1.RollMode, err error)
}
