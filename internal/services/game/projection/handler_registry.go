package projection

import (
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// idRequirement specifies which event envelope fields a handler requires.
// The field constants below double as both registration-site names and bitmask
// values, eliminating the former envelopeField→idRequirement mapping switch.
type idRequirement uint8

const (
	fieldCampaignID idRequirement = 1 << iota
	fieldEntityID
	fieldSessionID
)

// storeRequirement specifies which stores a handler depends on. Hard
// requirements are checked before dispatch; the handler will not execute
// if any required store is nil.
//
// The store constants below double as both registration-site names and bitmask
// values, eliminating the former storeDependency→storeRequirement mapping switch.
type storeRequirement uint16

const (
	storeCampaign storeRequirement = 1 << iota
	storeCharacter
	storeCampaignFork
	storeInvite
	storeParticipant
	storeSession
	storeSessionGate
	storeSessionSpotlight
	storeSessionInteraction
	storeScene
	storeSceneCharacter
	storeSceneGate
	storeSceneSpotlight
	storeSceneInteraction
	storeAdapters
	// ClaimIndex is intentionally absent — handlers that use it perform soft
	// nil checks and skip claim logic when the store is nil.
)

// handlerEntry declares preconditions for validatePreconditions. Used by the
// CoreRouter for store/ID checks before handler dispatch.
type handlerEntry struct {
	stores storeRequirement
	ids    idRequirement
}

// registrationRequirements keeps contributor-facing projection registration
// readable while preserving the internal bitset checks used at dispatch time.
type registrationRequirements struct {
	stores storeRequirement
	ids    idRequirement
}

type requirementOption func(*registrationRequirements)

func requirements(options ...requirementOption) registrationRequirements {
	var req registrationRequirements
	for _, option := range options {
		if option != nil {
			option(&req)
		}
	}
	return req
}

func needsStores(deps ...storeRequirement) requirementOption {
	return func(req *registrationRequirements) {
		for _, dep := range deps {
			req.stores |= dep
		}
	}
}

func needsEnvelope(fields ...idRequirement) requirementOption {
	return func(req *registrationRequirements) {
		for _, field := range fields {
			req.ids |= field
		}
	}
}

// coreRouter is the package-level router that dispatches core projection events.
var coreRouter = buildCoreRouter()

// buildCoreRouter constructs the core projection router with all handler
// registrations. This replaces the former handlers map with typed dispatch
// via HandleProjection/HandleProjectionRaw.
func buildCoreRouter() *CoreRouter {
	r := NewCoreRouter()
	registerCampaignProjectionHandlers(r)
	registerParticipantProjectionHandlers(r)
	registerCharacterProjectionHandlers(r)
	registerInviteProjectionHandlers(r)
	registerSessionProjectionHandlers(r)
	registerSceneProjectionHandlers(r)

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
	{storeCampaign, "campaign", func(a Applier) bool { return a.Campaign == nil }},
	{storeCharacter, "character", func(a Applier) bool { return a.Character == nil }},
	{storeCampaignFork, "campaign fork", func(a Applier) bool { return a.CampaignFork == nil }},
	{storeInvite, "invite", func(a Applier) bool { return a.Invite == nil }},
	{storeParticipant, "participant", func(a Applier) bool { return a.Participant == nil }},
	{storeSession, "session", func(a Applier) bool { return a.Session == nil }},
	{storeSessionGate, "session gate", func(a Applier) bool { return a.SessionGate == nil }},
	{storeSessionSpotlight, "session spotlight", func(a Applier) bool { return a.SessionSpotlight == nil }},
	{storeSessionInteraction, "session interaction", func(a Applier) bool { return a.SessionInteraction == nil }},
	{storeScene, "scene", func(a Applier) bool { return a.Scene == nil }},
	{storeSceneCharacter, "scene character", func(a Applier) bool { return a.SceneCharacter == nil }},
	{storeSceneGate, "scene gate", func(a Applier) bool { return a.SceneGate == nil }},
	{storeSceneSpotlight, "scene spotlight", func(a Applier) bool { return a.SceneSpotlight == nil }},
	{storeSceneInteraction, "scene interaction", func(a Applier) bool { return a.SceneInteraction == nil }},
	{storeAdapters, "system adapters", func(a Applier) bool { return a.Adapters == nil }},
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
//
// In addition to core router requirements, it checks that Adapters is present
// whenever the event registry contains system-owned event types, since those
// events are routed through the adapter path rather than the core router.
func (a Applier) ValidateStorePreconditions() error {
	// Collect the union of all store requirements across router entries.
	var required storeRequirement
	for _, h := range coreRouter.handlers {
		required |= h.stores
	}

	// System-owned events bypass the core router and route through Adapters.
	// Require Adapters when the event registry contains any system event types.
	if a.Events != nil {
		for _, def := range a.Events.ListDefinitions() {
			if def.Owner == event.OwnerSystem {
				required |= storeAdapters
				break
			}
		}
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

	if h.ids&fieldCampaignID != 0 && strings.TrimSpace(string(evt.CampaignID)) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if h.ids&fieldEntityID != 0 && strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("entity id is required")
	}
	if h.ids&fieldSessionID != 0 && strings.TrimSpace(evt.SessionID.String()) == "" {
		return fmt.Errorf("session id is required")
	}
	return nil
}
