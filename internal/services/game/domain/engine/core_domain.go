package engine

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// CoreDomain bundles the registration hooks that every core domain
// package exports. Adding a new core domain means creating a CoreDomain
// entry in CoreDomains() and wiring its fold function in the aggregate
// applier — the compiler and startup validators catch the rest.
type CoreDomain struct {
	name                   string
	RegisterCommands       func(*command.Registry) error
	RegisterEvents         func(*event.Registry) error
	EmittableEventTypes    func() []event.Type
	FoldHandledTypes       func() []event.Type
	DeciderHandledCommands func() []command.Type
	ProjectionHandledTypes func() []event.Type
}

// Name returns a human-readable label for error messages and diagnostics.
func (d CoreDomain) Name() string { return d.name }

// CoreDomains returns the authoritative list of core domain registrations.
// BuildRegistries iterates this slice so adding a new domain is a single
// append rather than editing 5+ locations.
func CoreDomains() []CoreDomain {
	return []CoreDomain{
		{
			name:                   "campaign",
			RegisterCommands:       campaign.RegisterCommands,
			RegisterEvents:         campaign.RegisterEvents,
			EmittableEventTypes:    campaign.EmittableEventTypes,
			FoldHandledTypes:       campaign.FoldHandledTypes,
			DeciderHandledCommands: campaign.DeciderHandledCommands,
			ProjectionHandledTypes: campaign.ProjectionHandledTypes,
		},
		{
			name:                   "action",
			RegisterCommands:       action.RegisterCommands,
			RegisterEvents:         action.RegisterEvents,
			EmittableEventTypes:    action.EmittableEventTypes,
			FoldHandledTypes:       action.FoldHandledTypes,
			DeciderHandledCommands: action.DeciderHandledCommands,
			ProjectionHandledTypes: action.ProjectionHandledTypes,
		},
		{
			name:                   "session",
			RegisterCommands:       session.RegisterCommands,
			RegisterEvents:         session.RegisterEvents,
			EmittableEventTypes:    session.EmittableEventTypes,
			FoldHandledTypes:       session.FoldHandledTypes,
			DeciderHandledCommands: session.DeciderHandledCommands,
			ProjectionHandledTypes: session.ProjectionHandledTypes,
		},
		{
			name:                   "participant",
			RegisterCommands:       participant.RegisterCommands,
			RegisterEvents:         participant.RegisterEvents,
			EmittableEventTypes:    participant.EmittableEventTypes,
			FoldHandledTypes:       participant.FoldHandledTypes,
			DeciderHandledCommands: participant.DeciderHandledCommands,
			ProjectionHandledTypes: participant.ProjectionHandledTypes,
		},
		{
			name:                   "invite",
			RegisterCommands:       invite.RegisterCommands,
			RegisterEvents:         invite.RegisterEvents,
			EmittableEventTypes:    invite.EmittableEventTypes,
			FoldHandledTypes:       invite.FoldHandledTypes,
			DeciderHandledCommands: invite.DeciderHandledCommands,
			ProjectionHandledTypes: invite.ProjectionHandledTypes,
		},
		{
			name:                   "character",
			RegisterCommands:       character.RegisterCommands,
			RegisterEvents:         character.RegisterEvents,
			EmittableEventTypes:    character.EmittableEventTypes,
			FoldHandledTypes:       character.FoldHandledTypes,
			DeciderHandledCommands: character.DeciderHandledCommands,
			ProjectionHandledTypes: character.ProjectionHandledTypes,
		},
	}
}
