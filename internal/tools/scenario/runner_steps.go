package scenario

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/coreevent"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var coreStepKinds = map[string]struct{}{
	"campaign":                              {},
	"participant":                           {},
	"start_session":                         {},
	"end_session":                           {},
	"character":                             {},
	"prefab":                                {},
	"set_spotlight":                         {},
	"clear_spotlight":                       {},
	"create_scene":                          {},
	"end_scene":                             {},
	"scene_add_character":                   {},
	"scene_remove_character":                {},
	"scene_transfer_character":              {},
	"scene_transition":                      {},
	"scene_gate_open":                       {},
	"scene_gate_resolve":                    {},
	"scene_gate_abandon":                    {},
	"scene_set_spotlight":                   {},
	"scene_clear_spotlight":                 {},
	"update_scene":                          {},
	"interaction_set_gm_authority":          {},
	"interaction_set_active_scene":          {},
	"interaction_start_player_phase":        {},
	"interaction_post":                      {},
	"interaction_yield":                     {},
	"interaction_unyield":                   {},
	"interaction_end_player_phase":          {},
	"interaction_accept_player_phase":       {},
	"interaction_request_revisions":         {},
	"interaction_resolve_review":            {},
	"interaction_pause_ooc":                 {},
	"interaction_post_ooc":                  {},
	"interaction_ready_ooc":                 {},
	"interaction_clear_ready_ooc":           {},
	"interaction_resume_ooc":                {},
	"interaction_resolve_interrupted_phase": {},
	"interaction_expect":                    {},
}

func (r *Runner) runStep(ctx context.Context, state *scenarioState, step Step) error {
	expectedErr, err := parseExpectedStepError(step.Args)
	if err != nil {
		return err
	}
	err = r.runStepActual(ctx, state, step)
	if expectedErr == nil {
		return err
	}
	if err == nil {
		return r.failf("expected %s to fail with gRPC code %s", step.Kind, expectedErr.code.String())
	}
	if matchErr := expectedErr.Match(err); matchErr != nil {
		return r.failf("%s", matchErr.Error())
	}
	return nil
}

func (r *Runner) runStepActual(ctx context.Context, state *scenarioState, step Step) error {
	var err error
	ctx, err = r.stepContext(ctx, state, step)
	if err != nil {
		return err
	}
	if _, ok := coreStepKinds[step.Kind]; ok {
		if strings.TrimSpace(step.System) != "" {
			return r.failf("core step %q must not declare a system scope", step.Kind)
		}
		return r.runCoreStep(ctx, state, step)
	}
	known, err := isKnownScenarioSystemStepKind(step.Kind)
	if err != nil {
		return err
	}
	if !known {
		return r.failf("unknown step kind %q", step.Kind)
	}
	return r.runSystemStep(ctx, state, step)
}

type expectedStepError struct {
	code     codes.Code
	contains string
}

func parseExpectedStepError(args map[string]any) (*expectedStepError, error) {
	if args == nil {
		return nil, nil
	}
	raw, ok := args["expect_error"]
	if !ok || raw == nil {
		return nil, nil
	}
	spec, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expect_error must be a table")
	}
	codeText := strings.TrimSpace(optionalString(spec, "code", ""))
	if codeText == "" {
		return nil, fmt.Errorf("expect_error code is required")
	}
	code, ok := parseGRPCCode(codeText)
	if !ok {
		return nil, fmt.Errorf("expect_error code %q is not a recognized gRPC code", codeText)
	}
	return &expectedStepError{
		code:     code,
		contains: strings.TrimSpace(optionalString(spec, "contains", "")),
	}, nil
}

func parseGRPCCode(raw string) (codes.Code, bool) {
	want := normalizeGRPCCodeName(raw)
	for code := codes.OK; code <= codes.Unauthenticated; code++ {
		if normalizeGRPCCodeName(code.String()) == want {
			return code, true
		}
	}
	return codes.OK, false
}

func normalizeGRPCCodeName(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

func (e *expectedStepError) Match(err error) error {
	if e == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("expected gRPC code %s but step returned non-gRPC error: %v", e.code.String(), err)
	}
	if st.Code() != e.code {
		return fmt.Errorf("expected gRPC code %s, got %s: %v", e.code.String(), st.Code().String(), err)
	}
	if e.contains != "" && !strings.Contains(st.Message(), e.contains) {
		return fmt.Errorf("expected gRPC message containing %q for code %s, got %q", e.contains, e.code.String(), st.Message())
	}
	return nil
}

