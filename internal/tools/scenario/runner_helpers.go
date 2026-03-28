package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/coreevent"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	scenarioReadinessClassID       = "class.guardian"
	scenarioReadinessSubclassID    = "subclass.stalwart"
	scenarioReadinessAncestryID    = "heritage.human"
	scenarioReadinessCommunityID   = "heritage.highborne"
	scenarioReadinessWeaponID      = "weapon.longsword"
	scenarioReadinessArmorID       = "armor.gambeson-armor"
	scenarioReadinessPotionItemID  = "item.minor-health-potion"
	scenarioReadinessDomainCardID1 = "domain_card.valor-bare-bones"
	scenarioReadinessDomainCardID2 = "domain_card.valor-forceful-push"
)

func (r *Runner) failf(format string, args ...any) error {
	return r.assertions.Failf(format, args...)
}

func (r *Runner) assertf(format string, args ...any) error {
	return r.assertions.Assertf(format, args...)
}

func (r *Runner) ensureCampaign(state *scenarioState) error {
	if state.campaignID == "" {
		return r.failf("campaign is required")
	}
	return nil
}

func (r *Runner) ensureSession(ctx context.Context, state *scenarioState) error {
	if state.campaignID == "" {
		return r.failf("campaign is required")
	}
	if state.sessionID != "" {
		return nil
	}
	if err := r.ensureSessionStartReadiness(ctx, state); err != nil {
		return err
	}
	response, err := r.env.sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId:           state.campaignID,
		Name:                 "Scenario Session",
		CharacterControllers: r.sessionCharacterControllers(ctx, state),
	})
	if err != nil {
		return fmt.Errorf("auto start session: %w", err)
	}
	if response.GetSession() == nil {
		return r.failf("expected session")
	}
	state.sessionID = response.GetSession().GetId()
	state.sessionImplicit = true
	return nil
}

func (r *Runner) ensureSessionStartReadiness(ctx context.Context, state *scenarioState) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	if state.ownerParticipantID == "" || r.env.characterClient == nil {
		// Unit tests may construct partial state/env and call helpers directly.
		return nil
	}
	registration, hasSystem, err := scenarioSystemForState(state)
	if err != nil {
		return err
	}

	participantCtx := withParticipantID(ctx, state.ownerParticipantID)
	if r.env.participantClient != nil {
		participantsResp, err := r.env.participantClient.ListParticipants(participantCtx, &gamev1.ListParticipantsRequest{
			CampaignId: state.campaignID,
			PageSize:   200,
		})
		if err != nil {
			if status.Code(err) != codes.Unimplemented {
				return fmt.Errorf("list participants: %w", err)
			}
		} else {
			hasPlayerParticipant := false
			for _, participant := range participantsResp.GetParticipants() {
				if participant.GetRole() == gamev1.ParticipantRole_PLAYER && strings.TrimSpace(participant.GetId()) != "" {
					hasPlayerParticipant = true
					break
				}
			}
			if !hasPlayerParticipant {
				response, createErr := r.env.participantClient.CreateParticipant(participantCtx, &gamev1.CreateParticipantRequest{
					CampaignId: state.campaignID,
					Name:       "Scenario Player",
					Role:       gamev1.ParticipantRole_PLAYER,
					Controller: gamev1.Controller_CONTROLLER_HUMAN,
				})
				if createErr != nil {
					return fmt.Errorf("create readiness participant: %w", createErr)
				}
				if response.GetParticipant() == nil || strings.TrimSpace(response.GetParticipant().GetId()) == "" {
					return r.failf("create readiness participant returned empty participant")
				}
			}
		}
	}

	pageToken := ""
	for {
		charactersResp, listErr := r.env.characterClient.ListCharacters(participantCtx, &gamev1.ListCharactersRequest{
			CampaignId: state.campaignID,
			PageSize:   200,
			PageToken:  pageToken,
		})
		if listErr != nil {
			if status.Code(listErr) == codes.Unimplemented {
				return nil
			}
			return fmt.Errorf("list characters: %w", listErr)
		}
		for _, character := range charactersResp.GetCharacters() {
			characterID := strings.TrimSpace(character.GetId())
			if characterID == "" {
				continue
			}
			if hasSystem && registration.characterNeedsReadiness != nil {
				needsReadiness, readinessErr := registration.characterNeedsReadiness(r, participantCtx, state, characterID)
				if readinessErr != nil {
					return readinessErr
				}
				if needsReadiness && registration.ensureCharacterReadiness != nil {
					if readinessErr := registration.ensureCharacterReadiness(r, participantCtx, state, characterID, nil); readinessErr != nil {
						return readinessErr
					}
				}
			}
		}
		next := strings.TrimSpace(charactersResp.GetNextPageToken())
		if next == "" {
			break
		}
		pageToken = next
	}
	return nil
}

func (r *Runner) sessionCharacterControllers(ctx context.Context, state *scenarioState) []*gamev1.SessionCharacterControllerAssignment {
	if state == nil || state.campaignID == "" || r.env.characterClient == nil {
		return nil
	}
	resp, err := r.env.characterClient.ListCharacters(withParticipantID(ctx, state.ownerParticipantID), &gamev1.ListCharactersRequest{
		CampaignId: state.campaignID,
		PageSize:   200,
	})
	if err != nil || resp == nil {
		return nil
	}
	assignments := make([]*gamev1.SessionCharacterControllerAssignment, 0, len(resp.GetCharacters()))
	fallbackParticipantID := ""
	for _, participantID := range state.participants {
		trimmed := strings.TrimSpace(participantID)
		if trimmed == "" {
			continue
		}
		fallbackParticipantID = trimmed
		break
	}
	for _, character := range resp.GetCharacters() {
		characterID := strings.TrimSpace(character.GetId())
		if characterID == "" {
			continue
		}
		participantID := strings.TrimSpace(character.GetOwnerParticipantId().GetValue())
		if participantID == "" {
			participantID = fallbackParticipantID
		}
		if participantID == "" {
			continue
		}
		assignments = append(assignments, &gamev1.SessionCharacterControllerAssignment{
			CharacterId:   characterID,
			ParticipantId: participantID,
		})
	}
	return assignments
}

func (r *Runner) ensureScenarioCharacterReadiness(
	ctx context.Context,
	state *scenarioState,
	characterID string,
	args map[string]any,
) error {
	registration, hasSystem, err := scenarioSystemForState(state)
	if err != nil {
		return err
	}
	if !hasSystem || registration.ensureCharacterReadiness == nil {
		return nil
	}
	return registration.ensureCharacterReadiness(r, ctx, state, characterID, args)
}

func (r *Runner) ensureDaggerheartCharacterReadiness(
	ctx context.Context,
	state *scenarioState,
	characterID string,
	args map[string]any,
) error {
	input := buildScenarioDaggerheartWorkflowInput(args)
	_, err := r.env.characterClient.ApplyCharacterCreationWorkflow(ctx, &gamev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemWorkflow: &gamev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: input,
		},
	})
	if err != nil {
		return fmt.Errorf("apply readiness workflow for %s: %w", characterID, err)
	}
	if err := r.waitForDaggerheartCharacterProjection(ctx, state, characterID, nil, nil); err != nil {
		return err
	}
	return nil
}

func (r *Runner) waitForDaggerheartCharacterProjection(
	ctx context.Context,
	state *scenarioState,
	characterID string,
	expectedProfile *daggerheartv1.DaggerheartProfile,
	expectedState *daggerheartv1.DaggerheartCharacterState,
) error {
	const (
		maxAttempts = 12
		retryDelay  = 25 * time.Millisecond
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		sheet, err := r.env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
			CampaignId:  state.campaignID,
			CharacterId: characterID,
		})
		if err != nil {
			return fmt.Errorf("get character sheet after readiness workflow: %w", err)
		}
		profile := sheet.GetProfile().GetDaggerheart()
		current := sheet.GetState().GetDaggerheart()
		stateReady := current != nil
		if expectedState == nil {
			stateReady = stateReady && current.GetHp() > 0 && current.GetHopeMax() > 0
		}
		if profile != nil &&
			strings.TrimSpace(profile.GetClassId()) != "" &&
			strings.TrimSpace(profile.GetSubclassId()) != "" &&
			stateReady &&
			daggerheartProfileMatches(profile, expectedProfile) &&
			daggerheartCharacterStateMatches(current, expectedState) {
			return nil
		}
		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}
	return fmt.Errorf("daggerheart readiness did not project for %s", characterID)
}

func (r *Runner) waitForDaggerheartStatModifierProjection(
	ctx context.Context,
	state *scenarioState,
	characterID string,
	expected []*daggerheartv1.DaggerheartStatModifier,
) error {
	const (
		maxAttempts = 200
		retryDelay  = 25 * time.Millisecond
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		sheet, err := r.env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
			CampaignId:  state.campaignID,
			CharacterId: characterID,
		})
		if err != nil {
			return fmt.Errorf("get character sheet after stat modifier change: %w", err)
		}
		current := sheet.GetState().GetDaggerheart()
		if current != nil && daggerheartStatModifiersMatch(current.GetStatModifiers(), expected) {
			return nil
		}
		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}
	return fmt.Errorf("daggerheart stat modifiers did not project for %s", characterID)
}

