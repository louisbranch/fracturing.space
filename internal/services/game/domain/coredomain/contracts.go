package coredomain

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Contracts describes the non-aggregate registration surface a core domain
// package exports to engine bootstrap and validation.
//
// Aggregate replay composes these contracts with aggregate-owned fold adapters
// so the built-in inventory can be authored from package-owned descriptors
// without making domain packages depend on aggregate state types.
type Contracts struct {
	DomainName             string
	RegisterCommands       func(*command.Registry) error
	RegisterEvents         func(*event.Registry) error
	EmittableEventTypes    func() []event.Type
	FoldHandledTypes       func() []event.Type
	DeciderHandledCommands func() []command.Type
	ProjectionHandledTypes func() []event.Type
	RejectionCodes         func() []string
}