func (r *Runner) runCoreStep(ctx context.Context, state *scenarioState, step Step) error {
	switch step.Kind {
	case "campaign":
		return r.runCampaignStep(ctx, state, step)
	case "participant":
		return r.runParticipantStep(ctx, state, step)
	case "start_session":
		return r.runStartSessionStep(ctx, state, step)
	case "end_session":
		return r.runEndSessionStep(ctx, state)
	case "character":
		return r.runCharacterStep(ctx, state, step)
	case "prefab":
		return r.runPrefabStep(ctx, state, step)
	case "set_spotlight":
		return r.runSetSpotlightStep(ctx, state, step)
	case "clear_spotlight":
		return r.runClearSpotlightStep(ctx, state, step)
	case "create_scene":
		return r.runCreateSceneStep(ctx, state, step)
	case "end_scene":
		return r.runEndSceneStep(ctx, state, step)
	case "scene_add_character":
		return r.runSceneAddCharacterStep(ctx, state, step)
	case "scene_remove_character":
		return r.runSceneRemoveCharacterStep(ctx, state, step)
	case "scene_transfer_character":
		return r.runSceneTransferCharacterStep(ctx, state, step)
	case "scene_transition":
		return r.runSceneTransitionStep(ctx, state, step)
	case "scene_gate_open":
		return r.runSceneGateOpenStep(ctx, state, step)
	case "scene_gate_resolve":
		return r.runSceneGateResolveStep(ctx, state, step)
	case "scene_gate_abandon":
		return r.runSceneGateAbandonStep(ctx, state, step)
	case "scene_set_spotlight":
		return r.runSceneSetSpotlightStep(ctx, state, step)
	case "scene_clear_spotlight":
		return r.runSceneClearSpotlightStep(ctx, state, step)
	case "update_scene":
		return r.runUpdateSceneStep(ctx, state, step)
	case "interaction_set_gm_authority":
		return r.runInteractionSetGMAuthorityStep(ctx, state, step)
	case "interaction_set_active_scene":
		return r.runInteractionSetActiveSceneStep(ctx, state, step)
	case "interaction_start_player_phase":
		return r.runInteractionStartPlayerPhaseStep(ctx, state, step)
	case "interaction_post":
		return r.runInteractionPostStep(ctx, state, step)
	case "interaction_yield":
		return r.runInteractionYieldStep(ctx, state, step)
	case "interaction_unyield":
		return r.runInteractionUnyieldStep(ctx, state, step)
	case "interaction_end_player_phase":
		return r.runInteractionEndPlayerPhaseStep(ctx, state, step)
	case "interaction_accept_player_phase":
		return r.runInteractionAcceptPlayerPhaseStep(ctx, state, step)
	case "interaction_request_revisions":
		return r.runInteractionRequestRevisionsStep(ctx, state, step)
	case "interaction_resolve_review":
		return r.runInteractionResolveReviewStep(ctx, state, step)
	case "interaction_pause_ooc":
		return r.runInteractionPauseOOCStep(ctx, state, step)
	case "interaction_post_ooc":
		return r.runInteractionPostOOCStep(ctx, state, step)
	case "interaction_ready_ooc":
		return r.runInteractionReadyOOCStep(ctx, state)
	case "interaction_clear_ready_ooc":
		return r.runInteractionClearReadyOOCStep(ctx, state)
	case "interaction_resume_ooc":
		return r.runInteractionResumeOOCStep(ctx, state)
	case "interaction_resolve_interrupted_phase":
		return r.runInteractionResolveInterruptedPhaseStep(ctx, state, step)
	case "interaction_expect":
		return r.runInteractionExpectStep(ctx, state, step)
	default:
		return r.failf("unknown core step kind %q", step.Kind)
	}
}

// stepContext resolves the acting participant for one step so scenarios can
// interleave GM and player-owned writes without modal runner state.
func (r *Runner) stepContext(ctx context.Context, state *scenarioState, step Step) (context.Context, error) {
	participantID, err := r.stepParticipantID(state, step)
	if err != nil {
		return nil, err
	}
	return withParticipantID(ctx, participantID), nil
}