func daggerheartProfileMatches(actual, expected *daggerheartv1.DaggerheartProfile) bool {
	if expected == nil {
		return true
	}
	if actual == nil {
		return false
	}
	if expected.GetLevel() != 0 && actual.GetLevel() != expected.GetLevel() {
		return false
	}
	if expected.GetHpMax() != 0 && actual.GetHpMax() != expected.GetHpMax() {
		return false
	}
	if classID := strings.TrimSpace(expected.GetClassId()); classID != "" && actual.GetClassId() != classID {
		return false
	}
	if subclassID := strings.TrimSpace(expected.GetSubclassId()); subclassID != "" && actual.GetSubclassId() != subclassID {
		return false
	}
	if !optionalInt32Matches(actual.GetStressMax(), expected.GetStressMax()) ||
		!optionalInt32Matches(actual.GetEvasion(), expected.GetEvasion()) ||
		!optionalInt32Matches(actual.GetMajorThreshold(), expected.GetMajorThreshold()) ||
		!optionalInt32Matches(actual.GetSevereThreshold(), expected.GetSevereThreshold()) ||
		!optionalInt32Matches(actual.GetArmorMax(), expected.GetArmorMax()) ||
		!optionalInt32Matches(actual.GetArmorScore(), expected.GetArmorScore()) {
		return false
	}
	return true
}

func daggerheartCharacterStateMatches(actual, expected *daggerheartv1.DaggerheartCharacterState) bool {
	if expected == nil {
		return true
	}
	if actual == nil {
		return false
	}
	return actual.GetHp() == expected.GetHp() &&
		actual.GetHope() == expected.GetHope() &&
		actual.GetHopeMax() == expected.GetHopeMax() &&
		actual.GetStress() == expected.GetStress() &&
		actual.GetArmor() == expected.GetArmor() &&
		actual.GetLifeState() == expected.GetLifeState()
}

func daggerheartStatModifiersMatch(actual, expected []*daggerheartv1.DaggerheartStatModifier) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualKeys := make([]string, 0, len(actual))
	for _, modifier := range actual {
		actualKeys = append(actualKeys, daggerheartStatModifierKey(modifier))
	}
	expectedKeys := make([]string, 0, len(expected))
	for _, modifier := range expected {
		expectedKeys = append(expectedKeys, daggerheartStatModifierKey(modifier))
	}
	sort.Strings(actualKeys)
	sort.Strings(expectedKeys)
	return reflect.DeepEqual(actualKeys, expectedKeys)
}

func daggerheartStatModifierKey(modifier *daggerheartv1.DaggerheartStatModifier) string {
	if modifier == nil {
		return ""
	}
	triggers := append([]daggerheartv1.DaggerheartConditionClearTrigger(nil), modifier.GetClearTriggers()...)
	sort.Slice(triggers, func(i, j int) bool { return triggers[i] < triggers[j] })
	return fmt.Sprintf(
		"%s|%s|%d|%s|%s|%v",
		strings.TrimSpace(modifier.GetId()),
		strings.TrimSpace(modifier.GetTarget()),
		modifier.GetDelta(),
		strings.TrimSpace(modifier.GetLabel()),
		strings.TrimSpace(modifier.GetSource()),
		triggers,
	)
}

func optionalInt32Matches(actual, expected *wrapperspb.Int32Value) bool {
	switch {
	case expected == nil:
		return true
	case actual == nil:
		return false
	default:
		return actual.GetValue() == expected.GetValue()
	}
}

func (r *Runner) scenarioCharacterNeedsReadiness(
	ctx context.Context,
	state *scenarioState,
	characterID string,
) (bool, error) {
	registration, hasSystem, err := scenarioSystemForState(state)
	if err != nil {
		return false, err
	}
	if !hasSystem || registration.characterNeedsReadiness == nil {
		return false, nil
	}
	return registration.characterNeedsReadiness(r, ctx, state, characterID)
}

func (r *Runner) daggerheartCharacterNeedsReadiness(
	ctx context.Context,
	state *scenarioState,
	characterID string,
) (bool, error) {
	sheet, err := r.env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return false, fmt.Errorf("get character sheet for readiness check: %w", err)
	}
	profile := sheet.GetProfile().GetDaggerheart()
	if profile == nil {
		return true, nil
	}
	return strings.TrimSpace(profile.GetClassId()) == "" ||
		strings.TrimSpace(profile.GetSubclassId()) == "", nil
}

func (r *Runner) latestSeq(ctx context.Context, state *scenarioState) (uint64, error) {
	if state.campaignID == "" {
		return 0, nil
	}
	eventCtx := withParticipantID(ctx, state.ownerParticipantID)
	response, err := r.env.eventClient.ListEvents(eventCtx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		return 0, fmt.Errorf("list events: %w", err)
	}
	if len(response.GetEvents()) == 0 {
		return 0, nil
	}
	return response.GetEvents()[0].GetSeq(), nil
}

