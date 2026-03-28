package workflowruntime

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// EventStore is the event-read contract consumed by shared replay checks.
type EventStore interface {
	ListEventsPage(ctx context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error)
}

// ExecuteDomainCommandFunc runs one domain command with the provided applier
// and write options.
type ExecuteDomainCommandFunc func(ctx context.Context, cmd command.Command, applier domainwrite.EventApplier, options domainwrite.Options) error

// ReplayCheckInput groups the replay-check arguments used by workflow support
// callers.
type ReplayCheckInput struct {
	CampaignID string
	SessionID  string
	RollSeq    uint64
	RequestID  string
	EventType  event.Type
	EntityID   string
}

// CommandInput is the unified struct for all Daggerheart command inputs that
// flow through the shared workflow write and runtime paths. Domain, core, and
// system callers use the same shape; fields that do not apply to a particular
// call site are left at their zero value.
type CommandInput struct {
	CampaignID      string
	CommandType     command.Type
	SessionID       string
	SceneID         string
	RequestID       string
	InvocationID    string
	CorrelationID   string
	EntityType      string
	EntityID        string
	PayloadJSON     []byte
	MissingEventMsg string
	ApplyErrMessage string
}

// SystemCommandInput is a type alias kept for call-site compatibility.
type SystemCommandInput = CommandInput

// Dependencies groups the exact collaborators needed by the shared workflow
// runtime support.
type Dependencies struct {
	Event                EventStore
	Daggerheart          projectionstore.Store
	ExecuteDomainCommand ExecuteDomainCommandFunc
}