// stepParticipantID keeps legacy owner-default behavior while allowing an
// explicit `as` actor override on any step.
func (r *Runner) stepParticipantID(state *scenarioState, step Step) (string, error) {
	alias := strings.TrimSpace(optionalString(step.Args, "as", ""))
	if alias == "" {
		return state.ownerParticipantID, nil
	}
	id, err := participantID(state, alias)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Runner) runSystemStep(ctx context.Context, state *scenarioState, step Step) error {
	systemID, err := r.resolveStepSystem(state, step)
	if err != nil {
		return err
	}
	step.System = systemID
	registration, ok, err := scenarioSystemForID(systemID)
	if err != nil {
		return err
	}
	if !ok {
		return unsupportedScenarioSystemError(systemID)
	}
	if _, ok := registration.stepKinds[step.Kind]; !ok {
		known, kindErr := isKnownScenarioSystemStepKind(step.Kind)
		if kindErr != nil {
			return kindErr
		}
		if !known {
			return r.failf("unknown step kind %q", step.Kind)
		}
		systems, systemsErr := registeredSystemsForStepKind(step.Kind)
		if systemsErr != nil {
			return systemsErr
		}
		return r.failf(
			"step kind %q is not supported for system %q (supported systems: %s)",
			step.Kind,
			systemID,
			strings.Join(systems, ", "),
		)
	}
	if registration.runStep == nil {
		return r.failf("scenario system %q has no step runner", systemID)
	}
	return registration.runStep(r, ctx, state, step)
}

func (r *Runner) runDaggerheartStep(ctx context.Context, state *scenarioState, step Step) error {
	switch step.Kind {
	case "adversary":
		return r.runAdversaryStep(ctx, state, step)
	case "creation_workflow":
		return r.runCreationWorkflowStep(ctx, state, step)
	case "expect_gm_fear":
		return r.runExpectGMFearStep(ctx, state, step)
	case "gm_fear":
		return r.runGMFearStep(ctx, state, step)
	case "reaction":
		return r.runReactionStep(ctx, state, step)
	case "group_reaction":
		return r.runGroupReactionStep(ctx, state, step)
	case "gm_spend_fear":
		return r.runGMSpendFearStep(ctx, state, step)
	case "apply_condition":
		return r.runApplyConditionStep(ctx, state, step)
	case "apply_stat_modifier":
		return r.runApplyStatModifierStep(ctx, state, step)
	case "group_action":
		return r.runGroupActionStep(ctx, state, step)
	case "tag_team":
		return r.runTagTeamStep(ctx, state, step)
	case "temporary_armor":
		return r.runTemporaryArmorStep(ctx, state, step)
	case "rest":
		return r.runRestStep(ctx, state, step)
	case "death_move":
		return r.runDeathMoveStep(ctx, state, step)
	case "blaze_of_glory":
		return r.runBlazeOfGloryStep(ctx, state, step)
	case "attack":
		return r.runAttackStep(ctx, state, step)
	case "multi_attack":
		return r.runMultiAttackStep(ctx, state, step)
	case "combined_damage":
		return r.runCombinedDamageStep(ctx, state, step)
	case "adversary_attack":
		return r.runAdversaryAttackStep(ctx, state, step)
	case "adversary_feature":
		return r.runAdversaryFeatureStep(ctx, state, step)
	case "adversary_reaction":
		return r.runAdversaryReactionStep(ctx, state, step)
	case "adversary_update":
		return r.runAdversaryUpdateStep(ctx, state, step)
	case "swap_loadout":
		return r.runSwapLoadoutStep(ctx, state, step)
	case "countdown_create":
		return r.runCountdownCreateStep(ctx, state, step)
	case "countdown_update":
		return r.runCountdownUpdateStep(ctx, state, step)
	case "countdown_delete":
		return r.runCountdownDeleteStep(ctx, state, step)
	case "action_roll":
		return r.runActionRollStep(ctx, state, step)
	case "reaction_roll":
		return r.runReactionRollStep(ctx, state, step)
	case "damage_roll":
		return r.runDamageRollStep(ctx, state, step)
	case "adversary_attack_roll":
		return r.runAdversaryAttackRollStep(ctx, state, step)
	case "apply_roll_outcome":
		return r.runApplyRollOutcomeStep(ctx, state, step)
	case "apply_attack_outcome":
		return r.runApplyAttackOutcomeStep(ctx, state, step)
	case "apply_adversary_attack_outcome":
		return r.runApplyAdversaryAttackOutcomeStep(ctx, state, step)
	case "apply_reaction_outcome":
		return r.runApplyReactionOutcomeStep(ctx, state, step)
	case "mitigate_damage":
		return r.runMitigateDamageStep(ctx, state, step)
	case "level_up":
		return r.runLevelUpStep(ctx, state, step)
	case "class_feature":
		return r.runClassFeatureStep(ctx, state, step)
	case "update_gold":
		return r.runUpdateGoldStep(ctx, state, step)
	case "acquire_domain_card":
		return r.runAcquireDomainCardStep(ctx, state, step)
	case "swap_equipment":
		return r.runSwapEquipmentStep(ctx, state, step)
	case "use_consumable":
		return r.runUseConsumableStep(ctx, state, step)
	case "acquire_consumable":
		return r.runAcquireConsumableStep(ctx, state, step)
	default:
		return r.failf("unknown Daggerheart step kind %q", step.Kind)
	}
}