func (r *Runner) requireEventTypesAfterSeq(ctx context.Context, state *scenarioState, before uint64, types ...event.Type) error {
	eventCtx := withParticipantID(ctx, state.ownerParticipantID)
	for _, eventType := range types {
		filter := fmt.Sprintf("type = \"%s\"", eventType)
		if state.sessionID != "" && isSessionEvent(string(eventType)) {
			filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
		}
		response, err := r.env.eventClient.ListEvents(eventCtx, &gamev1.ListEventsRequest{
			CampaignId: state.campaignID,
			PageSize:   1,
			OrderBy:    "seq desc",
			Filter:     filter,
		})
		if err != nil {
			return fmt.Errorf("list events for %s: %w", eventType, err)
		}
		if len(response.GetEvents()) == 0 {
			if err := r.assertf("expected event %s", eventType); err != nil {
				return err
			}
			continue
		}
		if response.GetEvents()[0].GetSeq() <= before {
			if err := r.assertf("expected %s after seq %d", eventType, before); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) requireDaggerheartEventTypesAfterSeq(ctx context.Context, state *scenarioState, before uint64, types ...any) error {
	eventTypes, err := convertDaggerheartEventTypes(types...)
	if err != nil {
		return fmt.Errorf("convert daggerheart event type: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, eventTypes...)
}

func convertDaggerheartEventTypes(types ...any) ([]event.Type, error) {
	if len(types) == 0 {
		return nil, nil
	}
	converted := make([]event.Type, 0, len(types))
	for _, eventType := range types {
		value := reflect.ValueOf(eventType)
		if !value.IsValid() || value.Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported event type %T", eventType)
		}
		converted = append(converted, event.Type(value.String()))
	}
	return converted, nil
}

func (r *Runner) requireAnyEventTypesAfterSeq(ctx context.Context, state *scenarioState, before uint64, types ...event.Type) error {
	for _, eventType := range types {
		ok, err := r.hasEventTypeAfterSeq(ctx, state, before, eventType)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	labels := make([]string, 0, len(types))
	for _, eventType := range types {
		labels = append(labels, string(eventType))
	}
	return r.assertf("expected event after seq %d: %s", before, strings.Join(labels, ", "))
}

func (r *Runner) requireNoSessionEventsAfterSeq(ctx context.Context, state *scenarioState, before uint64) error {
	if state.sessionID == "" {
		return r.failf("session is required")
	}
	eventCtx := withParticipantID(ctx, state.ownerParticipantID)
	response, err := r.env.eventClient.ListEvents(eventCtx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     fmt.Sprintf("session_id = \"%s\"", state.sessionID),
	})
	if err != nil {
		return fmt.Errorf("list session events: %w", err)
	}
	for _, evt := range response.GetEvents() {
		if evt.GetSeq() <= before {
			break
		}
		if strings.TrimSpace(evt.GetType()) == "" {
			continue
		}
		return r.assertf("expected no session events after seq %d, got %s@%d", before, evt.GetType(), evt.GetSeq())
	}
	return nil
}

func (r *Runner) resolveOpenSessionGate(ctx context.Context, state *scenarioState, before uint64) error {
	filter := fmt.Sprintf("type = \"%s\"", event.TypeSessionGateOpened)
	if state.sessionID != "" {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	eventCtx := withParticipantID(ctx, state.ownerParticipantID)
	response, err := r.env.eventClient.ListEvents(eventCtx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return fmt.Errorf("list events for %s: %w", event.TypeSessionGateOpened, err)
	}
	gateID := ""
	for _, evt := range response.GetEvents() {
		if evt.GetSeq() <= before {
			continue
		}
		if len(strings.TrimSpace(string(evt.GetPayloadJson()))) == 0 {
			continue
		}
		var payload event.SessionGateOpenedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return fmt.Errorf("decode session gate payload: %w", err)
		}
		if strings.TrimSpace(payload.GateID.String()) == "" {
			continue
		}
		gateID = payload.GateID.String()
		break
	}
	if gateID == "" {
		return r.failf("session gate opened event not found")
	}
	_, err = r.env.sessionClient.ResolveSessionGate(eventCtx, &gamev1.ResolveSessionGateRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
		GateId:     gateID,
		Decision:   "allow",
	})
	if err != nil {
		return fmt.Errorf("resolve session gate: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeSessionGateResolved)
}

func (r *Runner) hasEventTypeAfterSeq(ctx context.Context, state *scenarioState, before uint64, eventType event.Type) (bool, error) {
	filter := fmt.Sprintf("type = \"%s\"", eventType)
	if state.sessionID != "" && isSessionEvent(string(eventType)) {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	eventCtx := withParticipantID(ctx, state.ownerParticipantID)
	response, err := r.env.eventClient.ListEvents(eventCtx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return false, fmt.Errorf("list events for %s: %w", eventType, err)
	}
	if len(response.GetEvents()) == 0 {
		return false, nil
	}
	return response.GetEvents()[0].GetSeq() > before, nil
}

func isSessionEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "action.") || strings.HasPrefix(eventType, "session.")
}

func (r *Runner) applyDefaultDaggerheartProfile(ctx context.Context, state *scenarioState, characterID string, args map[string]any) (*daggerheartv1.DaggerheartProfile, error) {
	if !hasDaggerheartProfileOverrides(args) {
		return nil, nil
	}

	profile := &daggerheartv1.DaggerheartProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       wrapperspb.Int32(6),
		Evasion:         wrapperspb.Int32(10),
		MajorThreshold:  wrapperspb.Int32(3),
		SevereThreshold: wrapperspb.Int32(6),
	}

	armorValue := optionalInt(args, "armor", 0)
	armorMaxValue := optionalInt(args, "armor_max", 0)
	profile.Level = int32(optionalInt(args, "level", int(profile.GetLevel())))
	profile.HpMax = int32(optionalInt(args, "hp_max", int(profile.GetHpMax())))
	profile.StressMax = wrapperspb.Int32(int32(optionalInt(args, "stress_max", int(profile.GetStressMax().GetValue()))))
	profile.Evasion = wrapperspb.Int32(int32(optionalInt(args, "evasion", int(profile.GetEvasion().GetValue()))))
	profile.MajorThreshold = wrapperspb.Int32(int32(optionalInt(args, "major_threshold", int(profile.GetMajorThreshold().GetValue()))))
	profile.SevereThreshold = wrapperspb.Int32(int32(optionalInt(args, "severe_threshold", int(profile.GetSevereThreshold().GetValue()))))
	if armorMaxValue > 0 {
		profile.ArmorMax = wrapperspb.Int32(int32(armorMaxValue))
	} else if armorValue > 0 {
		profile.ArmorMax = wrapperspb.Int32(int32(armorValue))
	}
	if value := optionalInt(args, "armor_score", 0); value > 0 {
		profile.ArmorScore = wrapperspb.Int32(int32(value))
	}

	response, err := r.env.characterClient.PatchCharacterProfile(ctx, &gamev1.PatchCharacterProfileRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemProfilePatch: &gamev1.PatchCharacterProfileRequest_Daggerheart{
			Daggerheart: profile,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("patch character profile: %w", err)
	}
	if projected := response.GetProfile().GetDaggerheart(); projected != nil {
		return projected, nil
	}
	return nil, nil
}

func hasDaggerheartProfileOverrides(args map[string]any) bool {
	if len(args) == 0 {
		return false
	}
	keys := []string{
		"level",
		"hp_max",
		"stress_max",
		"evasion",
		"major_threshold",
		"severe_threshold",
		"armor",
		"armor_max",
		"armor_score",
	}
	for _, key := range keys {
		if _, ok := args[key]; ok {
			return true
		}
	}
	return false
}

func (r *Runner) applyOptionalCharacterState(
	ctx context.Context,
	state *scenarioState,
	characterID string,
	args map[string]any,
	profile *daggerheartv1.DaggerheartProfile,
) (*daggerheartv1.DaggerheartCharacterState, error) {
	patch := &daggerheartv1.DaggerheartCharacterState{}
	hasPatch := false
	armorSet := false
	hpSet := false
	hopeSet := false
	hopeMaxSet := false
	stressSet := false
	lifeStateSet := false
	if armor, ok := readInt(args, "armor"); ok {
		patch.Armor = int32(armor)
		hasPatch = true
		armorSet = true
	}
	if hp, ok := readInt(args, "hp"); ok {
		patch.Hp = int32(hp)
		hasPatch = true
		hpSet = true
	}
	if hope, ok := readInt(args, "hope"); ok {
		patch.Hope = int32(hope)
		hasPatch = true
		hopeSet = true
	}
	if hopeMax, ok := readInt(args, "hope_max"); ok {
		patch.HopeMax = int32(hopeMax)
		hasPatch = true
		hopeMaxSet = true
	}
	if stress, ok := readInt(args, "stress"); ok {
		patch.Stress = int32(stress)
		hasPatch = true
		stressSet = true
	}
	if lifeState := optionalString(args, "life_state", ""); lifeState != "" {
		value, err := parseLifeState(lifeState)
		if err != nil {
			return nil, err
		}
		patch.LifeState = value
		hasPatch = true
		lifeStateSet = true
	}
	if !hasPatch {
		return nil, nil
	}
	// PatchCharacterState overwrites the full state, so merge with current values.
	current, err := r.getCharacterState(ctx, state, characterID)
	if err != nil {
		return nil, err
	}
	if !hpSet {
		patch.Hp = current.GetHp()
	}
	if !hopeSet {
		patch.Hope = current.GetHope()
	}
	if !hopeMaxSet {
		patch.HopeMax = current.GetHopeMax()
	}
	if !stressSet {
		patch.Stress = current.GetStress()
	}
	if !armorSet {
		patch.Armor = current.GetArmor()
	}
	if !lifeStateSet {
		patch.LifeState = current.GetLifeState()
	}
	if !hpSet && profile != nil && patch.GetHp() > profile.GetHpMax() {
		patch.Hp = profile.GetHpMax()
	}
	if !stressSet && profile != nil && profile.GetStressMax() != nil && patch.GetStress() > profile.GetStressMax().GetValue() {
		patch.Stress = profile.GetStressMax().GetValue()
	}
	if !armorSet && profile != nil && profile.GetArmorMax() != nil && patch.GetArmor() > profile.GetArmorMax().GetValue() {
		patch.Armor = profile.GetArmorMax().GetValue()
	}
	if !hopeSet && patch.GetHope() > patch.GetHopeMax() {
		patch.Hope = patch.GetHopeMax()
	}
	response, err := r.env.snapshotClient.PatchCharacterState(ctx, &gamev1.PatchCharacterStateRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: patch,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("patch character state: %w", err)
	}
	if projected := response.GetState().GetDaggerheart(); projected != nil {
		return projected, nil
	}
	return nil, nil
}

func applyTraitValue(profile *daggerheartv1.DaggerheartProfile, key string, args map[string]any) {
	value := optionalInt(args, key, 0)
	if value == 0 {
		return
	}
	boxed := wrapperspb.Int32(int32(value))
	switch key {
	case "agility":
		profile.Agility = boxed
	case "strength":
		profile.Strength = boxed
	case "finesse":
		profile.Finesse = boxed
	case "instinct":
		profile.Instinct = boxed
	case "presence":
		profile.Presence = boxed
	case "knowledge":
		profile.Knowledge = boxed
	}
}

func (r *Runner) getSnapshot(ctx context.Context, state *scenarioState) (*daggerheartv1.DaggerheartSnapshot, error) {
	response, err := r.env.snapshotClient.GetSnapshot(ctx, &gamev1.GetSnapshotRequest{CampaignId: state.campaignID})
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	if response.GetSnapshot() == nil || response.GetSnapshot().GetDaggerheart() == nil {
		return nil, r.failf("expected daggerheart snapshot")
	}
	return response.GetSnapshot().GetDaggerheart(), nil
}

// getDaggerheartProfile reads the Daggerheart profile projection for one
// character so scenario steps can assert durable creation outcomes.
func (r *Runner) getDaggerheartProfile(
	ctx context.Context,
	state *scenarioState,
	characterID string,
) (*daggerheartv1.DaggerheartProfile, error) {
	response, err := r.env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return nil, fmt.Errorf("get character sheet: %w", err)
	}
	if response.GetProfile() == nil || response.GetProfile().GetDaggerheart() == nil {
		return nil, r.failf("expected daggerheart character profile")
	}
	return response.GetProfile().GetDaggerheart(), nil
}

func (r *Runner) getCharacterState(ctx context.Context, state *scenarioState, characterID string) (*daggerheartv1.DaggerheartCharacterState, error) {
	response, err := r.env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return nil, fmt.Errorf("get character sheet: %w", err)
	}
	if response.GetState() == nil || response.GetState().GetDaggerheart() == nil {
		return nil, r.failf("expected daggerheart character state")
	}
	return response.GetState().GetDaggerheart(), nil
}

func (r *Runner) getAdversary(ctx context.Context, state *scenarioState, adversaryID string) (*daggerheartv1.DaggerheartAdversary, error) {
	response, err := r.env.daggerheartClient.GetAdversary(ctx, &daggerheartv1.DaggerheartGetAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryID,
	})
	if err != nil {
		return nil, fmt.Errorf("get adversary: %w", err)
	}
	if response.GetAdversary() == nil {
		return nil, r.failf("expected adversary")
	}
	return response.GetAdversary(), nil
}

