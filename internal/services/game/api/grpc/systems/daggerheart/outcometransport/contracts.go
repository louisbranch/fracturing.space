package outcometransport

import (
	"context"

	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is the campaign-read contract consumed by the outcome
// transport slice.
type CampaignStore = daggerheartguard.CampaignStore

// SessionStore is the session-read contract consumed by the outcome transport
// slice.
type SessionStore = daggerheartguard.SessionStore

// DaggerheartStore is the system-owned gameplay projection contract needed by
// outcome application.
type DaggerheartStore interface {
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error)
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error)
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error)
}

// ContentStore provides subclass catalog reads needed for derived subclass
// runtime effects.
type ContentStore interface {
	GetDaggerheartSubclass(ctx context.Context, id string) (contentstore.DaggerheartSubclass, error)
}

// EventStore is the event-read contract consumed by outcome validation and
// idempotency checks.
type EventStore interface {
	GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error)
	ListEventsPage(ctx context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error)
}

// SessionGateStore is the read-only gate contract needed to block writes when
// a session gate is already open.
type SessionGateStore = daggerheartguard.SessionGateStore

// SessionSpotlightStore is the read-only spotlight contract needed when a GM
// consequence may have to repair spotlight state.
type SessionSpotlightStore interface {
	GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error)
}

// SystemCommandInput is an alias for the shared workflow runtime type, kept for
// local readability inside the outcome transport slice.
type SystemCommandInput = workflowruntime.SystemCommandInput

// CoreCommandInput describes one core command emitted by the outcome transport
// slice.
type CoreCommandInput = workflowwrite.CoreCommandInput

// ApplyStressVulnerableConditionInput groups the arguments needed to reuse the
// existing stress/vulnerable write helper that still lives in the root package.
type ApplyStressVulnerableConditionInput struct {
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

// Dependencies groups the exact read stores and write callbacks the outcome
// transport slice consumes.
type Dependencies struct {
	Campaign         CampaignStore
	Session          SessionStore
	SessionGate      SessionGateStore
	SessionSpotlight SessionSpotlightStore
	Daggerheart      DaggerheartStore
	Content          ContentStore
	Event            EventStore

	ExecuteSystemCommand func(ctx context.Context, in SystemCommandInput) error
	ExecuteCoreCommand   func(ctx context.Context, in CoreCommandInput) error

	ApplyStressVulnerableCondition func(ctx context.Context, in ApplyStressVulnerableConditionInput) error
}
