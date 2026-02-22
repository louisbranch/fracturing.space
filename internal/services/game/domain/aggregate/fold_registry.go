package aggregate

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// foldEntry describes how a set of event types maps to a fold function that
// updates one slice of aggregate state. Each entry is either direct (single
// field on State) or entity-keyed (map on State keyed by EntityID).
type foldEntry struct {
	// types returns the event types handled by this fold entry.
	types func() []event.Type
	// fold applies a single event to a sub-state and writes the result back
	// into the aggregate state. Entity-keyed entries receive the EntityID from
	// the event envelope.
	fold func(state *State, evt event.Event) error
}

// coreFoldEntries returns the declarative fold dispatch table for all core
// domains. Adding a new core domain requires only adding an entry here.
func coreFoldEntries() []foldEntry {
	return []foldEntry{
		{
			types: campaign.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				updated, err := campaign.Fold(state.Campaign, evt)
				if err != nil {
					return err
				}
				state.Campaign = updated
				return nil
			},
		},
		{
			types: session.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				updated, err := session.Fold(state.Session, evt)
				if err != nil {
					return err
				}
				state.Session = updated
				return nil
			},
		},
		{
			types: action.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				updated, err := action.Fold(state.Action, evt)
				if err != nil {
					return err
				}
				state.Action = updated
				return nil
			},
		},
		{
			types: participant.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				if evt.EntityID == "" {
					return fmt.Errorf("participant fold requires EntityID but got empty for %s", evt.Type)
				}
				if state.Participants == nil {
					state.Participants = make(map[string]participant.State)
				}
				pState := state.Participants[evt.EntityID]
				updated, err := participant.Fold(pState, evt)
				if err != nil {
					return err
				}
				state.Participants[evt.EntityID] = updated
				return nil
			},
		},
		{
			types: character.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				if evt.EntityID == "" {
					return fmt.Errorf("character fold requires EntityID but got empty for %s", evt.Type)
				}
				if state.Characters == nil {
					state.Characters = make(map[string]character.State)
				}
				cState := state.Characters[evt.EntityID]
				updated, err := character.Fold(cState, evt)
				if err != nil {
					return err
				}
				state.Characters[evt.EntityID] = updated
				return nil
			},
		},
		{
			types: invite.FoldHandledTypes,
			fold: func(state *State, evt event.Event) error {
				if evt.EntityID == "" {
					return fmt.Errorf("invite fold requires EntityID but got empty for %s", evt.Type)
				}
				if state.Invites == nil {
					state.Invites = make(map[string]invite.State)
				}
				iState := state.Invites[evt.EntityID]
				updated, err := invite.Fold(iState, evt)
				if err != nil {
					return err
				}
				state.Invites[evt.EntityID] = updated
				return nil
			},
		},
	}
}