func (r *Runner) resolveStepSystem(state *scenarioState, step Step) (string, error) {
	systemID := strings.ToUpper(strings.TrimSpace(step.System))
	if systemID == "" {
		if state.campaignSystem == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
			return "", r.failf("system step %q requires explicit system scope", step.Kind)
		}
		var err error
		systemID, err = registeredScenarioSystemIDForGameSystem(state.campaignSystem)
		if err != nil {
			return "", err
		}
	}
	registeredSystemID, parsed, err := registeredScenarioSystemIDForValue(systemID)
	if err != nil {
		return "", err
	}
	if state.campaignSystem != commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED && state.campaignSystem != parsed {
		return "", r.failf(
			"step system %q does not match campaign system %q",
			registeredSystemID,
			strings.TrimPrefix(state.campaignSystem.String(), "GAME_SYSTEM_"),
		)
	}
	return registeredSystemID, nil
}

func (r *Runner) runParticipantStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("participant name is required")
	}
	roleValue, err := parseParticipantRole(optionalString(step.Args, "role", "PLAYER"))
	if err != nil {
		return err
	}
	controllerValue, err := parseController(optionalString(step.Args, "controller", "HUMAN"))
	if err != nil {
		return err
	}

	request := &gamev1.CreateParticipantRequest{
		CampaignId: state.campaignID,
		Name:       name,
		Role:       roleValue,
		Controller: controllerValue,
	}
	if userID := optionalString(step.Args, "user_id", ""); userID != "" {
		request.UserId = userID
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithOwner := withParticipantID(ctx, state.ownerParticipantID)
	response, err := r.env.participantClient.CreateParticipant(ctxWithOwner, request)
	if err != nil {
		return fmt.Errorf("create participant: %w", err)
	}
	if response.GetParticipant() == nil {
		return r.failf("expected participant")
	}
	state.participants[name] = response.GetParticipant().GetId()
	r.logf("participant created: name=%s id=%s role=%s controller=%s", name, state.participants[name], roleValue.String(), controllerValue.String())
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeParticipantJoined)
}

func (r *Runner) runCampaignStep(ctx context.Context, state *scenarioState, step Step) error {
	if state.campaignID != "" {
		return r.failf("campaign already created")
	}
	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("campaign name is required")
	}
	system := strings.TrimSpace(requiredString(step.Args, "system"))
	if system == "" {
		return r.failf("campaign system is required")
	}
	gmMode := optionalString(step.Args, "gm_mode", "HUMAN")
	intent := optionalString(step.Args, "intent", "SANDBOX")
	accessPolicy := optionalString(step.Args, "access_policy", "PRIVATE")

	systemValue, err := parseGameSystem(system)
	if err != nil {
		return err
	}
	if _, err := registeredScenarioSystemIDForGameSystem(systemValue); err != nil {
		return err
	}
	gmModeValue, err := parseGmMode(gmMode)
	if err != nil {
		return err
	}
	intentValue, err := parseCampaignIntent(intent)
	if err != nil {
		return err
	}
	accessPolicyValue, err := parseCampaignAccessPolicy(accessPolicy)
	if err != nil {
		return err
	}

	request := &gamev1.CreateCampaignRequest{
		Name:         name,
		System:       systemValue,
		GmMode:       gmModeValue,
		Intent:       intentValue,
		AccessPolicy: accessPolicyValue,
	}
	if theme := optionalString(step.Args, "theme", ""); theme != "" {
		request.ThemePrompt = theme
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.campaignClient.CreateCampaign(withUserID(ctx, state.userID), request)
	if err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}
	if response.GetCampaign() == nil {
		return r.failf("expected campaign response")
	}
	state.campaignID = response.GetCampaign().GetId()
	if response.GetOwnerParticipant() == nil {
		return r.failf("expected owner participant")
	}
	state.campaignSystem = response.GetCampaign().GetSystem()
	if state.campaignSystem == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		state.campaignSystem = systemValue
	}
	state.ownerParticipantID = response.GetOwnerParticipant().GetId()
	r.logf("campaign created: id=%s owner_participant=%s", state.campaignID, state.ownerParticipantID)
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeCampaignCreated)
}

