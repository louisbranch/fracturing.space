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
	storeParticipant
	storeSession
	storeSessionRecap
	storeSessionGate
	storeSessionSpotlight
	storeSessionInteraction
	storeScene
	storeSceneCharacter
	storeSceneGate
	storeSceneSpotlight
	storeSceneInteraction
	storeSceneGMInteraction
	storeAdapters
	// ClaimIndex is intentionally absent — handlers that use it perform soft
	// nil checks and skip claim logic when the store is nil.
)

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

// storeChecks is the single table that drives both startup validation and
// per-event dispatch preconditions, eliminating duplicated nil-check logic.
var storeChecks = []storeCheck{
	{storeCampaign, "campaign", func(a Applier) bool { return a.Campaign == nil }},
	{storeCharacter, "character", func(a Applier) bool { return a.Character == nil }},
	{storeCampaignFork, "campaign fork", func(a Applier) bool { return a.CampaignFork == nil }},
	{storeParticipant, "participant", func(a Applier) bool { return a.Participant == nil }},
	{storeSession, "session", func(a Applier) bool { return a.Session == nil }},
	{storeSessionRecap, "session recap", func(a Applier) bool { return a.SessionRecap == nil }},
	{storeSessionGate, "session gate", func(a Applier) bool { return a.SessionGate == nil }},
	{storeSessionSpotlight, "session spotlight", func(a Applier) bool { return a.SessionSpotlight == nil }},
	{storeSessionInteraction, "session interaction", func(a Applier) bool { return a.SessionInteraction == nil }},
	{storeScene, "scene", func(a Applier) bool { return a.Scene == nil }},
	{storeSceneCharacter, "scene character", func(a Applier) bool { return a.SceneCharacter == nil }},
	{storeSceneGate, "scene gate", func(a Applier) bool { return a.SceneGate == nil }},
	{storeSceneSpotlight, "scene spotlight", func(a Applier) bool { return a.SceneSpotlight == nil }},
	{storeSceneInteraction, "scene interaction", func(a Applier) bool { return a.SceneInteraction == nil }},
	{storeSceneGMInteraction, "scene gm interaction", func(a Applier) bool { return a.SceneGMInteraction == nil }},
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

func coreRequiredStores() storeRequirement {
	var required storeRequirement
	for _, h := range coreRouter.handlers {
		required |= h.stores
	}
	return required
}

func requiresSystemAdapters(events *event.Registry) bool {
	if events == nil {
		return false
	}
	for _, def := range events.ListDefinitions() {
		if def.Owner == event.OwnerSystem {
			return true
		}
	}
	return false
}

// ValidateCoreStorePreconditions verifies that every store dependency declared
// by core projection handlers is satisfied by this Applier.
func (a Applier) ValidateCoreStorePreconditions() error {
	if missing := checkMissingStores(coreRequiredStores(), a); len(missing) > 0 {
		return fmt.Errorf("projection stores not configured: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateRuntimePreconditions verifies that the applier is fully configured
// for runtime projection work.
//
// This checks both core projection-handler stores and the system adapter
// registry required to apply system-owned events declared in the event
// registry.
func (a Applier) ValidateRuntimePreconditions() error {
	if err := a.ValidateCoreStorePreconditions(); err != nil {
		return err
	}
	if requiresSystemAdapters(a.Events) && a.Adapters == nil {
		return fmt.Errorf("projection system adapters are not configured")
	}
	return nil
}

// validateHandlerPreconditions checks that the applier's stores and event
// envelope fields satisfy one handler's declared requirements.
func (a Applier) validateHandlerPreconditions(stores storeRequirement, ids idRequirement, evt event.Event) error {
	if missing := checkMissingStores(stores, a); len(missing) > 0 {
		return fmt.Errorf("%s store is not configured", missing[0])
	}

	if ids&fieldCampaignID != 0 && strings.TrimSpace(string(evt.CampaignID)) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if ids&fieldEntityID != 0 && strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("entity id is required")
	}
	if ids&fieldSessionID != 0 && strings.TrimSpace(evt.SessionID.String()) == "" {
		return fmt.Errorf("session id is required")
	}
	return nil
}
