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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
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
	needScene
	needSceneCharacter
	needSceneGate
	needSceneSpotlight
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
	storeScene
	storeSceneCharacter
	storeSceneGate
	storeSceneSpotlight
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
			case storeScene:
				req.stores |= needScene
			case storeSceneCharacter:
				req.stores |= needSceneCharacter
			case storeSceneGate:
				req.stores |= needSceneGate
			case storeSceneSpotlight:
				req.stores |= needSceneSpotlight
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

	// campaign
	HandleProjection(r, campaign.EventTypeCreated, requirements(needsStores(storeCampaign), needsEnvelope(fieldEntityID)), Applier.applyCampaignCreated)
	HandleProjection(r, campaign.EventTypeUpdated, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignUpdated)
	HandleProjection(r, campaign.EventTypeAIBound, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignAIBound)
	HandleProjection(r, campaign.EventTypeAIUnbound, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignAIUnbound)
	HandleProjection(r, campaign.EventTypeAIAuthRotated, requirements(needsStores(storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyCampaignAIAuthRotated)
	HandleProjection(r, campaign.EventTypeForked, requirements(needsStores(storeCampaignFork), needsEnvelope(fieldCampaignID)), Applier.applyCampaignForked)

	// participant
	HandleProjection(r, participant.EventTypeJoined, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantJoined)
	HandleProjection(r, participant.EventTypeUpdated, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantUpdated)
	HandleProjectionRaw(r, participant.EventTypeLeft, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantLeft)
	HandleProjection(r, participant.EventTypeBound, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantBound)
	HandleProjection(r, participant.EventTypeUnbound, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantUnbound)
	HandleProjection(r, participant.EventTypeSeatReassigned, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applySeatReassigned)

	// character
	HandleProjection(r, character.EventTypeCreated, requirements(needsStores(storeCharacter, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyCharacterCreated)
	HandleProjection(r, character.EventTypeUpdated, requirements(needsStores(storeCharacter, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyCharacterUpdated)
	HandleProjection(r, character.EventTypeDeleted, requirements(needsStores(storeCharacter, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyCharacterDeleted)

	// invite — InviteID comes from payload with EntityID fallback for created/updated.
	HandleProjection(r, invite.EventTypeCreated, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyInviteCreated)
	HandleProjection(r, invite.EventTypeClaimed, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyInviteClaimed)
	HandleProjection(r, invite.EventTypeDeclined, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyInviteDeclined)
	HandleProjection(r, invite.EventTypeRevoked, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyInviteRevoked)
	HandleProjection(r, invite.EventTypeUpdated, requirements(needsStores(storeInvite)), Applier.applyInviteUpdated)

	// session — SessionID from payload with EntityID fallback, so EntityID
	// is not a hard envelope requirement for started/ended.
	HandleProjection(r, session.EventTypeStarted, requirements(needsStores(storeSession), needsEnvelope(fieldCampaignID)), Applier.applySessionStarted)
	HandleProjection(r, session.EventTypeEnded, requirements(needsStores(storeSession), needsEnvelope(fieldCampaignID)), Applier.applySessionEnded)
	// Gate handlers derive GateID from payload with EntityID fallback.
	HandleProjection(r, session.EventTypeGateOpened, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateOpened)
	HandleProjection(r, session.EventTypeGateResponseRecorded, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateResponseRecorded)
	HandleProjection(r, session.EventTypeGateResolved, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateResolved)
	HandleProjection(r, session.EventTypeGateAbandoned, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateAbandoned)
	HandleProjection(r, session.EventTypeSpotlightSet, requirements(needsStores(storeSessionSpotlight), needsEnvelope(fieldSessionID)), Applier.applySessionSpotlightSet)
	HandleProjectionRaw(r, session.EventTypeSpotlightCleared, requirements(needsStores(storeSessionSpotlight), needsEnvelope(fieldSessionID)), Applier.applySessionSpotlightCleared)

	// scene
	HandleProjection(r, scene.EventTypeCreated, requirements(needsStores(storeScene, storeSceneCharacter), needsEnvelope(fieldCampaignID)), Applier.applySceneCreated)
	HandleProjection(r, scene.EventTypeUpdated, requirements(needsStores(storeScene), needsEnvelope(fieldCampaignID)), Applier.applySceneUpdated)
	HandleProjection(r, scene.EventTypeEnded, requirements(needsStores(storeScene, storeSceneSpotlight), needsEnvelope(fieldCampaignID)), Applier.applySceneEnded)
	HandleProjection(r, scene.EventTypeCharacterAdded, requirements(needsStores(storeSceneCharacter), needsEnvelope(fieldCampaignID)), Applier.applySceneCharacterAdded)
	HandleProjection(r, scene.EventTypeCharacterRemoved, requirements(needsStores(storeSceneCharacter), needsEnvelope(fieldCampaignID)), Applier.applySceneCharacterRemoved)
	HandleProjection(r, scene.EventTypeGateOpened, requirements(needsStores(storeSceneGate), needsEnvelope(fieldCampaignID)), Applier.applySceneGateOpened)
	HandleProjection(r, scene.EventTypeGateResolved, requirements(needsStores(storeSceneGate), needsEnvelope(fieldCampaignID)), Applier.applySceneGateResolved)
	HandleProjection(r, scene.EventTypeGateAbandoned, requirements(needsStores(storeSceneGate), needsEnvelope(fieldCampaignID)), Applier.applySceneGateAbandoned)
	HandleProjection(r, scene.EventTypeSpotlightSet, requirements(needsStores(storeSceneSpotlight), needsEnvelope(fieldCampaignID)), Applier.applySceneSpotlightSet)
	HandleProjection(r, scene.EventTypeSpotlightCleared, requirements(needsStores(storeSceneSpotlight), needsEnvelope(fieldCampaignID)), Applier.applySceneSpotlightCleared)

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
	{needScene, "scene", func(a Applier) bool { return a.Scene == nil }},
	{needSceneCharacter, "scene character", func(a Applier) bool { return a.SceneCharacter == nil }},
	{needSceneGate, "scene gate", func(a Applier) bool { return a.SceneGate == nil }},
	{needSceneSpotlight, "scene spotlight", func(a Applier) bool { return a.SceneSpotlight == nil }},
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
	if a.BuildErr != nil {
		return fmt.Errorf("projection applier initialization failed: %w", a.BuildErr)
	}
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
