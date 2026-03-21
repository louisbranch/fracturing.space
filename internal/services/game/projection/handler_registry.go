package projection

import (
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
	needSessionInteraction
	needScene
	needSceneCharacter
	needSceneGate
	needSceneSpotlight
	needSceneInteraction
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

// registrationRequirements keeps contributor-facing projection registration
// readable while preserving the internal bitset checks used at dispatch time.
type registrationRequirements struct {
	stores storeRequirement
	ids    idRequirement
}

type requirementOption func(*registrationRequirements)

type storeDependency uint8

const (
	storeCampaign storeDependency = iota
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
)

type envelopeField uint8

const (
	fieldCampaignID envelopeField = iota
	fieldEntityID
	fieldSessionID
)

func requirements(options ...requirementOption) registrationRequirements {
	var req registrationRequirements
	for _, option := range options {
		if option != nil {
			option(&req)
		}
	}
	return req
}

func needsStores(dependencies ...storeDependency) requirementOption {
	return func(req *registrationRequirements) {
		for _, dependency := range dependencies {
			switch dependency {
			case storeCampaign:
				req.stores |= needCampaign
			case storeCharacter:
				req.stores |= needCharacter
			case storeCampaignFork:
				req.stores |= needCampaignFork
			case storeInvite:
				req.stores |= needInvite
			case storeParticipant:
				req.stores |= needParticipant
			case storeSession:
				req.stores |= needSession
			case storeSessionGate:
				req.stores |= needSessionGate
			case storeSessionSpotlight:
				req.stores |= needSessionSpotlight
			case storeSessionInteraction:
				req.stores |= needSessionInteraction
			case storeScene:
				req.stores |= needScene
			case storeSceneCharacter:
				req.stores |= needSceneCharacter
			case storeSceneGate:
				req.stores |= needSceneGate
			case storeSceneSpotlight:
				req.stores |= needSceneSpotlight
			case storeSceneInteraction:
				req.stores |= needSceneInteraction
			case storeAdapters:
				req.stores |= needAdapters
			}
		}
	}
}

func needsEnvelope(fields ...envelopeField) requirementOption {
	return func(req *registrationRequirements) {
		for _, field := range fields {
			switch field {
			case fieldCampaignID:
				req.ids |= requireCampaignID
			case fieldEntityID:
				req.ids |= requireEntityID
			case fieldSessionID:
				req.ids |= requireSessionID
			}
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
	{needCampaign, "campaign", func(a Applier) bool { return a.Campaign == nil }},
	{needCharacter, "character", func(a Applier) bool { return a.Character == nil }},
	{needCampaignFork, "campaign fork", func(a Applier) bool { return a.CampaignFork == nil }},
	{needInvite, "invite", func(a Applier) bool { return a.Invite == nil }},
	{needParticipant, "participant", func(a Applier) bool { return a.Participant == nil }},
	{needSession, "session", func(a Applier) bool { return a.Session == nil }},
	{needSessionGate, "session gate", func(a Applier) bool { return a.SessionGate == nil }},
	{needSessionSpotlight, "session spotlight", func(a Applier) bool { return a.SessionSpotlight == nil }},
	{needSessionInteraction, "session interaction", func(a Applier) bool { return a.SessionInteraction == nil }},
	{needScene, "scene", func(a Applier) bool { return a.Scene == nil }},
	{needSceneCharacter, "scene character", func(a Applier) bool { return a.SceneCharacter == nil }},
	{needSceneGate, "scene gate", func(a Applier) bool { return a.SceneGate == nil }},
	{needSceneSpotlight, "scene spotlight", func(a Applier) bool { return a.SceneSpotlight == nil }},
	{needSceneInteraction, "scene interaction", func(a Applier) bool { return a.SceneInteraction == nil }},
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
				required |= needAdapters
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

	if h.ids&requireCampaignID != 0 && strings.TrimSpace(string(evt.CampaignID)) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if h.ids&requireEntityID != 0 && strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("entity id is required")
	}
	if h.ids&requireSessionID != 0 && strings.TrimSpace(evt.SessionID.String()) == "" {
		return fmt.Errorf("session id is required")
	}
	return nil
}