// buildScenarioDaggerheartWorkflowInput produces a full valid Daggerheart
// creation workflow payload using scenario defaults, with optional explicit
// scenario overrides layered on top for focused coverage.
func buildScenarioDaggerheartWorkflowInput(args map[string]any) *daggerheartv1.DaggerheartCreationWorkflowInput {
	heritageArgs := readMap(args, "heritage")
	equipmentArgs := readMap(args, "equipment")
	firstAncestryID := optionalString(heritageArgs, "first_feature_ancestry_id", scenarioReadinessAncestryID)
	secondAncestryID := optionalString(heritageArgs, "second_feature_ancestry_id", firstAncestryID)
	domainCardIDs := readStringSlice(args, "domain_card_ids")
	if len(domainCardIDs) == 0 {
		domainCardIDs = []string{scenarioReadinessDomainCardID1, scenarioReadinessDomainCardID2}
	}

	input := &daggerheartv1.DaggerheartCreationWorkflowInput{
		ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{
			ClassId:    optionalString(args, "class_id", scenarioReadinessClassID),
			SubclassId: optionalString(args, "subclass_id", scenarioReadinessSubclassID),
		},
		HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{
			Heritage: &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
				AncestryLabel:           optionalString(heritageArgs, "ancestry_label", ""),
				FirstFeatureAncestryId:  firstAncestryID,
				SecondFeatureAncestryId: secondAncestryID,
				CommunityId:             optionalString(heritageArgs, "community_id", scenarioReadinessCommunityID),
			},
		},
		TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{
			Agility:   2,
			Strength:  1,
			Finesse:   1,
			Instinct:  0,
			Presence:  0,
			Knowledge: -1,
		},
		DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{
			Description: "A brave adventurer.",
		},
		EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
			WeaponIds:    []string{scenarioReadinessWeaponID},
			ArmorId:      optionalString(equipmentArgs, "armor_id", scenarioReadinessArmorID),
			PotionItemId: optionalString(equipmentArgs, "potion_item_id", scenarioReadinessPotionItemID),
		},
		BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{
			Background: "scenario background",
		},
		ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{
			Experiences: []*daggerheartv1.DaggerheartExperience{
				{Name: "scenario experience", Modifier: 2},
				{Name: "scenario patrol", Modifier: 2},
			},
		},
		DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{
			DomainCardIds: append([]string(nil), domainCardIDs...),
		},
		ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{
			Connections: "scenario connections",
		},
	}

	companionArgs := readMap(args, "companion")
	if len(companionArgs) > 0 {
		input.ClassSubclassInput.Companion = &daggerheartv1.DaggerheartCreationCompanionInput{
			AnimalKind:        optionalString(companionArgs, "animal_kind", ""),
			Name:              optionalString(companionArgs, "name", ""),
			ExperienceIds:     readStringSlice(companionArgs, "experience_ids"),
			AttackDescription: optionalString(companionArgs, "attack_description", ""),
			DamageType:        optionalString(companionArgs, "damage_type", ""),
		}
	}

	return input
}

func chooseActionSeed(args map[string]any, difficulty int) (uint64, error) {
	hint := strings.ToLower(optionalString(args, "outcome", ""))
	total, hasTotal := readInt(args, "total")
	modifier := optionalInt(args, "modifier", 0)
	modifier += actionRollModifierTotal(args, "modifiers")
	advantage := optionalInt(args, "advantage", 0)
	disadvantage := optionalInt(args, "disadvantage", 0)
	key := actionSeedKey{
		difficulty:   difficulty,
		hint:         hint,
		total:        total,
		exactTotal:   hasTotal,
		modifier:     modifier,
		advantage:    advantage,
		disadvantage: disadvantage,
	}
	if hint == "" && !hasTotal {
		return 42, nil
	}
	if seed, ok := cachedActionSeed(key); ok {
		return seed, nil
	}
	for seed := uint64(1); seed < 50000; seed++ {
		result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
			Modifier:     modifier,
			Difficulty:   &difficulty,
			Advantage:    advantage,
			Disadvantage: disadvantage,
			Seed:         int64(seed),
		})
		if err != nil {
			continue
		}
		if hasTotal && result.Total != total {
			continue
		}
		if hint == "" || matchesOutcomeHint(result, hint) {
			cacheActionSeed(key, seed)
			return seed, nil
		}
	}
	if hasTotal {
		if hint == "" {
			return 0, fmt.Errorf("no seed found for total %d", total)
		}
		return 0, fmt.Errorf("no seed found for outcome %q and total %d", hint, total)
	}
	return 0, fmt.Errorf("no seed found for outcome %q", hint)
}

type actionSeedKey struct {
	difficulty   int
	hint         string
	total        int
	exactTotal   bool
	modifier     int
	advantage    int
	disadvantage int
}

var (
	actionSeedMu    sync.Mutex
	actionSeedCache = map[actionSeedKey]uint64{}
)

func cachedActionSeed(key actionSeedKey) (uint64, bool) {
	actionSeedMu.Lock()
	defer actionSeedMu.Unlock()
	seed, ok := actionSeedCache[key]
	return seed, ok
}

func cacheActionSeed(key actionSeedKey, seed uint64) {
	actionSeedMu.Lock()
	defer actionSeedMu.Unlock()
	actionSeedCache[key] = seed
}

func actionRollModifierTotal(args map[string]any, key string) int {
	value, ok := args[key]
	if !ok {
		return 0
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return 0
	}
	total := 0
	for index, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		source := optionalString(item, "source", fmt.Sprintf("modifier_%d", index))
		value, ok := readInt(item, "value")
		if !ok {
			if isHopeSpendSource(source) {
				value = 0
			} else {
				continue
			}
		}
		total += value
	}
	return total
}

func matchesOutcomeHint(result daggerheartdomain.ActionResult, hint string) bool {
	switch hint {
	case "fear":
		return result.Outcome == daggerheartdomain.OutcomeRollWithFear ||
			result.Outcome == daggerheartdomain.OutcomeSuccessWithFear ||
			result.Outcome == daggerheartdomain.OutcomeFailureWithFear
	case "hope":
		return result.Outcome == daggerheartdomain.OutcomeRollWithHope ||
			result.Outcome == daggerheartdomain.OutcomeSuccessWithHope ||
			result.Outcome == daggerheartdomain.OutcomeFailureWithHope
	case "critical":
		return result.IsCrit
	case "on_crit":
		return result.IsCrit
	case "failure_hope":
		return result.Outcome == daggerheartdomain.OutcomeFailureWithHope
	case "failure_fear":
		return result.Outcome == daggerheartdomain.OutcomeFailureWithFear
	case "success_hope":
		return result.Outcome == daggerheartdomain.OutcomeSuccessWithHope
	case "success_fear":
		return result.Outcome == daggerheartdomain.OutcomeSuccessWithFear
	default:
		return false
	}
}

func resolveOutcomeSeed(args map[string]any, key string, difficulty int, fallback uint64) (uint64, error) {
	hint := optionalString(args, key, "")
	if hint == "" {
		return fallback, nil
	}
	return chooseActionSeed(map[string]any{"outcome": hint}, difficulty)
}

func actionRollResultFromResponse(response *daggerheartv1.SessionActionRollResponse) actionRollResult {
	if response == nil {
		return actionRollResult{}
	}
	return actionRollResult{
		rollSeq:    response.GetRollSeq(),
		hopeDie:    int(response.GetHopeDie()),
		fearDie:    int(response.GetFearDie()),
		total:      int(response.GetTotal()),
		difficulty: int(response.GetDifficulty()),
		success:    response.GetSuccess(),
		crit:       response.GetCrit(),
	}
}

func parseOutcomeBranchSteps(value any, defaultSystem string) ([]Step, error) {
	if value == nil {
		return nil, nil
	}

	raw, ok := value.([]any)
	if !ok {
		entry, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("outcome branch expects a list of step objects")
		}
		raw = []any{entry}
	}
	steps := make([]Step, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("outcome branch step must be an object")
		}
		kind := optionalString(item, "kind", "")
		if kind == "" {
			return nil, fmt.Errorf("outcome branch step requires kind")
		}
		system := strings.ToUpper(strings.TrimSpace(optionalString(item, "system", "")))
		if _, isCoreStep := coreStepKinds[kind]; isCoreStep {
			if system != "" {
				return nil, fmt.Errorf("core step %q must not declare a system scope", kind)
			}
			system = ""
		} else if system == "" {
			system = strings.ToUpper(strings.TrimSpace(defaultSystem))
		}
		args := map[string]any{}
		for key, value := range item {
			if key == "kind" || key == "system" {
				continue
			}
			args[key] = value
		}
		steps = append(steps, Step{System: system, Kind: kind, Args: args})
	}
	return steps, nil
}

func resolveOutcomeBranches(args map[string]any, allowed map[string]struct{}, defaultSystem string) (map[string][]Step, error) {
	if _, hasCritical := args["on_critical"]; hasCritical {
		if _, hasCrit := args["on_crit"]; hasCrit {
			return nil, fmt.Errorf("on_critical and on_crit are aliases")
		}
	}
	branches := make(map[string][]Step)
	for key, value := range args {
		if !strings.HasPrefix(key, "on_") {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("unknown outcome branch %q", key)
		}
		if value == nil {
			continue
		}
		steps, err := parseOutcomeBranchSteps(value, defaultSystem)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		branches[key] = steps
	}
	return branches, nil
}

func evaluateActionOutcomeBranch(result actionRollResult, branch string) bool {
	switch branch {
	case "on_success":
		return result.success
	case "on_failure":
		return !result.success
	case "on_hope":
		return result.hopeDie > result.fearDie
	case "on_fear":
		return result.fearDie > result.hopeDie
	case "on_success_hope":
		return result.success && result.hopeDie > result.fearDie
	case "on_failure_hope":
		return !result.success && result.hopeDie > result.fearDie
	case "on_success_fear":
		return result.success && result.fearDie > result.hopeDie
	case "on_failure_fear":
		return !result.success && result.fearDie > result.hopeDie
	case "on_critical", "on_crit":
		return result.crit
	default:
		return false
	}
}

func evaluateReactionOutcomeBranch(result *daggerheartv1.DaggerheartReactionOutcomeResult, branch string) bool {
	if result == nil {
		return false
	}
	switch branch {
	case "on_success":
		return result.GetSuccess()
	case "on_failure":
		return !result.GetSuccess()
	case "on_hope":
		switch result.GetOutcome() {
		case daggerheartv1.Outcome_SUCCESS_WITH_HOPE, daggerheartv1.Outcome_FAILURE_WITH_HOPE, daggerheartv1.Outcome_ROLL_WITH_HOPE:
			return true
		default:
			return false
		}
	case "on_fear":
		switch result.GetOutcome() {
		case daggerheartv1.Outcome_SUCCESS_WITH_FEAR, daggerheartv1.Outcome_FAILURE_WITH_FEAR, daggerheartv1.Outcome_ROLL_WITH_FEAR:
			return true
		default:
			return false
		}
	case "on_success_hope":
		return result.GetSuccess() && result.GetOutcome() == daggerheartv1.Outcome_SUCCESS_WITH_HOPE
	case "on_failure_hope":
		return !result.GetSuccess() && result.GetOutcome() == daggerheartv1.Outcome_FAILURE_WITH_HOPE
	case "on_success_fear":
		return result.GetSuccess() && result.GetOutcome() == daggerheartv1.Outcome_SUCCESS_WITH_FEAR
	case "on_failure_fear":
		return !result.GetSuccess() && result.GetOutcome() == daggerheartv1.Outcome_FAILURE_WITH_FEAR
	case "on_critical", "on_crit":
		return result.GetCrit()
	default:
		return false
	}
}

