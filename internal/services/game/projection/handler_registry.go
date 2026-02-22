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
	campaign.EventTypeCreated: {stores: needCampaign, ids: requireEntityID, apply: Applier.applyCampaignCreated},
	campaign.EventTypeUpdated: {stores: needCampaign, ids: requireCampaignID, apply: Applier.applyCampaignUpdated},
	campaign.EventTypeForked:  {stores: needCampaignFork, ids: requireCampaignID, apply: Applier.applyCampaignForked},

	// participant
	participant.EventTypeJoined:         {stores: needParticipant | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyParticipantJoined},
	participant.EventTypeUpdated:        {stores: needParticipant | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyParticipantUpdated},
	participant.EventTypeLeft:           {stores: needParticipant | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyParticipantLeft},
	participant.EventTypeBound:          {stores: needParticipant | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyParticipantBound},
	participant.EventTypeUnbound:        {stores: needParticipant | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyParticipantUnbound},
	participant.EventTypeSeatReassigned: {stores: needParticipant | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applySeatReassigned},

	// character
	character.EventTypeCreated:        {stores: needCharacter | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyCharacterCreated},
	character.EventTypeUpdated:        {stores: needCharacter | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyCharacterUpdated},
	character.EventTypeDeleted:        {stores: needCharacter | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyCharacterDeleted},
	character.EventTypeProfileUpdated: {stores: needAdapters, ids: requireCampaignID | requireEntityID, apply: Applier.applyCharacterProfileUpdated},

	// invite — InviteID comes from payload with EntityID fallback for created/updated.
	invite.EventTypeCreated: {stores: needInvite | needCampaign, ids: requireCampaignID, apply: Applier.applyInviteCreated},
	invite.EventTypeClaimed: {stores: needInvite | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyInviteClaimed},
	invite.EventTypeRevoked: {stores: needInvite | needCampaign, ids: requireCampaignID | requireEntityID, apply: Applier.applyInviteRevoked},
	invite.EventTypeUpdated: {stores: needInvite, ids: 0, apply: Applier.applyInviteUpdated},

	// session — SessionID from payload with EntityID fallback, so EntityID
	// is not a hard envelope requirement for started/ended.
	session.EventTypeStarted: {stores: needSession, ids: requireCampaignID, apply: Applier.applySessionStarted},
	session.EventTypeEnded:   {stores: needSession, ids: requireCampaignID, apply: Applier.applySessionEnded},
	// Gate handlers derive GateID from payload with EntityID fallback.
	session.EventTypeGateOpened:       {stores: needSessionGate, ids: requireCampaignID | requireSessionID, apply: Applier.applySessionGateOpened},
	session.EventTypeGateResolved:     {stores: needSessionGate, ids: requireCampaignID | requireSessionID, apply: Applier.applySessionGateResolved},
	session.EventTypeGateAbandoned:    {stores: needSessionGate, ids: requireCampaignID | requireSessionID, apply: Applier.applySessionGateAbandoned},
	session.EventTypeSpotlightSet:     {stores: needSessionSpotlight, ids: requireSessionID, apply: Applier.applySessionSpotlightSet},
	session.EventTypeSpotlightCleared: {stores: needSessionSpotlight, ids: requireSessionID, apply: Applier.applySessionSpotlightCleared},
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

// ValidateStorePreconditions verifies that every store dependency declared in
// the handler registry is satisfied by this Applier. Call at startup to fail
// fast on misconfiguration instead of discovering nil stores at runtime when the
// first event of a given type arrives.
func (a Applier) ValidateStorePreconditions() error {
	// Collect the union of all store requirements across handlers.
	var required storeRequirement
	for _, h := range handlers {
		required |= h.stores
	}

	var missing []string
	if required&needCampaign != 0 && a.Campaign == nil {
		missing = append(missing, "campaign")
	}
	if required&needCharacter != 0 && a.Character == nil {
		missing = append(missing, "character")
	}
	if required&needCampaignFork != 0 && a.CampaignFork == nil {
		missing = append(missing, "campaign fork")
	}
	if required&needInvite != 0 && a.Invite == nil {
		missing = append(missing, "invite")
	}
	if required&needParticipant != 0 && a.Participant == nil {
		missing = append(missing, "participant")
	}
	if required&needSession != 0 && a.Session == nil {
		missing = append(missing, "session")
	}
	if required&needSessionGate != 0 && a.SessionGate == nil {
		missing = append(missing, "session gate")
	}
	if required&needSessionSpotlight != 0 && a.SessionSpotlight == nil {
		missing = append(missing, "session spotlight")
	}
	if required&needAdapters != 0 && a.Adapters == nil {
		missing = append(missing, "system adapters")
	}
	if len(missing) > 0 {
		return fmt.Errorf("projection stores not configured: %s", strings.Join(missing, ", "))
	}
	return nil
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
