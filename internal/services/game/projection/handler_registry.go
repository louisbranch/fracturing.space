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

// storeCheck maps a store requirement bit to the Applier field that satisfies
// it and a human-readable label for error messages.
type storeCheck struct {
	bit   storeRequirement
	label string
	isNil func(Applier) bool
}

// storeChecks is the single table that drives both ValidateStorePreconditions
// and validatePreconditions, eliminating duplicated nil-check logic.
var storeChecks = []storeCheck{
	{needCampaign, "campaign", func(a Applier) bool { return a.Campaign == nil }},
	{needCharacter, "character", func(a Applier) bool { return a.Character == nil }},
	{needCampaignFork, "campaign fork", func(a Applier) bool { return a.CampaignFork == nil }},
	{needInvite, "invite", func(a Applier) bool { return a.Invite == nil }},
	{needParticipant, "participant", func(a Applier) bool { return a.Participant == nil }},
	{needSession, "session", func(a Applier) bool { return a.Session == nil }},
	{needSessionGate, "session gate", func(a Applier) bool { return a.SessionGate == nil }},
	{needSessionSpotlight, "session spotlight", func(a Applier) bool { return a.SessionSpotlight == nil }},
	{needAdapters, "system adapters", func(a Applier) bool { return a.Adapters == nil }},
}

// checkMissingStores returns the labels of stores that are required by the
// given requirement bitmask but nil in the Applier.
func checkMissingStores(required storeRequirement, a Applier) []string {
	var missing []string
	for _, sc := range storeChecks {
		if required&sc.bit != 0 && sc.isNil(a) {
			missing = append(missing, sc.label)
		}
	}
	return missing
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

	if missing := checkMissingStores(required, a); len(missing) > 0 {
		return fmt.Errorf("projection stores not configured: %s", strings.Join(missing, ", "))
	}
	return nil
}

// validatePreconditions checks that the applier's stores and event envelope
// fields satisfy the handler's declared requirements.
func (a Applier) validatePreconditions(h handlerEntry, evt event.Event) error {
	if missing := checkMissingStores(h.stores, a); len(missing) > 0 {
		return fmt.Errorf("%s store is not configured", missing[0])
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
