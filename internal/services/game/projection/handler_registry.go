package projection

import (
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

// handlerEntry declares preconditions for validatePreconditions. Used by the
// CoreRouter for store/ID checks before handler dispatch.
type handlerEntry struct {
	stores storeRequirement
	ids    idRequirement
}

// coreRouter is the package-level router that dispatches core projection events.
var coreRouter = buildCoreRouter()

// buildCoreRouter constructs the core projection router with all handler
// registrations. This replaces the former handlers map with typed dispatch
// via HandleProjection/HandleProjectionRaw.
func buildCoreRouter() *CoreRouter {
	r := NewCoreRouter()

	// campaign
	HandleProjection(r, campaign.EventTypeCreated, needCampaign, requireEntityID, Applier.applyCampaignCreated)
	HandleProjection(r, campaign.EventTypeUpdated, needCampaign, requireCampaignID, Applier.applyCampaignUpdated)
	HandleProjection(r, campaign.EventTypeForked, needCampaignFork, requireCampaignID, Applier.applyCampaignForked)

	// participant
	HandleProjection(r, participant.EventTypeJoined, needParticipant|needCampaign, requireCampaignID|requireEntityID, Applier.applyParticipantJoined)
	HandleProjection(r, participant.EventTypeUpdated, needParticipant|needCampaign, requireCampaignID|requireEntityID, Applier.applyParticipantUpdated)
	HandleProjectionRaw(r, participant.EventTypeLeft, needParticipant|needCampaign, requireCampaignID|requireEntityID, Applier.applyParticipantLeft)
	HandleProjection(r, participant.EventTypeBound, needParticipant|needCampaign, requireCampaignID|requireEntityID, Applier.applyParticipantBound)
	HandleProjection(r, participant.EventTypeUnbound, needParticipant|needCampaign, requireCampaignID|requireEntityID, Applier.applyParticipantUnbound)
	HandleProjection(r, participant.EventTypeSeatReassigned, needParticipant|needCampaign, requireCampaignID|requireEntityID, Applier.applySeatReassigned)

	// character
	HandleProjection(r, character.EventTypeCreated, needCharacter|needCampaign, requireCampaignID|requireEntityID, Applier.applyCharacterCreated)
	HandleProjection(r, character.EventTypeUpdated, needCharacter|needCampaign, requireCampaignID|requireEntityID, Applier.applyCharacterUpdated)
	HandleProjection(r, character.EventTypeDeleted, needCharacter|needCampaign, requireCampaignID|requireEntityID, Applier.applyCharacterDeleted)
	HandleProjection(r, character.EventTypeProfileUpdated, needAdapters, requireCampaignID|requireEntityID, Applier.applyCharacterProfileUpdated)

	// invite — InviteID comes from payload with EntityID fallback for created/updated.
	HandleProjection(r, invite.EventTypeCreated, needInvite|needCampaign, requireCampaignID, Applier.applyInviteCreated)
	HandleProjection(r, invite.EventTypeClaimed, needInvite|needCampaign, requireCampaignID|requireEntityID, Applier.applyInviteClaimed)
	HandleProjection(r, invite.EventTypeRevoked, needInvite|needCampaign, requireCampaignID|requireEntityID, Applier.applyInviteRevoked)
	HandleProjection(r, invite.EventTypeUpdated, needInvite, 0, Applier.applyInviteUpdated)

	// session — SessionID from payload with EntityID fallback, so EntityID
	// is not a hard envelope requirement for started/ended.
	HandleProjection(r, session.EventTypeStarted, needSession, requireCampaignID, Applier.applySessionStarted)
	HandleProjection(r, session.EventTypeEnded, needSession, requireCampaignID, Applier.applySessionEnded)
	// Gate handlers derive GateID from payload with EntityID fallback.
	HandleProjection(r, session.EventTypeGateOpened, needSessionGate, requireCampaignID|requireSessionID, Applier.applySessionGateOpened)
	HandleProjection(r, session.EventTypeGateResolved, needSessionGate, requireCampaignID|requireSessionID, Applier.applySessionGateResolved)
	HandleProjection(r, session.EventTypeGateAbandoned, needSessionGate, requireCampaignID|requireSessionID, Applier.applySessionGateAbandoned)
	HandleProjection(r, session.EventTypeSpotlightSet, needSessionSpotlight, requireSessionID, Applier.applySessionSpotlightSet)
	HandleProjectionRaw(r, session.EventTypeSpotlightCleared, needSessionSpotlight, requireSessionID, Applier.applySessionSpotlightCleared)

	return r
}

// registeredHandlerTypes returns the sorted list of event types in the handler
// registry. Used by ProjectionHandledTypes to derive the list from the router.
func registeredHandlerTypes() []event.Type {
	types := coreRouter.HandledTypes()
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
	// Collect the union of all store requirements across router entries.
	var required storeRequirement
	for _, h := range coreRouter.handlers {
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
