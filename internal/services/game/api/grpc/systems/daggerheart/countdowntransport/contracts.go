package countdowntransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by countdown transport.
type CampaignStore interface {
	Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}

// SessionStore is the session-read contract consumed by countdown transport.
type SessionStore interface {
	GetSession(ctx context.Context, campaignID, sessionID string) (storage.SessionRecord, error)
}

// SessionGateStore blocks writes while a session gate is open.
type SessionGateStore interface {
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error)
}

// DaggerheartStore is the countdown projection contract consumed by countdown
// transport.
type DaggerheartStore interface {
	GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error)
	ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error)
}

// DomainCommandInput describes one Daggerheart domain command emitted by the
// countdown transport slice.
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

// CreateResult is the countdown returned after a successful create.
type CreateResult struct {
	Countdown projectionstore.DaggerheartCountdown
}

// AdvanceResult is the countdown plus the canonical advance summary returned
// after a successful advance.
type AdvanceResult struct {
	Countdown projectionstore.DaggerheartCountdown
	Summary   CountdownAdvanceSummary
}

type CountdownAdvanceSummary struct {
	BeforeRemaining int
	AfterRemaining  int
	AdvancedBy      int
	StatusBefore    string
	StatusAfter     string
	Triggered       bool
}

type TriggerResolveResult struct {
	Countdown projectionstore.DaggerheartCountdown
}

// DeleteResult is the identity returned after a successful delete.
type DeleteResult struct {
	CountdownID string
}

// Dependencies groups the exact stores and callbacks the countdown transport
// slice consumes.
type Dependencies struct {
	Campaign    CampaignStore
	Session     SessionStore
	SessionGate SessionGateStore
	Daggerheart DaggerheartStore

	NewID                func() (string, error)
	ExecuteDomainCommand func(ctx context.Context, in DomainCommandInput) error
}