func runOutcomeBranchSteps(ctx context.Context, state *scenarioState, r *Runner, branches map[string][]Step, orderedKeys []string, evaluator func(string) bool) error {
	for _, key := range orderedKeys {
		if !evaluator(key) {
			continue
		}
		steps, ok := branches[key]
		if !ok || len(steps) == 0 {
			continue
		}
		for index, step := range steps {
			if err := r.runStep(ctx, state, step); err != nil {
				return fmt.Errorf("%s step %d (%s): %w", key, index+1, step.Kind, err)
			}
		}
	}
	return nil
}

func buildActionRollModifiers(args map[string]any, key string) []*daggerheartv1.ActionRollModifier {
	value, ok := args[key]
	list, hasList := value.([]any)
	modifiers := make([]*daggerheartv1.ActionRollModifier, 0, 1)
	if ok && hasList && len(list) > 0 {
		modifiers = make([]*daggerheartv1.ActionRollModifier, 0, len(list)+1)
		for index, entry := range list {
			item, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			source := optionalString(item, "source", fmt.Sprintf("modifier_%d", index))
			value, ok := readInt(item, "value")
			if !ok {
				if isHopeSpendSource(source) {
					value = 0
				} else {
					continue
				}
			}
			modifiers = append(modifiers, &daggerheartv1.ActionRollModifier{
				Source: source,
				Value:  int32(value),
			})
		}
	}
	if _, hasModifier := args["modifier"]; hasModifier {
		modifier, ok := readInt(args, "modifier")
		if !ok {
			modifier = 0
		}
		modifiers = append(modifiers, &daggerheartv1.ActionRollModifier{
			Source: "modifier",
			Value:  int32(modifier),
		})
	}
	if len(modifiers) == 0 {
		return nil
	}
	return modifiers
}

func buildAdversaryRollModifiers(args map[string]any) []*daggerheartv1.ActionRollModifier {
	modifiers := buildActionRollModifiers(args, "modifiers")
	if len(modifiers) > 0 {
		return modifiers
	}
	if _, hasAttackModifier := args["attack_modifier"]; hasAttackModifier {
		modifier := optionalInt(args, "attack_modifier", 0)
		return []*daggerheartv1.ActionRollModifier{{
			Source: "attack_modifier",
			Value:  int32(modifier),
		}}
	}
	if _, hasModifier := args["modifier"]; hasModifier {
		modifier := optionalInt(args, "modifier", 0)
		return []*daggerheartv1.ActionRollModifier{{
			Source: "modifier",
			Value:  int32(modifier),
		}}
	}
	return nil
}

func buildDamageDice(args map[string]any) []*daggerheartv1.DiceSpec {
	value, ok := args["damage_dice"]
	if !ok {
		return []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}}
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}}
	}
	results := make([]*daggerheartv1.DiceSpec, 0, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		sides := optionalInt(item, "sides", 6)
		count := optionalInt(item, "count", 1)
		results = append(results, &daggerheartv1.DiceSpec{Sides: int32(sides), Count: int32(count)})
	}
	if len(results) == 0 {
		return []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}}
	}
	return results
}

func buildDamageSpec(args map[string]any, actorID, source string) *daggerheartv1.DaggerheartAttackDamageSpec {
	damageType := parseDamageType(optionalString(args, "damage_type", "physical"))
	spec := &daggerheartv1.DaggerheartAttackDamageSpec{DamageType: damageType}
	if source != "" {
		spec.Source = source
	}
	if actorID != "" {
		spec.SourceCharacterIds = []string{actorID}
	}
	spec.ResistPhysical = optionalBool(args, "resist_physical", false)
	spec.ResistMagic = optionalBool(args, "resist_magic", false)
	spec.ImmunePhysical = optionalBool(args, "immune_physical", false)
	spec.ImmuneMagic = optionalBool(args, "immune_magic", false)
	spec.Direct = optionalBool(args, "direct", false)
	spec.MassiveDamage = optionalBool(args, "massive_damage", false)
	return spec
}

func buildAttackRange(args map[string]any) daggerheartv1.DaggerheartAttackRange {
	switch strings.ToLower(strings.TrimSpace(optionalString(args, "attack_range", "melee"))) {
	case "", "melee":
		return daggerheartv1.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE
	case "ranged":
		return daggerheartv1.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED
	default:
		return daggerheartv1.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED
	}
}

func buildIncomingAttackArmorReaction(args map[string]any, seed uint64) *daggerheartv1.DaggerheartIncomingAttackArmorReaction {
	switch strings.ToLower(strings.TrimSpace(optionalString(args, "armor_reaction", ""))) {
	case "":
		return nil
	case "shifting":
		return &daggerheartv1.DaggerheartIncomingAttackArmorReaction{
			Reaction: &daggerheartv1.DaggerheartIncomingAttackArmorReaction_Shifting{
				Shifting: &daggerheartv1.DaggerheartShiftingArmorReaction{},
			},
		}
	case "timeslowing":
		return &daggerheartv1.DaggerheartIncomingAttackArmorReaction{
			Reaction: &daggerheartv1.DaggerheartIncomingAttackArmorReaction_Timeslowing{
				Timeslowing: &daggerheartv1.DaggerheartTimeslowingArmorReaction{
					Rng: &commonv1.RngRequest{
						Seed:     &seed,
						RollMode: commonv1.RollMode_REPLAY,
					},
				},
			},
		}
	default:
		return nil
	}
}

func buildDamageArmorReaction(args map[string]any, seed uint64) *daggerheartv1.DaggerheartDamageArmorReaction {
	switch strings.ToLower(strings.TrimSpace(optionalString(args, "armor_reaction", ""))) {
	case "":
		return nil
	case "resilient":
		return &daggerheartv1.DaggerheartDamageArmorReaction{
			Reaction: &daggerheartv1.DaggerheartDamageArmorReaction_Resilient{
				Resilient: &daggerheartv1.DaggerheartResilientArmorReaction{
					Rng: &commonv1.RngRequest{
						Seed:     &seed,
						RollMode: commonv1.RollMode_REPLAY,
					},
				},
			},
		}
	case "impenetrable":
		return &daggerheartv1.DaggerheartDamageArmorReaction{
			Reaction: &daggerheartv1.DaggerheartDamageArmorReaction_Impenetrable{
				Impenetrable: &daggerheartv1.DaggerheartImpenetrableArmorReaction{},
			},
		}
	default:
		return nil
	}
}

func buildDamageRequest(args map[string]any, actorID, source string, amount int32) *daggerheartv1.DaggerheartDamageRequest {
	damageType := parseDamageType(optionalString(args, "damage_type", "physical"))
	request := &daggerheartv1.DaggerheartDamageRequest{Amount: amount, DamageType: damageType}
	if source != "" {
		request.Source = source
	}
	if actorID != "" {
		request.SourceCharacterIds = []string{actorID}
	}
	request.ResistPhysical = optionalBool(args, "resist_physical", false)
	request.ResistMagic = optionalBool(args, "resist_magic", false)
	request.ImmunePhysical = optionalBool(args, "immune_physical", false)
	request.ImmuneMagic = optionalBool(args, "immune_magic", false)
	request.Direct = optionalBool(args, "direct", false)
	request.MassiveDamage = optionalBool(args, "massive_damage", false)
	return request
}

func buildDamageRequestWithSources(
	args map[string]any,
	source string,
	amount int32,
	sourceIDs []string,
) *daggerheartv1.DaggerheartDamageRequest {
	request := buildDamageRequest(args, "", source, amount)
	request.SourceCharacterIds = uniqueNonEmptyStrings(sourceIDs)
	return request
}