func (r *Runner) runStartSessionStep(ctx context.Context, state *scenarioState, step Step) error {
	if state.campaignID == "" {
		return r.failf("campaign is required before session")
	}
	if state.sessionID != "" {
		if state.sessionImplicit {
			state.sessionImplicit = false
			r.logf("session adopted: id=%s name=%s", state.sessionID, optionalString(step.Args, "name", "Scenario Session"))
			return nil
		}
		return r.failf("session is already started")
	}
	if err := r.ensureSessionStartReadiness(ctx, state); err != nil {
		return err
	}
	name := optionalString(step.Args, "name", "Scenario Session")
	request := &gamev1.StartSessionRequest{CampaignId: state.campaignID, Name: name}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.sessionClient.StartSession(ctx, request)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	if response.GetSession() == nil {
		return r.failf("expected session")
	}
	state.sessionID = response.GetSession().GetId()
	state.sessionImplicit = false
	r.logf("session started: id=%s name=%s", state.sessionID, name)
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeSessionStarted)
}

func (r *Runner) runEndSessionStep(ctx context.Context, state *scenarioState) error {
	if state.sessionID == "" {
		return r.failf("session is required to end")
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.sessionClient.EndSession(ctx, &gamev1.EndSessionRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
	})
	if err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, event.TypeSessionEnded); err != nil {
		return err
	}
	r.logf("session ended: id=%s", state.sessionID)
	state.sessionID = ""
	state.sessionImplicit = false
	return nil
}

func (r *Runner) runCharacterStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}

	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("character name is required")
	}
	kind := optionalString(step.Args, "kind", "PC")
	parsedKind, err := parseCharacterKind(kind)
	if err != nil {
		return err
	}
	request := &gamev1.CreateCharacterRequest{
		CampaignId: state.campaignID,
		Name:       name,
		Kind:       parsedKind,
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.characterClient.CreateCharacter(ctx, request)
	if err != nil {
		return fmt.Errorf("create character: %w", err)
	}
	if response.GetCharacter() == nil {
		return r.failf("expected character")
	}
	characterID := response.GetCharacter().GetId()
	state.actors[name] = characterID
	r.logf("character created: name=%s id=%s kind=%s", name, characterID, parsedKind.String())
	if !optionalBool(step.Args, "skip_system_readiness", false) {
		if err := r.ensureScenarioCharacterReadiness(ctx, state, characterID, step.Args); err != nil {
			return err
		}
	}

	expectedProfile, err := r.applyDefaultDaggerheartProfile(ctx, state, characterID, step.Args)
	if err != nil {
		return err
	}
	expectedState, err := r.applyOptionalCharacterState(ctx, state, characterID, step.Args, expectedProfile)
	if err != nil {
		return err
	}
	if expectedProfile != nil || expectedState != nil {
		if err := r.waitForDaggerheartCharacterProjection(ctx, state, characterID, expectedProfile, expectedState); err != nil {
			return err
		}
	}

	if control := optionalString(step.Args, "control", ""); control != "" {
		mode, err := parseControl(control)
		if err != nil {
			return err
		}
		switch mode {
		case "participant":
			participantName := optionalString(step.Args, "participant", "")
			if participantName == "" {
				return r.failf("character control participant is required")
			}
			participantID, ok := state.participants[participantName]
			if !ok {
				return r.failf("unknown participant %q", participantName)
			}
			_, err := r.env.characterClient.UpdateCharacter(withParticipantID(ctx, state.ownerParticipantID), &gamev1.UpdateCharacterRequest{
				CampaignId:         state.campaignID,
				CharacterId:        characterID,
				OwnerParticipantId: wrapperspb.String(participantID),
			})
			if err != nil {
				return fmt.Errorf("set character owner participant: %w", err)
			}
			_, err = r.env.characterClient.SetDefaultControl(withParticipantID(ctx, state.ownerParticipantID), &gamev1.SetDefaultControlRequest{
				CampaignId:    state.campaignID,
				CharacterId:   characterID,
				ParticipantId: wrapperspb.String(participantID),
			})
			if err != nil {
				return fmt.Errorf("set default control: %w", err)
			}
			r.logf("character control: name=%s control=participant participant=%s", name, participantName)
		case "gm", "none":
			_, err := r.env.characterClient.SetDefaultControl(withParticipantID(ctx, state.ownerParticipantID), &gamev1.SetDefaultControlRequest{
				CampaignId:  state.campaignID,
				CharacterId: characterID,
			})
			if err != nil {
				return fmt.Errorf("clear default control: %w", err)
			}
			r.logf("character control: name=%s control=%s", name, mode)
		}
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeCharacterCreated)
}

func (r *Runner) runPrefabStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("prefab name is required")
	}
	options := prefabOptions(name)
	step.Args["name"] = name
	for key, value := range options {
		step.Args[key] = value
	}
	return r.runCharacterStep(ctx, state, step)
}
