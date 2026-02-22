package projection

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// idRequirement specifies which event envelope fields a handler requires.
type idRequirement uint8

const (
	requireCampaignID idRequirement = 1 << iota
	requireEntityID
	requireSessionID
)

// storeRequirement specifies which stores a handler depends on. Hard
// requirements are checked before dispatch; the handler will not execute
// if any required store is nil.
type storeRequirement uint16

const (
	needCampaign storeRequirement = 1 << iota
	needCharacter
	needCampaignFork
	needInvite
	needParticipant
	needSession
	needSessionGate
	needSessionSpotlight
	needAdapters
	// ClaimIndex is intentionally absent — handlers that use it perform soft
	// nil checks and skip claim logic when the store is nil.
)

// handlerEntry declares the preconditions and apply function for one event type.
type handlerEntry struct {
	stores storeRequirement
	ids    idRequirement
	apply  func(Applier, context.Context, event.Event) error
}

// handlers maps each core projection event type to its handler entry.
var handlers = map[event.Type]handlerEntry{
	// campaign
	campaign.EventTypeCreated: {
		stores: needCampaign,
		ids:    requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyCampaignCreated(ctx, evt) },
	},
	campaign.EventTypeUpdated: {
		stores: needCampaign,
		ids:    requireCampaignID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyCampaignUpdated(ctx, evt) },
	},
	campaign.EventTypeForked: {
		stores: needCampaignFork,
		ids:    requireCampaignID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyCampaignForked(ctx, evt) },
	},

	// participant
	participant.EventTypeJoined: {
		stores: needParticipant | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyParticipantJoined(ctx, evt) },
	},
	participant.EventTypeUpdated: {
		stores: needParticipant | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applyParticipantUpdated(ctx, evt)
		},
	},
	participant.EventTypeLeft: {
		stores: needParticipant | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyParticipantLeft(ctx, evt) },
	},
	participant.EventTypeBound: {
		stores: needParticipant | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyParticipantBound(ctx, evt) },
	},
	participant.EventTypeUnbound: {
		stores: needParticipant | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applyParticipantUnbound(ctx, evt)
		},
	},
	participant.EventTypeSeatReassigned: {
		stores: needParticipant | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applySeatReassigned(ctx, evt) },
	},

	// character
	character.EventTypeCreated: {
		stores: needCharacter | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyCharacterCreated(ctx, evt) },
	},
	character.EventTypeUpdated: {
		stores: needCharacter | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyCharacterUpdated(ctx, evt) },
	},
	character.EventTypeDeleted: {
		stores: needCharacter | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyCharacterDeleted(ctx, evt) },
	},
	character.EventTypeProfileUpdated: {
		stores: needAdapters,
		ids:    requireCampaignID | requireEntityID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applyCharacterProfileUpdated(ctx, evt)
		},
	},

	// invite — InviteID comes from payload with EntityID fallback for created/updated.
	invite.EventTypeCreated: {
		stores: needInvite | needCampaign,
		ids:    requireCampaignID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyInviteCreated(ctx, evt) },
	},
	invite.EventTypeClaimed: {
		stores: needInvite | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyInviteClaimed(ctx, evt) },
	},
	invite.EventTypeRevoked: {
		stores: needInvite | needCampaign,
		ids:    requireCampaignID | requireEntityID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyInviteRevoked(ctx, evt) },
	},
	invite.EventTypeUpdated: {
		stores: needInvite,
		ids:    0,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applyInviteUpdated(ctx, evt) },
	},

	// session — SessionID from payload with EntityID fallback, so EntityID
	// is not a hard envelope requirement for started/ended.
	session.EventTypeStarted: {
		stores: needSession,
		ids:    requireCampaignID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applySessionStarted(ctx, evt) },
	},
	session.EventTypeEnded: {
		stores: needSession,
		ids:    requireCampaignID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applySessionEnded(ctx, evt) },
	},
	// Gate handlers derive GateID from payload with EntityID fallback.
	session.EventTypeGateOpened: {
		stores: needSessionGate,
		ids:    requireCampaignID | requireSessionID,
		apply:  func(a Applier, ctx context.Context, evt event.Event) error { return a.applySessionGateOpened(ctx, evt) },
	},
	session.EventTypeGateResolved: {
		stores: needSessionGate,
		ids:    requireCampaignID | requireSessionID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applySessionGateResolved(ctx, evt)
		},
	},
	session.EventTypeGateAbandoned: {
		stores: needSessionGate,
		ids:    requireCampaignID | requireSessionID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applySessionGateAbandoned(ctx, evt)
		},
	},
	session.EventTypeSpotlightSet: {
		stores: needSessionSpotlight,
		ids:    requireSessionID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applySessionSpotlightSet(ctx, evt)
		},
	},
	session.EventTypeSpotlightCleared: {
		stores: needSessionSpotlight,
		ids:    requireSessionID,
		apply: func(a Applier, ctx context.Context, evt event.Event) error {
			return a.applySessionSpotlightCleared(ctx, evt)
		},
	},
}

// registeredHandlerTypes returns the sorted list of event types in the handler
// registry. Used by ProjectionHandledTypes to derive the list from the map.
func registeredHandlerTypes() []event.Type {
	types := make([]event.Type, 0, len(handlers))
	for t := range handlers {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})
	return types
}

// validatePreconditions checks that the applier's stores and event envelope
// fields satisfy the handler's declared requirements.
func (a Applier) validatePreconditions(h handlerEntry, evt event.Event) error {
	if h.stores&needCampaign != 0 && a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if h.stores&needCharacter != 0 && a.Character == nil {
		return fmt.Errorf("character store is not configured")
	}
	if h.stores&needCampaignFork != 0 && a.CampaignFork == nil {
		return fmt.Errorf("campaign fork store is not configured")
	}
	if h.stores&needInvite != 0 && a.Invite == nil {
		return fmt.Errorf("invite store is not configured")
	}
	if h.stores&needParticipant != 0 && a.Participant == nil {
		return fmt.Errorf("participant store is not configured")
	}
	if h.stores&needSession != 0 && a.Session == nil {
		return fmt.Errorf("session store is not configured")
	}
	if h.stores&needSessionGate != 0 && a.SessionGate == nil {
		return fmt.Errorf("session gate store is not configured")
	}
	if h.stores&needSessionSpotlight != 0 && a.SessionSpotlight == nil {
		return fmt.Errorf("session spotlight store is not configured")
	}
	if h.stores&needAdapters != 0 && a.Adapters == nil {
		return fmt.Errorf("system adapters are not configured")
	}

	if h.ids&requireCampaignID != 0 && strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if h.ids&requireEntityID != 0 && strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("entity id is required")
	}
	if h.ids&requireSessionID != 0 && strings.TrimSpace(evt.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	return nil
}