func (r *Runner) applyAdversaryDamage(
	ctx context.Context,
	state *scenarioState,
	adversaryID string,
	name string,
	damageRoll *daggerheartv1.SessionDamageRollResponse,
	args map[string]any,
) (bool, error) {
	before, err := r.getAdversary(ctx, state, adversaryID)
	if err != nil {
		return false, err
	}
	hpBefore := int(before.GetHp())
	armorBefore := int(before.GetArmor())
	majorThreshold := int(before.GetMajorThreshold())
	severeThreshold := int(before.GetSevereThreshold())

	amount := int(damageRoll.GetTotal())
	resistance := rules.ResistanceProfile{
		ResistPhysical: optionalBool(args, "resist_physical", false),
		ResistMagic:    optionalBool(args, "resist_magic", false),
		ImmunePhysical: optionalBool(args, "immune_physical", false),
		ImmuneMagic:    optionalBool(args, "immune_magic", false),
	}
	adjusted := rules.ApplyResistance(amount, damageTypesForArgs(args), resistance)
	if adjusted <= 0 {
		return false, nil
	}
	options := rules.DamageOptions{EnableMassiveDamage: optionalBool(args, "massive_damage", false)}

	result, err := rules.EvaluateDamage(adjusted, majorThreshold, severeThreshold, options)
	if err != nil {
		return false, fmt.Errorf("adversary damage: %w", err)
	}
	result = downgradeDamageResult(result, optionalInt(args, "severity_downgrade", 0))

	var app rules.DamageApplication
	if optionalBool(args, "direct", false) {
		app, err = rules.ApplyDamage(hpBefore, adjusted, majorThreshold, severeThreshold, options)
		if err != nil {
			return false, fmt.Errorf("adversary damage: %w", err)
		}
		app.Result = result
		_, app.HPAfter = rules.ApplyDamageMarks(hpBefore, app.Result.Marks)
	} else {
		app = rules.ApplyDamageWithArmor(hpBefore, 0, armorBefore, result, rules.ArmorDamageRules{})
	}
	if app.HPAfter >= hpBefore && app.ArmorAfter >= armorBefore {
		if err := r.assertf("expected damage to affect hp or armor for %s", name); err != nil {
			return false, err
		}
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	if _, err := r.env.daggerheartClient.ApplyAdversaryDamage(ctxWithSession, &daggerheartv1.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  state.campaignID,
		SceneId:     state.activeSceneID,
		AdversaryId: adversaryID,
		Damage: buildDamageRequest(
			args,
			"",
			optionalString(args, "source", "attack"),
			damageRoll.GetTotal(),
		),
		RequireDamageRoll: false,
	}); err != nil {
		return false, fmt.Errorf("apply adversary damage: %w", err)
	}
	after, err := r.getAdversary(ctx, state, adversaryID)
	if err != nil {
		return false, err
	}
	if after.GetHp() >= before.GetHp() && after.GetArmor() >= before.GetArmor() {
		if err := r.assertf("expected damage to affect hp or armor for %s", name); err != nil {
			return false, err
		}
	}
	return true, nil
}

func downgradeDamageResult(result rules.DamageResult, steps int) rules.DamageResult {
	if steps <= 0 || result.Marks <= 0 {
		return result
	}
	downgraded := result
	for i := 0; i < steps && downgraded.Marks > 0; i++ {
		if downgraded.Severity > rules.DamageNone {
			downgraded.Severity--
		}
		downgraded.Marks--
	}
	return downgraded
}

func parseDamageType(value string) daggerheartv1.DaggerheartDamageType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "magic":
		return daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "mixed":
		return daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	}
}

func damageTypesForArgs(args map[string]any) rules.DamageTypes {
	switch parseDamageType(optionalString(args, "damage_type", "physical")) {
	case daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		return rules.DamageTypes{Magic: true}
	case daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		return rules.DamageTypes{Physical: true, Magic: true}
	default:
		return rules.DamageTypes{Physical: true}
	}
}

func adjustedDamageAmount(args map[string]any, amount int32) int {
	resistance := rules.ResistanceProfile{
		ResistPhysical: optionalBool(args, "resist_physical", false),
		ResistMagic:    optionalBool(args, "resist_magic", false),
		ImmunePhysical: optionalBool(args, "immune_physical", false),
		ImmuneMagic:    optionalBool(args, "immune_magic", false),
	}
	return rules.ApplyResistance(int(amount), damageTypesForArgs(args), resistance)
}

func expectDamageEffect(args map[string]any, roll *daggerheartv1.SessionDamageRollResponse) bool {
	if roll == nil {
		return false
	}
	return adjustedDamageAmount(args, roll.GetTotal()) > 0
}

func parseConditions(values []string) ([]daggerheartv1.DaggerheartCondition, error) {
	result := make([]daggerheartv1.DaggerheartCondition, 0, len(values))
	for _, value := range values {
		switch strings.ToUpper(strings.TrimSpace(value)) {
		case "VULNERABLE":
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		case "RESTRAINED":
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED)
		case "HIDDEN":
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		default:
			return nil, fmt.Errorf("unknown condition %q", value)
		}
	}
	return result, nil
}

func parseConditionStates(values []string) ([]*daggerheartv1.DaggerheartConditionState, error) {
	conditions, err := parseConditions(values)
	if err != nil {
		return nil, err
	}
	result := make([]*daggerheartv1.DaggerheartConditionState, 0, len(conditions))
	for _, condition := range conditions {
		code, err := standardConditionCode(condition)
		if err != nil {
			return nil, err
		}
		result = append(result, &daggerheartv1.DaggerheartConditionState{
			Id:       code,
			Code:     code,
			Label:    code,
			Class:    daggerheartv1.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: condition,
		})
	}
	return result, nil
}

func parseConditionIDs(values []string) ([]string, error) {
	conditions, err := parseConditions(values)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		code, err := standardConditionCode(condition)
		if err != nil {
			return nil, err
		}
		result = append(result, code)
	}
	return result, nil
}

func standardConditionCode(condition daggerheartv1.DaggerheartCondition) (string, error) {
	switch condition {
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
		return "vulnerable", nil
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
		return "restrained", nil
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
		return "hidden", nil
	case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED:
		return "cloaked", nil
	default:
		return "", fmt.Errorf("unknown condition %v", condition)
	}
}

func parseGameSystem(value string) (commonv1.GameSystem, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unsupported system %q", value)
	}
	if !strings.HasPrefix(normalized, "GAME_SYSTEM_") {
		normalized = "GAME_SYSTEM_" + normalized
	}
	number, ok := commonv1.GameSystem_value[normalized]
	if !ok {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unsupported system %q", strings.TrimPrefix(normalized, "GAME_SYSTEM_"))
	}
	parsed := commonv1.GameSystem(number)
	if parsed == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, fmt.Errorf("unsupported system %q", strings.TrimPrefix(normalized, "GAME_SYSTEM_"))
	}
	return parsed, nil
}

func parseGmMode(value string) (gamev1.GmMode, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return gamev1.GmMode_HUMAN, nil
	case "AI":
		return gamev1.GmMode_AI, nil
	case "HYBRID":
		return gamev1.GmMode_HYBRID, nil
	default:
		return gamev1.GmMode_GM_MODE_UNSPECIFIED, fmt.Errorf("unsupported gm_mode %q", value)
	}
}

func parseCampaignIntent(value string) (gamev1.CampaignIntent, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "STANDARD":
		return gamev1.CampaignIntent_STANDARD, nil
	case "STARTER":
		return gamev1.CampaignIntent_STARTER, nil
	case "SANDBOX":
		return gamev1.CampaignIntent_SANDBOX, nil
	default:
		return gamev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED, fmt.Errorf("unsupported intent %q", value)
	}
}

func parseCampaignAccessPolicy(value string) (gamev1.CampaignAccessPolicy, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PRIVATE":
		return gamev1.CampaignAccessPolicy_PRIVATE, nil
	case "RESTRICTED":
		return gamev1.CampaignAccessPolicy_RESTRICTED, nil
	case "PUBLIC":
		return gamev1.CampaignAccessPolicy_PUBLIC, nil
	default:
		return gamev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED, fmt.Errorf("unsupported access policy %q", value)
	}
}

func parseCharacterKind(value string) (gamev1.CharacterKind, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return gamev1.CharacterKind_PC, nil
	case "NPC":
		return gamev1.CharacterKind_NPC, nil
	default:
		return gamev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, fmt.Errorf("unsupported character kind %q", value)
	}
}

func parseParticipantRole(value string) (gamev1.ParticipantRole, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PLAYER", "":
		return gamev1.ParticipantRole_PLAYER, nil
	case "GM":
		return gamev1.ParticipantRole_GM, nil
	default:
		return gamev1.ParticipantRole_ROLE_UNSPECIFIED, fmt.Errorf("unsupported participant role %q", value)
	}
}

func parseController(value string) (gamev1.Controller, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN", "":
		return gamev1.Controller_CONTROLLER_HUMAN, nil
	case "AI":
		return gamev1.Controller_CONTROLLER_AI, nil
	default:
		return gamev1.Controller_CONTROLLER_UNSPECIFIED, fmt.Errorf("unsupported controller %q", value)
	}
}

func parseOwnership(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "", nil
	}
	switch trimmed {
	case "participant", "unassigned":
		return trimmed, nil
	default:
		return "", fmt.Errorf("unsupported owner %q", value)
	}
}

func prefabOptions(name string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "frodo":
		return map[string]any{
			"kind":             "PC",
			"armor":            1,
			"hp_max":           6,
			"stress_max":       6,
			"evasion":          10,
			"major_threshold":  3,
			"severe_threshold": 6,
		}
	default:
		return map[string]any{"kind": "PC"}
	}
}

func actorID(state *scenarioState, name string) (string, error) {
	id, ok := state.actors[name]
	if !ok {
		for key, value := range state.actors {
			if strings.EqualFold(key, name) {
				id = value
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("unknown actor %q", name)
	}
	return id, nil
}

func participantID(state *scenarioState, name string) (string, error) {
	id, ok := state.participants[name]
	if !ok {
		for key, value := range state.participants {
			if strings.EqualFold(key, name) {
				id = value
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("unknown participant %q", name)
	}
	return id, nil
}

func adversaryID(state *scenarioState, name string) (string, error) {
	id, ok := state.adversaries[name]
	if !ok {
		for key, value := range state.adversaries {
			if strings.EqualFold(key, name) {
				id = value
				ok = true
				break
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("unknown adversary %q", name)
	}
	return id, nil
}

func resolveTargetID(state *scenarioState, name string) (string, bool, error) {
	if id, ok := state.actors[name]; ok {
		return id, false, nil
	}
	if id, ok := state.adversaries[name]; ok {
		return id, true, nil
	}
	for key, value := range state.actors {
		if strings.EqualFold(key, name) {
			return value, false, nil
		}
	}
	for key, value := range state.adversaries {
		if strings.EqualFold(key, name) {
			return value, true, nil
		}
	}
	return "", false, fmt.Errorf("unknown target %q", name)
}

func resolveCountdownID(state *scenarioState, args map[string]any) (string, error) {
	if countdownID := optionalString(args, "countdown_id", ""); countdownID != "" {
		return countdownID, nil
	}
	name := optionalString(args, "name", "")
	if name == "" {
		return "", nil
	}
	countdownID, ok := state.countdowns[name]
	if !ok {
		return "", fmt.Errorf("unknown countdown %q", name)
	}
	return countdownID, nil
}

func resolveCountdownReference(state *scenarioState, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if countdownID, ok := state.countdowns[value]; ok {
		return countdownID
	}
	return value
}

func resolveOutcomeTargets(state *scenarioState, args map[string]any) ([]string, error) {
	list := readStringSlice(args, "targets")
	if len(list) == 0 {
		if name := optionalString(args, "target", ""); name != "" {
			list = []string{name}
		}
	}
	if len(list) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, err := actorID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func resolveAttackTargets(state *scenarioState, args map[string]any) ([]string, error) {
	list := readStringSlice(args, "targets")
	if len(list) == 0 {
		if name := optionalString(args, "target", ""); name != "" {
			list = []string{name}
		}
	}
	if len(list) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, _, err := resolveTargetID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func requireDamageDice(args map[string]any, context string) error {
	value, ok := args["damage_dice"]
	if !ok {
		return fmt.Errorf("%s requires damage_dice", context)
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return fmt.Errorf("%s damage_dice must be a list", context)
	}
	return nil
}

func requiredString(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if ok && text != "" {
		return text
	}
	return ""
}

func readInt(args map[string]any, key string) (int, bool) {
	value, ok := args[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func optionalString(args map[string]any, key, fallback string) string {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if ok && text != "" {
		return text
	}
	return fallback
}

func optionalInt(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func optionalBool(args map[string]any, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		if lower == "true" || lower == "yes" || lower == "1" {
			return true
		}
		if lower == "false" || lower == "no" || lower == "0" {
			return false
		}
	}
	return fallback
}

func ensureRollOutcomeState(state *scenarioState) {
	if state.rollOutcomes == nil {
		state.rollOutcomes = map[uint64]actionRollResult{}
	}
}

func readBool(args map[string]any, key string) (bool, bool) {
	value, ok := args[key]
	if !ok {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "true", "yes", "1":
			return true, true
		case "false", "no", "0":
			return false, true
		}
	}
	return false, false
}

type expectedDeltas struct {
	name        string
	characterID string
	hpDelta     *int
	hopeDelta   *int
	stressDelta *int
	armorDelta  *int
}

type expectedAdversaryDeltas struct {
	name        string
	adversaryID string
	hpDelta     *int
	stressDelta *int
	armorDelta  *int
	deleted     *bool
}

func (r *Runner) captureExpectedDeltas(
	ctx context.Context,
	state *scenarioState,
	args map[string]any,
	fallbackName string,
) (*expectedDeltas, *daggerheartv1.DaggerheartCharacterState, error) {
	hopeDelta, hopeOk := readInt(args, "expect_hope_delta")
	stressDelta, stressOk := readInt(args, "expect_stress_delta")
	hpDelta, hpOk := readInt(args, "expect_hp_delta")
	armorDelta, armorOk := readInt(args, "expect_armor_delta")
	if !hpOk && !hopeOk && !stressOk && !armorOk {
		return nil, nil, nil
	}
	name := optionalString(args, "expect_target", fallbackName)
	if strings.TrimSpace(name) == "" {
		return nil, nil, r.failf("expect_*_delta requires expect_target or a default character")
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return nil, nil, err
	}
	before, err := r.getCharacterState(ctx, state, characterID)
	if err != nil {
		return nil, nil, err
	}
	spec := &expectedDeltas{name: name, characterID: characterID}
	if hpOk {
		spec.hpDelta = &hpDelta
	}
	if hopeOk {
		spec.hopeDelta = &hopeDelta
	}
	if stressOk {
		spec.stressDelta = &stressDelta
	}
	if armorOk {
		spec.armorDelta = &armorDelta
	}
	return spec, before, nil
}

func (r *Runner) captureExpectedAdversaryDeltas(
	ctx context.Context,
	state *scenarioState,
	args map[string]any,
	fallbackName string,
) (*expectedAdversaryDeltas, *daggerheartv1.DaggerheartAdversary, error) {
	hpDelta, hpOk := readInt(args, "expect_adversary_hp_delta")
	stressDelta, stressOk := readInt(args, "expect_adversary_stress_delta")
	armorDelta, armorOk := readInt(args, "expect_adversary_armor_delta")
	deleted, deletedOk := readBool(args, "expect_adversary_deleted")
	if !hpOk && !stressOk && !armorOk && !deletedOk {
		return nil, nil, nil
	}
	name := optionalString(args, "expect_adversary_target", fallbackName)
	if strings.TrimSpace(name) == "" {
		return nil, nil, r.failf("expect_adversary_* requires expect_adversary_target or an adversary target")
	}
	adversaryID, err := adversaryID(state, name)
	if err != nil {
		return nil, nil, err
	}
	before, err := r.getAdversary(ctx, state, adversaryID)
	if err != nil {
		return nil, nil, err
	}
	spec := &expectedAdversaryDeltas{name: name, adversaryID: adversaryID}
	if hpOk {
		spec.hpDelta = &hpDelta
	}
	if stressOk {
		spec.stressDelta = &stressDelta
	}
	if armorOk {
		spec.armorDelta = &armorDelta
	}
	if deletedOk {
		spec.deleted = &deleted
	}
	return spec, before, nil
}

func (r *Runner) assertExpectedDeltas(
	ctx context.Context,
	state *scenarioState,
	spec *expectedDeltas,
	before *daggerheartv1.DaggerheartCharacterState,
) error {
	if spec == nil || before == nil {
		return nil
	}
	after, err := r.getCharacterState(ctx, state, spec.characterID)
	if err != nil {
		return err
	}
	return r.assertExpectedDeltasAfterState(spec, before, after)
}

func (r *Runner) assertExpectedDeltasAfterState(
	spec *expectedDeltas,
	before *daggerheartv1.DaggerheartCharacterState,
	after *daggerheartv1.DaggerheartCharacterState,
) error {
	if spec == nil || before == nil || after == nil {
		return nil
	}
	if spec.hpDelta != nil {
		delta := int(after.GetHp()) - int(before.GetHp())
		if delta != *spec.hpDelta {
			if err := r.assertf("hp delta for %s = %d (before=%d after=%d), want %d", spec.name, delta, before.GetHp(), after.GetHp(), *spec.hpDelta); err != nil {
				return err
			}
		}
	}
	if spec.hopeDelta != nil {
		delta := int(after.GetHope()) - int(before.GetHope())
		if delta != *spec.hopeDelta {
			if err := r.assertf("hope delta for %s = %d, want %d", spec.name, delta, *spec.hopeDelta); err != nil {
				return err
			}
		}
	}
	if spec.stressDelta != nil {
		delta := int(after.GetStress()) - int(before.GetStress())
		if delta != *spec.stressDelta {
			if err := r.assertf("stress delta for %s = %d, want %d", spec.name, delta, *spec.stressDelta); err != nil {
				return err
			}
		}
	}
	if spec.armorDelta != nil {
		delta := int(after.GetArmor()) - int(before.GetArmor())
		if delta != *spec.armorDelta {
			if err := r.assertf("armor delta for %s = %d (before=%d after=%d), want %d", spec.name, delta, before.GetArmor(), after.GetArmor(), *spec.armorDelta); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) assertExpectedAdversaryDeltas(
	ctx context.Context,
	state *scenarioState,
	spec *expectedAdversaryDeltas,
	before *daggerheartv1.DaggerheartAdversary,
) error {
	if spec == nil || before == nil {
		return nil
	}
	after, err := r.getAdversary(ctx, state, spec.adversaryID)
	if spec.deleted != nil && *spec.deleted {
		if err == nil {
			return r.assertf("expected adversary %s to be deleted", spec.name)
		}
		if status.Code(err) != codes.NotFound {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	if spec.deleted != nil && !*spec.deleted {
		if after == nil {
			return r.assertf("expected adversary %s to remain present", spec.name)
		}
	}
	if spec.hpDelta != nil {
		delta := int(after.GetHp()) - int(before.GetHp())
		if delta != *spec.hpDelta {
			if err := r.assertf("adversary hp delta for %s = %d (before=%d after=%d), want %d", spec.name, delta, before.GetHp(), after.GetHp(), *spec.hpDelta); err != nil {
				return err
			}
		}
	}
	if spec.stressDelta != nil {
		delta := int(after.GetStress()) - int(before.GetStress())
		if delta != *spec.stressDelta {
			if err := r.assertf("adversary stress delta for %s = %d (before=%d after=%d), want %d", spec.name, delta, before.GetStress(), after.GetStress(), *spec.stressDelta); err != nil {
				return err
			}
		}
	}
	if spec.armorDelta != nil {
		delta := int(after.GetArmor()) - int(before.GetArmor())
		if delta != *spec.armorDelta {
			if err := r.assertf("adversary armor delta for %s = %d (before=%d after=%d), want %d", spec.name, delta, before.GetArmor(), after.GetArmor(), *spec.armorDelta); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) assertExpectedSpotlight(ctx context.Context, state *scenarioState, args map[string]any) error {
	expected := strings.ToLower(strings.TrimSpace(optionalString(args, "expect_spotlight", "")))
	if expected == "" {
		return nil
	}
	if state.sessionID == "" {
		return r.failf("expect_spotlight requires an active session")
	}
	request := &gamev1.GetSessionSpotlightRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
	}
	if expected == "none" {
		if _, err := r.env.sessionClient.GetSessionSpotlight(ctx, request); err == nil {
			return r.failf("expected no session spotlight")
		}
		return nil
	}
	response, err := r.env.sessionClient.GetSessionSpotlight(ctx, request)
	if err != nil {
		return fmt.Errorf("get session spotlight: %w", err)
	}
	spotlight := response.GetSpotlight()
	if spotlight == nil {
		return r.failf("expected session spotlight")
	}
	if expected == "gm" {
		if spotlight.GetType() != gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM {
			return r.failf("spotlight type = %v, want GM", spotlight.GetType())
		}
		if spotlight.GetCharacterId() != "" {
			return r.failf("spotlight character id = %q, want empty", spotlight.GetCharacterId())
		}
		return nil
	}
	characterID, err := actorID(state, expected)
	if err != nil {
		return err
	}
	if spotlight.GetType() != gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER {
		return r.failf("spotlight type = %v, want CHARACTER", spotlight.GetType())
	}
	if spotlight.GetCharacterId() != characterID {
		return r.failf("spotlight character id = %q, want %q", spotlight.GetCharacterId(), characterID)
	}
	return nil
}

type damageFlagExpect struct {
	resistPhysical *bool
	resistMagic    *bool
	immunePhysical *bool
	immuneMagic    *bool
}

func readDamageFlagExpect(args map[string]any) (damageFlagExpect, bool) {
	expect := damageFlagExpect{}
	if value, ok := readBool(args, "resist_physical"); ok {
		expect.resistPhysical = &value
	}
	if value, ok := readBool(args, "resist_magic"); ok {
		expect.resistMagic = &value
	}
	if value, ok := readBool(args, "immune_physical"); ok {
		expect.immunePhysical = &value
	}
	if value, ok := readBool(args, "immune_magic"); ok {
		expect.immuneMagic = &value
	}
	if expect.resistPhysical == nil && expect.resistMagic == nil && expect.immunePhysical == nil && expect.immuneMagic == nil {
		return damageFlagExpect{}, false
	}
	return expect, true
}

func (r *Runner) assertDamageFlags(
	ctx context.Context,
	state *scenarioState,
	before uint64,
	targetID string,
	args map[string]any,
) error {
	expect, ok := readDamageFlagExpect(args)
	if !ok {
		return nil
	}
	filter := fmt.Sprintf("type = \"%s\"", daggerheartpayload.EventTypeDamageApplied)
	if state.sessionID != "" {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	eventCtx := withParticipantID(ctx, state.ownerParticipantID)
	response, err := r.env.eventClient.ListEvents(eventCtx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return fmt.Errorf("list damage events: %w", err)
	}
	var payload daggerheartpayload.DamageAppliedPayload
	for _, evt := range response.GetEvents() {
		if evt.GetSeq() <= before {
			continue
		}
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return fmt.Errorf("decode damage payload: %w", err)
		}
		if targetID != "" && payload.CharacterID.String() != targetID {
			continue
		}
		if expect.resistPhysical != nil && payload.ResistPhysical != *expect.resistPhysical {
			return r.assertf("resist_physical = %v, want %v", payload.ResistPhysical, *expect.resistPhysical)
		}
		if expect.resistMagic != nil && payload.ResistMagic != *expect.resistMagic {
			return r.assertf("resist_magic = %v, want %v", payload.ResistMagic, *expect.resistMagic)
		}
		if expect.immunePhysical != nil && payload.ImmunePhysical != *expect.immunePhysical {
			return r.assertf("immune_physical = %v, want %v", payload.ImmunePhysical, *expect.immunePhysical)
		}
		if expect.immuneMagic != nil && payload.ImmuneMagic != *expect.immuneMagic {
			return r.assertf("immune_magic = %v, want %v", payload.ImmuneMagic, *expect.immuneMagic)
		}
		return nil
	}
	return r.assertf("expected damage_applied after seq %d", before)
}

func isHopeSpendSource(source string) bool {
	normalized := normalizeModifierSource(source)
	switch normalized {
	case "experience", "help", "tag_team", "hope_feature":
		return true
	default:
		return false
	}
}

func normalizeModifierSource(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "-", "_")
	return replacer.Replace(strings.ToLower(trimmed))
}

func readStringSlice(args map[string]any, key string) []string {
	value, ok := args[key]
	if !ok {
		return nil
	}
	switch list := value.(type) {
	case []string:
		results := make([]string, 0, len(list))
		for _, entry := range list {
			trimmed := strings.TrimSpace(entry)
			if trimmed != "" {
				results = append(results, trimmed)
			}
		}
		return results
	case []any:
		results := make([]string, 0, len(list))
		for _, entry := range list {
			text, ok := entry.(string)
			if !ok {
				continue
			}
			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				results = append(results, trimmed)
			}
		}
		return results
	default:
		return nil
	}
}

// readMap returns a nested table-like argument map when present.
func readMap(args map[string]any, key string) map[string]any {
	if args == nil {
		return nil
	}
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	parsed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return parsed
}

func readMapSlice(args map[string]any, key string) []map[string]any {
	if args == nil {
		return nil
	}
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(list))
	for _, entry := range list {
		parsed, ok := entry.(map[string]any)
		if ok {
			result = append(result, parsed)
		}
	}
	return result
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func resolveCharacterList(state *scenarioState, args map[string]any, key string) ([]string, error) {
	list := readStringSlice(args, key)
	if len(list) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, err := actorID(state, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func allActorIDs(state *scenarioState) []string {
	if len(state.actors) == 0 {
		return nil
	}
	names := make([]string, 0, len(state.actors))
	for name := range state.actors {
		names = append(names, name)
	}
	sort.Strings(names)
	ids := make([]string, 0, len(names))
	for _, name := range names {
		ids = append(ids, state.actors[name])
	}
	return ids
}

func parseRestType(value string) (daggerheartv1.DaggerheartRestType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "short":
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT, nil
	case "long":
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG, nil
	default:
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED, fmt.Errorf("unsupported rest type %q", value)
	}
}

func parseCountdownKind(value string) (daggerheartv1.DaggerheartCountdownTone, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "progress":
		return daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS, nil
	case "consequence":
		return daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE, nil
	case "loop", "long_term":
		return daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS, nil
	case "neutral":
		return daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL, nil
	default:
		return daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED, fmt.Errorf("unsupported countdown kind %q", value)
	}
}

func parseCountdownAdvancementPolicy(value string) (daggerheartv1.DaggerheartCountdownAdvancementPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "manual":
		return daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL, nil
	case "action_standard", "action standard":
		return daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD, nil
	case "action_dynamic", "action dynamic":
		return daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC, nil
	case "long_rest", "long rest":
		return daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST, nil
	default:
		return daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED, fmt.Errorf("unsupported countdown advancement policy %q", value)
	}
}

func parseCountdownDirection(value string) (daggerheartv1.DaggerheartCountdownLoopBehavior, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "increase":
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE, nil
	case "decrease":
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START, nil
	default:
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED, fmt.Errorf("unsupported countdown direction %q", value)
	}
}

func parseCountdownLoopBehavior(value string) (daggerheartv1.DaggerheartCountdownLoopBehavior, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE, nil
	case "reset":
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET, nil
	case "reset_increase_start", "reset increase start":
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START, nil
	case "reset_decrease_start", "reset decrease start":
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START, nil
	default:
		return daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED, fmt.Errorf("unsupported countdown loop behavior %q", value)
	}
}

func parseCountdownStatus(value string) (daggerheartv1.DaggerheartCountdownStatus, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE, nil
	case "trigger_pending", "trigger pending":
		return daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING, nil
	default:
		return daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_UNSPECIFIED, fmt.Errorf("unsupported countdown status %q", value)
	}
}

func parseDowntimeMove(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "tend_to_wounds":
		return "tend_to_wounds", nil
	case "clear_stress":
		return "clear_stress", nil
	case "repair_armor":
		return "repair_armor", nil
	case "clear_all_stress":
		return "clear_all_stress", nil
	case "repair_all_armor":
		return "repair_all_armor", nil
	case "prepare":
		return "prepare", nil
	case "tend_to_all_wounds":
		return "tend_to_all_wounds", nil
	case "work_on_project":
		return "work_on_project", nil
	default:
		return "", fmt.Errorf("unsupported downtime move %q", value)
	}
}

func parseTemporaryArmorDuration(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "short_rest", "long_rest", "session", "scene":
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("unsupported temporary_armor duration %q", value)
	}
}

func parseDeathMove(value string) (daggerheartv1.DaggerheartDeathMove, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "blaze_of_glory":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY, nil
	case "avoid_death":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH, nil
	case "risk_it_all":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL, nil
	default:
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED, fmt.Errorf("unsupported death move %q", value)
	}
}

func parseLifeState(value string) (daggerheartv1.DaggerheartLifeState, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alive":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE, nil
	case "unconscious":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS, nil
	case "blaze_of_glory":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY, nil
	case "dead":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD, nil
	default:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED, fmt.Errorf("unsupported life_state %q", value)
	}
}

func withUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.UserIDHeader, userID)
}

func withSessionID(ctx context.Context, sessionID string) context.Context {
	if sessionID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.SessionIDHeader, sessionID)
}

func withCampaignID(ctx context.Context, campaignID string) context.Context {
	if campaignID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.CampaignIDHeader, campaignID)
}

func withParticipantID(ctx context.Context, participantID string) context.Context {
	if participantID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, grpcmeta.ParticipantIDHeader, participantID)
}
