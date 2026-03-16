package scenario

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/coreevent"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (r *Runner) runAdversaryStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}

	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("adversary name is required")
	}
	if err := r.ensureDefaultScene(ctx, state); err != nil {
		return err
	}
	adversaryEntryID, err := r.resolveAdversaryEntryID(ctx, step.Args, name)
	if err != nil {
		return err
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	request := &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId:       state.campaignID,
		SessionId:        state.sessionID,
		SceneId:          state.activeSceneID,
		AdversaryEntryId: adversaryEntryID,
	}
	response, err := r.env.daggerheartClient.CreateAdversary(ctx, request)
	if err != nil {
		return fmt.Errorf("create adversary: %w", err)
	}
	if response.GetAdversary() == nil {
		return r.failf("expected adversary")
	}
	state.adversaries[name] = response.GetAdversary().GetId()
	r.logf("adversary created: name=%s id=%s", name, state.adversaries[name])
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryCreated)
}

func (r *Runner) ensureDefaultScene(ctx context.Context, state *scenarioState) error {
	if strings.TrimSpace(state.activeSceneID) != "" {
		return nil
	}
	characters := make([]string, 0, len(state.actors))
	for name := range state.actors {
		characters = append(characters, name)
	}
	sort.Strings(characters)
	return r.runCreateSceneStep(ctx, state, Step{
		Kind: "create_scene",
		Args: map[string]any{
			"name":       "Scenario Scene",
			"characters": characters,
		},
	})
}

func (r *Runner) resolveAdversaryEntryID(ctx context.Context, args map[string]any, name string) (string, error) {
	if entryID := strings.TrimSpace(optionalString(args, "adversary_entry_id", "")); entryID != "" {
		return entryID, nil
	}
	if r.env.resolveDaggerheartAdversaryEntryID == nil {
		return "", fmt.Errorf("adversary entry id is required for %s", name)
	}
	entryID, err := r.env.resolveDaggerheartAdversaryEntryID(ctx, name)
	if err != nil {
		return "", err
	}
	return entryID, nil
}

// runCreationWorkflowStep applies an explicit Daggerheart creation workflow so
// scenarios can exercise real creation rules instead of only the readiness
// shortcut.
func (r *Runner) runCreationWorkflowStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	targetName := requiredString(step.Args, "target")
	if targetName == "" {
		return r.failf("creation_workflow target is required")
	}
	characterID, err := actorID(state, targetName)
	if err != nil {
		return err
	}
	input := buildScenarioDaggerheartWorkflowInput(step.Args)
	_, err = r.env.characterClient.ApplyCharacterCreationWorkflow(ctx, &gamev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemWorkflow: &gamev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: input,
		},
	})
	if err != nil {
		return fmt.Errorf("apply creation workflow: %w", err)
	}
	return r.assertCreationWorkflowExpectations(ctx, state, characterID, step.Args)
}

// assertCreationWorkflowExpectations checks only the durable fields a scenario
// explicitly asked to prove after a creation workflow apply.
func (r *Runner) assertCreationWorkflowExpectations(
	ctx context.Context,
	state *scenarioState,
	characterID string,
	args map[string]any,
) error {
	if !hasCreationWorkflowExpectations(args) {
		return nil
	}
	profile, err := r.getDaggerheartProfile(ctx, state, characterID)
	if err != nil {
		return err
	}
	if want := strings.TrimSpace(optionalString(args, "expect_class_id", "")); want != "" && profile.GetClassId() != want {
		return r.assertf("creation_workflow class_id = %q, want %q", profile.GetClassId(), want)
	}
	if want := strings.TrimSpace(optionalString(args, "expect_subclass_id", "")); want != "" && profile.GetSubclassId() != want {
		return r.assertf("creation_workflow subclass_id = %q, want %q", profile.GetSubclassId(), want)
	}
	heritage := profile.GetHeritage()
	if want := strings.TrimSpace(optionalString(args, "expect_heritage_label", "")); want != "" {
		if heritage == nil || heritage.GetAncestryLabel() != want {
			return r.assertf("creation_workflow heritage_label = %q, want %q", heritage.GetAncestryLabel(), want)
		}
	}
	if want := strings.TrimSpace(optionalString(args, "expect_first_feature_ancestry_id", "")); want != "" {
		if heritage == nil || heritage.GetFirstFeatureAncestryId() != want {
			return r.assertf("creation_workflow first_feature_ancestry_id = %q, want %q", heritage.GetFirstFeatureAncestryId(), want)
		}
	}
	if want := strings.TrimSpace(optionalString(args, "expect_second_feature_ancestry_id", "")); want != "" {
		if heritage == nil || heritage.GetSecondFeatureAncestryId() != want {
			return r.assertf("creation_workflow second_feature_ancestry_id = %q, want %q", heritage.GetSecondFeatureAncestryId(), want)
		}
	}
	if want := strings.TrimSpace(optionalString(args, "expect_community_id", "")); want != "" {
		if heritage == nil || heritage.GetCommunityId() != want {
			return r.assertf("creation_workflow community_id = %q, want %q", heritage.GetCommunityId(), want)
		}
	}
	if want, ok := readBool(args, "expect_companion_present"); ok {
		got := profile.GetCompanionSheet() != nil
		if got != want {
			return r.assertf("creation_workflow companion_present = %t, want %t", got, want)
		}
	}
	companion := profile.GetCompanionSheet()
	if want := strings.TrimSpace(optionalString(args, "expect_companion_name", "")); want != "" {
		if companion == nil || companion.GetName() != want {
			return r.assertf("creation_workflow companion_name = %q, want %q", companion.GetName(), want)
		}
	}
	if want := strings.TrimSpace(optionalString(args, "expect_companion_animal_kind", "")); want != "" {
		if companion == nil || companion.GetAnimalKind() != want {
			return r.assertf("creation_workflow companion_animal_kind = %q, want %q", companion.GetAnimalKind(), want)
		}
	}
	if want := strings.TrimSpace(optionalString(args, "expect_companion_damage_type", "")); want != "" {
		if companion == nil || companion.GetDamageType() != want {
			return r.assertf("creation_workflow companion_damage_type = %q, want %q", companion.GetDamageType(), want)
		}
	}
	return nil
}

// hasCreationWorkflowExpectations reports whether a scenario asked the runner
// to verify persisted creation fields after applying the workflow.
func hasCreationWorkflowExpectations(args map[string]any) bool {
	for _, key := range []string{
		"expect_class_id",
		"expect_subclass_id",
		"expect_heritage_label",
		"expect_first_feature_ancestry_id",
		"expect_second_feature_ancestry_id",
		"expect_community_id",
		"expect_companion_present",
		"expect_companion_name",
		"expect_companion_animal_kind",
		"expect_companion_damage_type",
	} {
		if _, ok := args[key]; ok {
			return true
		}
	}
	return false
}

func (r *Runner) runGMFearStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	value, ok := readInt(step.Args, "value")
	if !ok {
		return r.failf("gm_fear value is required")
	}
	_, err := r.env.snapshotClient.UpdateSnapshotState(ctx, &gamev1.UpdateSnapshotStateRequest{
		CampaignId: state.campaignID,
		SystemSnapshotUpdate: &gamev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: int32(value)},
		},
	})
	if err != nil {
		return fmt.Errorf("update snapshot: %w", err)
	}
	snapshot, err := r.getSnapshot(ctx, state)
	if err != nil {
		return err
	}
	if snapshot.GetGmFear() != int32(value) {
		if err := r.assertf("gm_fear = %d, want %d", snapshot.GetGmFear(), value); err != nil {
			return err
		}
	}
	state.gmFear = value
	return nil
}

// runExpectGMFearStep asserts the current Daggerheart GM Fear pool without
// mutating it.
func (r *Runner) runExpectGMFearStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	value, ok := readInt(step.Args, "value")
	if !ok {
		return r.failf("expect_gm_fear value is required")
	}
	snapshot, err := r.getSnapshot(ctx, state)
	if err != nil {
		return err
	}
	if int(snapshot.GetGmFear()) != value {
		return r.assertf("expect_gm_fear = %d, want %d", snapshot.GetGmFear(), value)
	}
	state.gmFear = value
	return nil
}

func (r *Runner) runReactionStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("reaction requires actor")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	seed := uint64(optionalInt(step.Args, "seed", 0))
	if seed == 0 {
		seedValue, err := chooseActionSeed(step.Args, difficulty)
		if err != nil {
			return err
		}
		seed = seedValue
	}

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, actorName)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	actorIDValue, err := actorID(state, actorName)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionReactionFlow(ctx, &daggerheartv1.SessionReactionFlowRequest{
		CampaignId:   state.campaignID,
		SessionId:    state.sessionID,
		SceneId:      state.activeSceneID,
		CharacterId:  actorIDValue,
		Trait:        trait,
		Difficulty:   int32(difficulty),
		Advantage:    int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage: int32(optionalInt(step.Args, "disadvantage", 0)),
		Modifiers:    buildActionRollModifiers(step.Args, "modifiers"),
		ReactionRng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("reaction flow: %w", err)
	}
	if response.GetActionRoll() == nil {
		return r.failf("expected reaction action roll")
	}
	state.lastRollSeq = response.GetActionRoll().GetRollSeq()
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runGroupReactionStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	targetNames := uniqueNonEmptyStrings(readStringSlice(step.Args, "targets"))
	if len(targetNames) == 0 {
		return r.failf("group_reaction requires targets")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	baseSeed := optionalInt(step.Args, "seed", 0)
	failureConditions := readStringSlice(step.Args, "failure_conditions")
	failureSource := optionalString(step.Args, "source", "group_reaction")
	damageAmount := optionalInt(step.Args, "damage", 0)
	if damageAmount < 0 {
		return r.failf("group_reaction damage must be non-negative")
	}
	halfDamageOnSuccess := optionalBool(step.Args, "half_damage_on_success", false)
	damageSource := optionalString(step.Args, "damage_source", failureSource)

	for index, targetName := range targetNames {
		rollArgs := map[string]any{
			"actor":        targetName,
			"trait":        trait,
			"difficulty":   difficulty,
			"advantage":    optionalInt(step.Args, "advantage", 0),
			"disadvantage": optionalInt(step.Args, "disadvantage", 0),
		}
		if modifiersRaw, ok := step.Args["modifiers"]; ok {
			rollArgs["modifiers"] = modifiersRaw
		}
		if outcome := optionalString(step.Args, "outcome", ""); outcome != "" {
			rollArgs["outcome"] = outcome
		}
		if baseSeed > 0 {
			rollArgs["seed"] = baseSeed + index
		}
		if err := r.runReactionRollStep(ctx, state, Step{Kind: "reaction_roll", Args: rollArgs}); err != nil {
			return err
		}

		rollSeq := state.lastRollSeq
		if err := r.runApplyReactionOutcomeStep(ctx, state, Step{
			Kind: "apply_reaction_outcome",
			Args: map[string]any{"roll_seq": int(rollSeq)},
		}); err != nil {
			return err
		}

		ensureRollOutcomeState(state)
		result, ok := state.rollOutcomes[rollSeq]
		if !ok {
			return r.failf("missing action roll outcome for roll_seq %d", rollSeq)
		}
		if !result.success && len(failureConditions) > 0 {
			add := make([]any, 0, len(failureConditions))
			for _, value := range failureConditions {
				add = append(add, value)
			}
			conditionArgs := map[string]any{
				"target": targetName,
				"add":    add,
				"source": failureSource,
			}
			if err := r.runApplyConditionStep(ctx, state, Step{Kind: "apply_condition", Args: conditionArgs}); err != nil {
				return err
			}
		}
		targetID, err := actorID(state, targetName)
		if err != nil {
			return err
		}
		if err := r.waitForDaggerheartCharacterProjection(ctx, state, targetID, nil, nil); err != nil {
			return err
		}
		appliedDamage := damageAmount
		if result.success && halfDamageOnSuccess {
			appliedDamage = appliedDamage / 2
		}
		if appliedDamage > 0 {
			beforeDamage, err := r.latestSeq(ctx, state)
			if err != nil {
				return err
			}
			ctxWithMeta := withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID)
			_, err = r.env.daggerheartClient.ApplyDamage(ctxWithMeta, &daggerheartv1.DaggerheartApplyDamageRequest{
				CampaignId:  state.campaignID,
				SceneId:     state.activeSceneID,
				CharacterId: targetID,
				Damage:      buildDamageRequest(step.Args, "", damageSource, int32(appliedDamage)),
			})
			if err != nil {
				return fmt.Errorf("group_reaction apply damage: %w", err)
			}
			if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, beforeDamage, daggerheart.EventTypeDamageApplied); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) runGMSpendFearStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	amount, ok := readInt(step.Args, "amount")
	if !ok {
		return r.failf("gm_spend_fear amount is required")
	}
	if amount <= 0 {
		return r.failf("gm_spend_fear amount must be greater than zero")
	}
	move := strings.TrimSpace(optionalString(step.Args, "move", ""))
	description := optionalString(step.Args, "description", "")
	if move == "" {
		if description != "" {
			move = "custom"
		} else {
			move = "spotlight"
		}
	}
	if target := optionalString(step.Args, "target", ""); target != "" {
		if description == "" {
			description = fmt.Sprintf("spotlight %s", target)
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	moveKind, moveShape, err := scenarioGMMoveType(move)
	if err != nil && optionalString(step.Args, "spend_target", "direct_move") == "direct_move" {
		return err
	}
	request, err := buildScenarioGMMoveRequest(ctx, r, state, step.Args, int32(amount), moveKind, moveShape, description)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.ApplyGmMove(ctx, request)
	if err != nil {
		return fmt.Errorf("apply gm move: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(
		ctx,
		state,
		before,
		daggerheart.EventTypeGMMoveApplied,
		daggerheart.EventTypeGMFearChanged,
	); err != nil {
		return err
	}
	state.gmFear = int(response.GetGmFearAfter())
	return nil
}

func buildScenarioGMMoveRequest(
	ctx context.Context,
	r *Runner,
	state *scenarioState,
	args map[string]any,
	amount int32,
	moveKind daggerheartv1.DaggerheartGmMoveKind,
	moveShape daggerheartv1.DaggerheartGmMoveShape,
	description string,
) (*daggerheartv1.DaggerheartApplyGmMoveRequest, error) {
	target := strings.TrimSpace(optionalString(args, "target", ""))
	adversaryID := strings.TrimSpace(optionalString(args, "adversary_id", ""))
	if adversaryID == "" {
		if target != "" {
			if resolved, ok := state.adversaries[target]; ok {
				adversaryID = resolved
			}
		}
	}
	if optionalString(args, "spend_target", "direct_move") == "direct_move" &&
		moveShape == daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY &&
		adversaryID == "" &&
		target != "" {
		if strings.TrimSpace(description) == "" {
			description = fmt.Sprintf("spotlight %s", target)
		}
		moveShape = daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM
	}
	request := &daggerheartv1.DaggerheartApplyGmMoveRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
		SceneId:    state.activeSceneID,
		FearSpent:  amount,
	}
	switch optionalString(args, "spend_target", "direct_move") {
	case "direct_move":
		request.SpendTarget = &daggerheartv1.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &daggerheartv1.DaggerheartDirectGmMoveTarget{
				Kind:        moveKind,
				Shape:       moveShape,
				Description: description,
				AdversaryId: adversaryID,
			},
		}
	case "adversary_feature":
		request.SpendTarget = &daggerheartv1.DaggerheartApplyGmMoveRequest_AdversaryFeature{
			AdversaryFeature: &daggerheartv1.DaggerheartAdversaryFearFeatureTarget{
				AdversaryId: adversaryID,
				FeatureId:   requiredString(args, "feature_id"),
				Description: description,
			},
		}
	case "environment_feature":
		environmentEntityID := optionalString(args, "environment_entity_id", "")
		if environmentEntityID == "" {
			environmentID := requiredString(args, "environment_id")
			sceneID := state.activeSceneID
			if strings.TrimSpace(sceneID) == "" {
				sceneID = state.sessionID
			}
			resp, err := r.env.daggerheartClient.CreateEnvironmentEntity(withSessionID(ctx, state.sessionID), &daggerheartv1.DaggerheartCreateEnvironmentEntityRequest{
				CampaignId:    state.campaignID,
				SessionId:     state.sessionID,
				SceneId:       sceneID,
				EnvironmentId: environmentID,
			})
			if err != nil {
				return nil, fmt.Errorf("create environment entity: %w", err)
			}
			environmentEntityID = resp.GetEnvironmentEntity().GetId()
		}
		request.SpendTarget = &daggerheartv1.DaggerheartApplyGmMoveRequest_EnvironmentFeature{
			EnvironmentFeature: &daggerheartv1.DaggerheartEnvironmentFearFeatureTarget{
				EnvironmentEntityId: environmentEntityID,
				FeatureId:           requiredString(args, "feature_id"),
				Description:         description,
			},
		}
	case "adversary_experience":
		request.SpendTarget = &daggerheartv1.DaggerheartApplyGmMoveRequest_AdversaryExperience{
			AdversaryExperience: &daggerheartv1.DaggerheartAdversaryExperienceTarget{
				AdversaryId:    adversaryID,
				ExperienceName: requiredString(args, "experience_name"),
				Description:    description,
			},
		}
	default:
		return nil, fmt.Errorf("unsupported gm_spend_fear spend_target %q", optionalString(args, "spend_target", ""))
	}
	return request, nil
}

func scenarioGMMoveType(move string) (daggerheartv1.DaggerheartGmMoveKind, daggerheartv1.DaggerheartGmMoveShape, error) {
	switch strings.ToLower(strings.TrimSpace(move)) {
	case "", "spotlight":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY,
			nil
	case "change_environment":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			nil
	case "reveal_danger":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER,
			nil
	case "mark_stress":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS,
			nil
	case "force_split":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT,
			nil
	case "show_world_reaction":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION,
			nil
	case "custom":
		return daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
			daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM,
			nil
	default:
		return 0, 0, fmt.Errorf("unsupported gm_spend_fear move %q", move)
	}
}

func (r *Runner) runSetSpotlightStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	spotlightType := strings.ToLower(strings.TrimSpace(optionalString(step.Args, "type", "")))
	name := optionalString(step.Args, "target", "")
	request := &gamev1.SetSessionSpotlightRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
	}
	if spotlightType == "" {
		if strings.TrimSpace(name) == "" {
			spotlightType = "gm"
		} else {
			spotlightType = "character"
		}
	}
	switch spotlightType {
	case "gm":
		request.Type = gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM
	case "character":
		if strings.TrimSpace(name) == "" {
			return r.failf("set_spotlight character requires target")
		}
		request.Type = gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER
		characterID, err := actorID(state, name)
		if err != nil {
			return err
		}
		request.CharacterId = characterID
	default:
		return r.failf("unsupported spotlight type %q", spotlightType)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	if _, err := r.env.sessionClient.SetSessionSpotlight(ctx, request); err != nil {
		return fmt.Errorf("set spotlight: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, event.TypeSessionSpotlightSet); err != nil {
		return err
	}
	return r.assertExpectedSpotlight(ctx, state, step.Args)
}

func (r *Runner) runClearSpotlightStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	if _, err := r.env.sessionClient.ClearSessionSpotlight(ctx, &gamev1.ClearSessionSpotlightRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
	}); err != nil {
		return fmt.Errorf("clear spotlight: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, event.TypeSessionSpotlightCleared); err != nil {
		return err
	}
	return r.assertExpectedSpotlight(ctx, state, step.Args)
}

func (r *Runner) runApplyConditionStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("apply_condition target is required")
	}
	add := readStringSlice(step.Args, "add")
	remove := readStringSlice(step.Args, "remove")
	lifeState := optionalString(step.Args, "life_state", "")
	if len(add) == 0 && len(remove) == 0 && lifeState == "" {
		return r.failf("apply_condition requires add, remove, or life_state")
	}
	addValues, err := parseConditionStates(add)
	if err != nil {
		return err
	}
	removeValues, err := parseConditionIDs(remove)
	if err != nil {
		return err
	}
	if lifeState != "" {
		if _, err := parseLifeState(lifeState); err != nil {
			return err
		}
	}

	characterID, characterErr := actorID(state, name)
	adversaryIDValue, adversaryErr := adversaryID(state, name)
	if characterErr != nil && adversaryErr != nil {
		return characterErr
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	source := optionalString(step.Args, "source", "")
	if characterErr == nil {
		request := &daggerheartv1.DaggerheartApplyConditionsRequest{
			CampaignId:         state.campaignID,
			SceneId:            state.activeSceneID,
			CharacterId:        characterID,
			AddConditions:      addValues,
			RemoveConditionIds: removeValues,
			Source:             source,
		}
		if lifeState != "" {
			value, err := parseLifeState(lifeState)
			if err != nil {
				return err
			}
			request.LifeState = value
		}
		_, err = r.env.daggerheartClient.ApplyConditions(withSessionID(ctx, state.sessionID), request)
		if err != nil {
			return fmt.Errorf("apply conditions: %w", err)
		}
		eventTypes := []event.Type{}
		if len(add) > 0 || len(remove) > 0 {
			converted, err := convertDaggerheartEventTypes(daggerheart.EventTypeConditionChanged)
			if err != nil {
				return err
			}
			eventTypes = append(eventTypes, converted...)
		}
		if lifeState != "" {
			converted, err := convertDaggerheartEventTypes(daggerheart.EventTypeCharacterStatePatched)
			if err != nil {
				return err
			}
			eventTypes = append(eventTypes, converted...)
		}
		return r.requireEventTypesAfterSeq(ctx, state, before, eventTypes...)
	}

	if lifeState != "" {
		return r.failf("apply_condition life_state is only supported for characters")
	}
	_, err = r.env.daggerheartClient.ApplyAdversaryConditions(withSessionID(ctx, state.sessionID), &daggerheartv1.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:         state.campaignID,
		SceneId:            state.activeSceneID,
		AdversaryId:        adversaryIDValue,
		AddConditions:      addValues,
		RemoveConditionIds: removeValues,
		Source:             source,
	})
	if err != nil {
		return fmt.Errorf("apply adversary conditions: %w", err)
	}
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryConditionChanged)
}

func (r *Runner) runGroupActionStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	leaderName := requiredString(step.Args, "leader")
	leaderTrait := requiredString(step.Args, "leader_trait")
	difficulty := optionalInt(step.Args, "difficulty", 0)
	if leaderName == "" || leaderTrait == "" || difficulty == 0 {
		return r.failf("group_action requires leader, leader_trait, and difficulty")
	}

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, leaderName)
	if err != nil {
		return err
	}

	supportersRaw, ok := step.Args["supporters"]
	if !ok {
		return r.failf("group_action requires supporters")
	}
	supporterList, ok := supportersRaw.([]any)
	if !ok || len(supporterList) == 0 {
		return r.failf("group_action supporters must be a list")
	}

	baseSeed := uint64(optionalInt(step.Args, "seed", 42))
	leaderSeed, err := resolveOutcomeSeed(step.Args, "outcome", difficulty, baseSeed)
	if err != nil {
		return err
	}
	leaderContext, err := actionRollContextFromScenario(optionalString(step.Args, "leader_context", ""))
	if err != nil {
		return err
	}
	leaderModifiers := buildActionRollModifiers(step.Args, "leader_modifiers")

	supporters := make([]*daggerheartv1.GroupActionSupporter, 0, len(supporterList))
	for index, entry := range supporterList {
		item, ok := entry.(map[string]any)
		if !ok {
			return r.failf("group_action supporter %d must be an object", index)
		}
		name := requiredString(item, "name")
		trait := requiredString(item, "trait")
		if name == "" || trait == "" {
			return r.failf("group_action supporter %d requires name and trait", index)
		}
		seed, err := resolveOutcomeSeed(item, "outcome", difficulty, baseSeed+uint64(index)+1)
		if err != nil {
			return err
		}
		contextValue, err := actionRollContextFromScenario(optionalString(item, "context", ""))
		if err != nil {
			return err
		}
		actorIDValue, err := actorID(state, name)
		if err != nil {
			return err
		}
		supporters = append(supporters, &daggerheartv1.GroupActionSupporter{
			CharacterId: actorIDValue,
			Trait:       trait,
			Modifiers:   buildActionRollModifiers(item, "modifiers"),
			Context:     contextValue,
			Rng: &commonv1.RngRequest{
				Seed:     &seed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		})
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	leaderID, err := actorID(state, leaderName)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.SessionGroupActionFlow(ctx, &daggerheartv1.SessionGroupActionFlowRequest{
		CampaignId:        state.campaignID,
		SessionId:         state.sessionID,
		SceneId:           state.activeSceneID,
		LeaderCharacterId: leaderID,
		LeaderTrait:       leaderTrait,
		Difficulty:        int32(difficulty),
		LeaderModifiers:   leaderModifiers,
		LeaderContext:     leaderContext,
		LeaderRng: &commonv1.RngRequest{
			Seed:     &leaderSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
		Supporters: supporters,
	})
	if err != nil {
		return fmt.Errorf("group_action: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runTagTeamStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	firstName := requiredString(step.Args, "first")
	secondName := requiredString(step.Args, "second")
	selectedName := requiredString(step.Args, "selected")
	firstTrait := requiredString(step.Args, "first_trait")
	secondTrait := requiredString(step.Args, "second_trait")
	difficulty := optionalInt(step.Args, "difficulty", 0)
	if firstName == "" || secondName == "" || selectedName == "" || firstTrait == "" || secondTrait == "" || difficulty == 0 {
		return r.failf("tag_team requires first, second, selected, first_trait, second_trait, and difficulty")
	}

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, selectedName)
	if err != nil {
		return err
	}

	baseSeed := uint64(optionalInt(step.Args, "seed", 42))
	firstSeed, err := resolveOutcomeSeed(step.Args, "first_outcome", difficulty, baseSeed)
	if err != nil {
		return err
	}
	secondSeed, err := resolveOutcomeSeed(step.Args, "second_outcome", difficulty, baseSeed+1)
	if err != nil {
		return err
	}
	selectedOutcome := optionalString(step.Args, "outcome", "")
	if selectedOutcome != "" {
		if selectedName == firstName {
			firstSeed, err = resolveOutcomeSeed(map[string]any{"outcome": selectedOutcome}, "outcome", difficulty, firstSeed)
			if err != nil {
				return err
			}
		} else if selectedName == secondName {
			secondSeed, err = resolveOutcomeSeed(map[string]any{"outcome": selectedOutcome}, "outcome", difficulty, secondSeed)
			if err != nil {
				return err
			}
		}
	}

	firstID, err := actorID(state, firstName)
	if err != nil {
		return err
	}
	secondID, err := actorID(state, secondName)
	if err != nil {
		return err
	}
	selectedID, err := actorID(state, selectedName)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.SessionTagTeamFlow(ctx, &daggerheartv1.SessionTagTeamFlowRequest{
		CampaignId:          state.campaignID,
		SessionId:           state.sessionID,
		Difficulty:          int32(difficulty),
		SelectedCharacterId: selectedID,
		First: &daggerheartv1.TagTeamParticipant{
			CharacterId: firstID,
			Trait:       firstTrait,
			Modifiers:   buildActionRollModifiers(step.Args, "first_modifiers"),
			Rng: &commonv1.RngRequest{
				Seed:     &firstSeed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		},
		Second: &daggerheartv1.TagTeamParticipant{
			CharacterId: secondID,
			Trait:       secondTrait,
			Modifiers:   buildActionRollModifiers(step.Args, "second_modifiers"),
			Rng: &commonv1.RngRequest{
				Seed:     &secondSeed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("tag_team: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runTemporaryArmorStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("temporary_armor target is required")
	}
	source := requiredString(step.Args, "source")
	if source == "" {
		return r.failf("temporary_armor source is required")
	}
	durationValue := requiredString(step.Args, "duration")
	if durationValue == "" {
		return r.failf("temporary_armor duration is required")
	}
	duration, err := parseTemporaryArmorDuration(durationValue)
	if err != nil {
		return err
	}
	amount, ok := readInt(step.Args, "amount")
	if !ok || amount <= 0 {
		return r.failf("temporary_armor amount must be positive")
	}
	sourceID := optionalString(step.Args, "source_id", "")

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, name)
	if err != nil {
		return err
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.ApplyTemporaryArmor(ctxWithSession, &daggerheartv1.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  state.campaignID,
		SceneId:     state.activeSceneID,
		CharacterId: characterID,
		Armor: &daggerheartv1.DaggerheartTemporaryArmor{
			Source:   source,
			Duration: duration,
			Amount:   int32(amount),
			SourceId: sourceID,
		},
	})
	if err != nil {
		return fmt.Errorf("temporary_armor: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCharacterTemporaryArmorApplied); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runRestStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	restType := optionalString(step.Args, "type", "")
	if restType == "" {
		restType = optionalString(step.Args, "rest_type", "")
	}
	if restType == "" {
		return r.failf("rest type is required")
	}
	interrupted := optionalBool(step.Args, "interrupted", false)
	seed := optionalInt(step.Args, "seed", 0)

	characterNames := readStringSlice(step.Args, "characters")
	participants, err := r.buildRestParticipants(state, step.Args)
	if err != nil {
		return err
	}

	fallbackName := ""
	if len(characterNames) == 1 {
		fallbackName = characterNames[0]
	}
	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, fallbackName)
	if err != nil {
		return err
	}

	parsedRestType, err := parseRestType(restType)
	if err != nil {
		return err
	}
	rest := &daggerheartv1.DaggerheartRestRequest{
		RestType:     parsedRestType,
		Interrupted:  interrupted,
		Participants: participants,
	}
	if seed != 0 {
		seedValue := uint64(seed)
		rest.Rng = &commonv1.RngRequest{
			Seed:     &seedValue,
			RollMode: commonv1.RollMode_REPLAY,
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.ApplyRest(ctxWithSession, &daggerheartv1.DaggerheartApplyRestRequest{
		CampaignId: state.campaignID,
		Rest:       rest,
	})
	if err != nil {
		return fmt.Errorf("rest: %w", err)
	}
	expectedEvents := []any{daggerheart.EventTypeRestTaken}
	if restContainsDowntimeMoves(participants) {
		expectedEvents = append(expectedEvents, daggerheart.EventTypeDowntimeMoveApplied)
	}
	if restContainsProjectWork(participants) {
		expectedEvents = append(expectedEvents, daggerheart.EventTypeCountdownUpdated)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, expectedEvents...); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) buildRestParticipants(state *scenarioState, args map[string]any) ([]*daggerheartv1.DaggerheartRestParticipant, error) {
	entries := readMapSlice(args, "participants")
	if len(entries) == 0 {
		characterIDs, err := resolveCharacterList(state, args, "characters")
		if err != nil {
			return nil, err
		}
		if len(characterIDs) == 0 {
			characterIDs = allActorIDs(state)
		}
		participants := make([]*daggerheartv1.DaggerheartRestParticipant, 0, len(characterIDs))
		for _, characterID := range characterIDs {
			participants = append(participants, &daggerheartv1.DaggerheartRestParticipant{CharacterId: characterID})
		}
		return participants, nil
	}

	participants := make([]*daggerheartv1.DaggerheartRestParticipant, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(optionalString(entry, "character", ""))
		if name == "" {
			return nil, r.failf("rest participant character is required")
		}
		characterID, err := actorID(state, name)
		if err != nil {
			return nil, err
		}
		moves, err := r.buildRestDowntimeSelections(state, entry)
		if err != nil {
			return nil, err
		}
		participants = append(participants, &daggerheartv1.DaggerheartRestParticipant{
			CharacterId:   characterID,
			DowntimeMoves: moves,
		})
	}
	return participants, nil
}

func (r *Runner) buildRestDowntimeSelections(state *scenarioState, participant map[string]any) ([]*daggerheartv1.DaggerheartDowntimeSelection, error) {
	entries := readMapSlice(participant, "downtime_moves")
	if len(entries) == 0 {
		return nil, nil
	}
	selections := make([]*daggerheartv1.DaggerheartDowntimeSelection, 0, len(entries))
	for _, entry := range entries {
		move, err := parseDowntimeMove(requiredString(entry, "move"))
		if err != nil {
			return nil, err
		}
		targetName := optionalString(entry, "target", "")
		targetCharacterID := ""
		if targetName != "" {
			targetCharacterID, err = actorID(state, targetName)
			if err != nil {
				return nil, err
			}
		}
		var rng *commonv1.RngRequest
		if seed, ok := readInt(entry, "seed"); ok {
			seedValue := uint64(seed)
			rng = &commonv1.RngRequest{Seed: &seedValue, RollMode: commonv1.RollMode_REPLAY}
		}
		switch move {
		case "tend_to_wounds":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_TendToWounds{
					TendToWounds: &daggerheartv1.DaggerheartTendToWoundsMove{TargetCharacterId: targetCharacterID, Rng: rng},
				},
			})
		case "clear_stress":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_ClearStress{
					ClearStress: &daggerheartv1.DaggerheartClearStressMove{Rng: rng},
				},
			})
		case "repair_armor":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_RepairArmor{
					RepairArmor: &daggerheartv1.DaggerheartRepairArmorMove{TargetCharacterId: targetCharacterID, Rng: rng},
				},
			})
		case "prepare":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_Prepare{
					Prepare: &daggerheartv1.DaggerheartPrepareMove{GroupId: optionalString(entry, "group_id", "")},
				},
			})
		case "tend_to_all_wounds":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_TendToAllWounds{
					TendToAllWounds: &daggerheartv1.DaggerheartTendToAllWoundsMove{TargetCharacterId: targetCharacterID},
				},
			})
		case "clear_all_stress":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_ClearAllStress{
					ClearAllStress: &daggerheartv1.DaggerheartClearAllStressMove{},
				},
			})
		case "repair_all_armor":
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_RepairAllArmor{
					RepairAllArmor: &daggerheartv1.DaggerheartRepairAllArmorMove{TargetCharacterId: targetCharacterID},
				},
			})
		case "work_on_project":
			projectID := optionalString(entry, "countdown_id", optionalString(entry, "countdown", ""))
			if resolved, ok := state.countdowns[projectID]; ok {
				projectID = resolved
			}
			mode := daggerheartv1.DaggerheartProjectAdvanceMode_DAGGERHEART_PROJECT_ADVANCE_MODE_AUTO
			if strings.EqualFold(optionalString(entry, "advance_mode", ""), "gm_set_delta") {
				mode = daggerheartv1.DaggerheartProjectAdvanceMode_DAGGERHEART_PROJECT_ADVANCE_MODE_GM_SET_DELTA
			}
			selections = append(selections, &daggerheartv1.DaggerheartDowntimeSelection{
				Move: &daggerheartv1.DaggerheartDowntimeSelection_WorkOnProject{
					WorkOnProject: &daggerheartv1.DaggerheartWorkOnProjectMove{
						CountdownId:  projectID,
						AdvanceMode:  mode,
						AdvanceDelta: int32(optionalInt(entry, "advance_delta", 0)),
						Reason:       optionalString(entry, "reason", ""),
					},
				},
			})
		default:
			return nil, r.failf("unsupported downtime move %q", move)
		}
	}
	return selections, nil
}

func restContainsDowntimeMoves(participants []*daggerheartv1.DaggerheartRestParticipant) bool {
	for _, participant := range participants {
		if participant != nil && len(participant.GetDowntimeMoves()) > 0 {
			return true
		}
	}
	return false
}

func restContainsProjectWork(participants []*daggerheartv1.DaggerheartRestParticipant) bool {
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		for _, move := range participant.GetDowntimeMoves() {
			if move != nil && move.GetWorkOnProject() != nil {
				return true
			}
		}
	}
	return false
}

func (r *Runner) runDeathMoveStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("death_move target is required")
	}
	move := requiredString(step.Args, "move")
	if move == "" {
		return r.failf("death_move move is required")
	}
	hpClear, hpOk := readInt(step.Args, "hp_clear")
	stressClear, stressOk := readInt(step.Args, "stress_clear")
	seed := optionalInt(step.Args, "seed", 0)

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, name)
	if err != nil {
		return err
	}
	expectedDeath, err := r.captureExpectedDeathMove(step.Args)
	if err != nil {
		return err
	}

	parsedMove, err := parseDeathMove(move)
	if err != nil {
		return err
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	request := &daggerheartv1.DaggerheartApplyDeathMoveRequest{
		CampaignId:  state.campaignID,
		SceneId:     state.activeSceneID,
		CharacterId: characterID,
		Move:        parsedMove,
	}
	if hpOk {
		value := int32(hpClear)
		request.HpClear = &value
	}
	if stressOk {
		value := int32(stressClear)
		request.StressClear = &value
	}
	if seed != 0 {
		seedValue := uint64(seed)
		request.Rng = &commonv1.RngRequest{
			Seed:     &seedValue,
			RollMode: commonv1.RollMode_REPLAY,
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	response, err := r.env.daggerheartClient.ApplyDeathMove(ctxWithSession, request)
	if err != nil {
		return fmt.Errorf("death_move: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCharacterStatePatched); err != nil {
		return err
	}
	if err := r.assertExpectedDeathMove(response, expectedDeath); err != nil {
		return err
	}
	if response.GetState() != nil {
		return r.assertExpectedDeltasAfterState(expectedSpec, expectedBefore, response.GetState())
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

type expectedDeathMove struct {
	lifeState  *daggerheartv1.DaggerheartLifeState
	scarGained *bool
	hopeMax    *int32
	hopeDie    *int32
}

func (r *Runner) captureExpectedDeathMove(args map[string]any) (*expectedDeathMove, error) {
	spec := &expectedDeathMove{}
	if value := strings.TrimSpace(optionalString(args, "expect_life_state", "")); value != "" {
		lifeState, err := parseLifeState(value)
		if err != nil {
			return nil, err
		}
		spec.lifeState = &lifeState
	}
	if _, present := args["expect_scar_gained"]; present {
		value, ok := readBool(args, "expect_scar_gained")
		if !ok {
			return nil, r.failf("expect_scar_gained must be a boolean")
		}
		spec.scarGained = &value
	}
	if _, present := args["expect_hope_max"]; present {
		value, ok := readInt(args, "expect_hope_max")
		if !ok {
			return nil, r.failf("expect_hope_max must be an integer")
		}
		boxed := int32(value)
		spec.hopeMax = &boxed
	}
	if _, present := args["expect_hope_die"]; present {
		value, ok := readInt(args, "expect_hope_die")
		if !ok {
			return nil, r.failf("expect_hope_die must be an integer")
		}
		boxed := int32(value)
		spec.hopeDie = &boxed
	}
	if spec.lifeState == nil && spec.scarGained == nil && spec.hopeMax == nil && spec.hopeDie == nil {
		return nil, nil
	}
	return spec, nil
}

func (r *Runner) assertExpectedDeathMove(response *daggerheartv1.DaggerheartApplyDeathMoveResponse, spec *expectedDeathMove) error {
	if spec == nil {
		return nil
	}
	if response == nil {
		return r.failf("expected death move response")
	}
	if spec.lifeState != nil || spec.scarGained != nil || spec.hopeDie != nil {
		if response.GetResult() == nil {
			return r.failf("expected death move result in response")
		}
	}
	if spec.hopeMax != nil || spec.lifeState != nil {
		if response.GetState() == nil {
			return r.failf("expected death move state in response")
		}
	}
	if spec.lifeState != nil && response.GetResult().GetLifeState() != *spec.lifeState {
		return r.assertf("death_move life_state = %s, want %s", response.GetResult().GetLifeState(), *spec.lifeState)
	}
	if spec.scarGained != nil && response.GetResult().GetScarGained() != *spec.scarGained {
		return r.assertf("death_move scar_gained = %t, want %t", response.GetResult().GetScarGained(), *spec.scarGained)
	}
	if spec.hopeDie != nil {
		after := response.GetResult().HopeDie
		if after == nil {
			return r.failf("death_move response missing hope_die")
		}
		if *after != *spec.hopeDie {
			return r.assertf("death_move hope_die = %d, want %d", *after, *spec.hopeDie)
		}
	}
	if spec.hopeMax != nil && response.GetState().GetHopeMax() != *spec.hopeMax {
		return r.assertf("death_move hope_max = %d, want %d", response.GetState().GetHopeMax(), *spec.hopeMax)
	}
	return nil
}

func (r *Runner) runBlazeOfGloryStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("blaze_of_glory target is required")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.ResolveBlazeOfGlory(ctxWithSession, &daggerheartv1.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  state.campaignID,
		SceneId:     state.activeSceneID,
		CharacterId: characterID,
	})
	if err != nil {
		return fmt.Errorf("blaze_of_glory: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCharacterStatePatched); err != nil {
		return err
	}
	return r.requireAnyEventTypesAfterSeq(ctx, state, before, event.TypeCharacterDeleted)
}

func (r *Runner) runAttackStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	targetName := requiredString(step.Args, "target")
	if actorName == "" || targetName == "" {
		return r.failf("attack requires actor and target")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	attackerID, err := actorID(state, actorName)
	if err != nil {
		return err
	}
	targetID, targetIsAdversary, err := resolveTargetID(state, targetName)
	if err != nil {
		return err
	}

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, actorName)
	if err != nil {
		return err
	}
	expectedAdversarySpec, expectedAdversaryBefore, err := r.captureExpectedAdversaryDeltas(ctx, state, step.Args, targetName)
	if err != nil {
		return err
	}

	actionSeed := uint64(optionalInt(step.Args, "seed", 0))
	if actionSeed == 0 {
		actionSeed, err = chooseActionSeed(step.Args, difficulty)
		if err != nil {
			return err
		}
	}
	damageSeed := uint64(optionalInt(step.Args, "damage_seed", 0))
	if damageSeed == 0 {
		damageSeed = actionSeed + 1
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	if !targetIsAdversary {
		stateBefore, err := r.getCharacterState(ctx, state, targetID)
		if err != nil {
			return err
		}
		response, err := r.env.daggerheartClient.SessionAttackFlow(ctx, &daggerheartv1.SessionAttackFlowRequest{
			CampaignId:           state.campaignID,
			SessionId:            state.sessionID,
			SceneId:              state.activeSceneID,
			CharacterId:          attackerID,
			Difficulty:           int32(difficulty),
			Modifiers:            buildActionRollModifiers(step.Args, "modifiers"),
			TargetId:             targetID,
			Damage:               buildDamageSpec(step.Args, attackerID, "attack"),
			RequireDamageRoll:    true,
			ReplaceHopeWithArmor: optionalBool(step.Args, "replace_hope_with_armor", false),
			AttackProfile: &daggerheartv1.SessionAttackFlowRequest_StandardAttack{
				StandardAttack: &daggerheartv1.SessionStandardAttackProfile{
					Trait:       trait,
					AttackRange: buildAttackRange(step.Args),
					DamageDice:  buildDamageDice(step.Args),
				},
			},
			ActionRng: &commonv1.RngRequest{
				Seed:     &actionSeed,
				RollMode: commonv1.RollMode_REPLAY,
			},
			DamageRng: &commonv1.RngRequest{
				Seed:     &damageSeed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		})
		if err != nil {
			return fmt.Errorf("attack flow: %w", err)
		}
		if want, ok := readInt(step.Args, "expect_action_total"); ok && int(response.GetActionRoll().GetTotal()) != want {
			return r.assertf("attack action_total = %d, want %d", response.GetActionRoll().GetTotal(), want)
		}
		if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied); err != nil {
			return err
		}
		if response.GetDamageApplied() != nil {
			if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
				return err
			}
			if err := r.assertDamageFlags(ctx, state, before, targetID, step.Args); err != nil {
				return err
			}
			if expectDamageEffect(step.Args, response.GetDamageRoll()) {
				stateAfter, err := r.getCharacterState(ctx, state, targetID)
				if err != nil {
					return err
				}
				if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
					if err := r.assertf("expected damage to affect hp or armor for %s", targetName); err != nil {
						return err
					}
				}
			}
		}
		return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
	}

	ctxWithMeta := withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID)
	rollResp, err := r.env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:   state.campaignID,
		SessionId:    state.sessionID,
		SceneId:      state.activeSceneID,
		CharacterId:  attackerID,
		Trait:        trait,
		RollKind:     daggerheartv1.RollKind_ROLL_KIND_ACTION,
		Difficulty:   int32(difficulty),
		Advantage:    int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage: int32(optionalInt(step.Args, "disadvantage", 0)),
		Modifiers:    buildActionRollModifiers(step.Args, "modifiers"),
		Rng: &commonv1.RngRequest{
			Seed:     &actionSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("attack action roll: %w", err)
	}

	rollOutcomeResponse, err := r.env.daggerheartClient.ApplyRollOutcome(ctxWithMeta, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return fmt.Errorf("attack roll outcome: %w", err)
	}
	if rollOutcomeResponse.GetRequiresComplication() {
		if err := r.resolveOpenSessionGate(ctx, state, before); err != nil {
			return err
		}
	}

	attackOutcome, err := r.env.daggerheartClient.ApplyAttackOutcome(ctxWithMeta, &daggerheartv1.DaggerheartApplyAttackOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   []string{targetID},
	})
	if err != nil {
		return fmt.Errorf("attack outcome: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied); err != nil {
		return err
	}

	if attackOutcome.GetResult() != nil && attackOutcome.GetResult().GetSuccess() {
		dice := buildDamageDice(step.Args)
		critical := attackOutcome.GetResult().GetCrit()
		damageRoll, err := r.env.daggerheartClient.SessionDamageRoll(ctx, &daggerheartv1.SessionDamageRollRequest{
			CampaignId:  state.campaignID,
			SessionId:   state.sessionID,
			SceneId:     state.activeSceneID,
			CharacterId: attackerID,
			Dice:        dice,
			Modifier:    0,
			Critical:    critical,
			Rng: &commonv1.RngRequest{
				Seed:     &damageSeed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		})
		if err != nil {
			return fmt.Errorf("attack damage roll: %w", err)
		}
		applied, err := r.applyAdversaryDamage(ctx, state, targetID, targetName, damageRoll, step.Args)
		if err != nil {
			return err
		}
		if applied {
			if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryDamageApplied); err != nil {
				return err
			}
		}
	}
	if err := r.assertExpectedAdversaryDeltas(ctx, state, expectedAdversarySpec, expectedAdversaryBefore); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runMultiAttackStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("multi_attack requires actor")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	attackerID, err := actorID(state, actorName)
	if err != nil {
		return err
	}

	targetNames := uniqueNonEmptyStrings(readStringSlice(step.Args, "targets"))
	if len(targetNames) == 0 {
		return r.failf("multi_attack requires targets")
	}
	type attackTarget struct {
		id        string
		name      string
		adversary bool
	}
	targets := make([]attackTarget, 0, len(targetNames))
	targetIDs := make([]string, 0, len(targetNames))
	for _, name := range targetNames {
		id, isAdversary, err := resolveTargetID(state, name)
		if err != nil {
			return err
		}
		targets = append(targets, attackTarget{id: id, name: name, adversary: isAdversary})
		targetIDs = append(targetIDs, id)
	}

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, actorName)
	if err != nil {
		return err
	}

	actionSeed, err := chooseActionSeed(step.Args, difficulty)
	if err != nil {
		return err
	}
	damageSeed := actionSeed + 1

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	rollResp, err := r.env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		SceneId:     state.activeSceneID,
		CharacterId: attackerID,
		Trait:       trait,
		RollKind:    daggerheartv1.RollKind_ROLL_KIND_ACTION,
		Difficulty:  int32(difficulty),
		Modifiers:   buildActionRollModifiers(step.Args, "modifiers"),
		Rng: &commonv1.RngRequest{
			Seed:     &actionSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("multi_attack action roll: %w", err)
	}

	ctxWithMeta := withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID)
	rollOutcomeResponse, err := r.env.daggerheartClient.ApplyRollOutcome(ctxWithMeta, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return fmt.Errorf("multi_attack roll outcome: %w", err)
	}
	if rollOutcomeResponse.GetRequiresComplication() {
		if err := r.resolveOpenSessionGate(ctx, state, before); err != nil {
			return err
		}
	}

	attackOutcome, err := r.env.daggerheartClient.ApplyAttackOutcome(ctxWithMeta, &daggerheartv1.DaggerheartApplyAttackOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   targetIDs,
	})
	if err != nil {
		return fmt.Errorf("multi_attack outcome: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied); err != nil {
		return err
	}

	if attackOutcome.GetResult() != nil && attackOutcome.GetResult().GetSuccess() {
		dice := buildDamageDice(step.Args)
		if err := requireDamageDice(step.Args, "multi_attack"); err != nil {
			return err
		}
		critical := attackOutcome.GetResult().GetCrit()
		damageRoll, err := r.env.daggerheartClient.SessionDamageRoll(ctx, &daggerheartv1.SessionDamageRollRequest{
			CampaignId:  state.campaignID,
			SessionId:   state.sessionID,
			SceneId:     state.activeSceneID,
			CharacterId: attackerID,
			Dice:        dice,
			Modifier:    int32(optionalInt(step.Args, "damage_modifier", 0)),
			Critical:    critical,
			Rng: &commonv1.RngRequest{
				Seed:     &damageSeed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		})
		if err != nil {
			return fmt.Errorf("multi_attack damage roll: %w", err)
		}

		expectedChange := adjustedDamageAmount(step.Args, damageRoll.GetTotal()) > 0
		for _, target := range targets {
			if target.adversary {
				applied, err := r.applyAdversaryDamage(ctx, state, target.id, target.name, damageRoll, step.Args)
				if err != nil {
					return err
				}
				if applied {
					if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryDamageApplied); err != nil {
						return err
					}
				}
				continue
			}
			stateBefore, err := r.getCharacterState(ctx, state, target.id)
			if err != nil {
				return err
			}
			_, err = r.env.daggerheartClient.ApplyDamage(ctxWithMeta, &daggerheartv1.DaggerheartApplyDamageRequest{
				CampaignId:        state.campaignID,
				SceneId:           state.activeSceneID,
				CharacterId:       target.id,
				Damage:            buildDamageRequest(step.Args, attackerID, "attack", damageRoll.GetTotal()),
				RollSeq:           &damageRoll.RollSeq,
				RequireDamageRoll: true,
			})
			if err != nil {
				return fmt.Errorf("multi_attack apply damage: %w", err)
			}
			if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
				return err
			}
			if err := r.assertDamageFlags(ctx, state, before, target.id, step.Args); err != nil {
				return err
			}
			if expectedChange {
				stateAfter, err := r.getCharacterState(ctx, state, target.id)
				if err != nil {
					return err
				}
				if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
					if err := r.assertf("expected damage to affect hp or armor for %s", target.name); err != nil {
						return err
					}
				}
			}
		}
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runCombinedDamageStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("combined_damage target is required")
	}
	targetID, targetIsAdversary, err := resolveTargetID(state, name)
	if err != nil {
		return err
	}

	sourcesRaw, ok := step.Args["sources"]
	if !ok {
		return r.failf("combined_damage sources are required")
	}
	sourceList, ok := sourcesRaw.([]any)
	if !ok || len(sourceList) == 0 {
		return r.failf("combined_damage sources must be a list")
	}

	amountTotal := 0
	sourceIDs := make([]string, 0, len(sourceList))
	seenSourceIDs := make(map[string]struct{}, len(sourceList))
	for index, entry := range sourceList {
		item, ok := entry.(map[string]any)
		if !ok {
			return r.failf("combined_damage source %d must be an object", index)
		}
		amount, ok := readInt(item, "amount")
		if !ok || amount <= 0 {
			return r.failf("combined_damage source %d requires amount", index)
		}
		amountTotal += amount
		if sourceName := optionalString(item, "character", ""); sourceName != "" {
			if strings.EqualFold(sourceName, "gm") {
				continue
			}
			id, err := actorID(state, sourceName)
			if err != nil {
				return err
			}
			if _, exists := seenSourceIDs[id]; exists {
				return r.failf("combined_damage source %d has duplicate source character %q", index, sourceName)
			}
			seenSourceIDs[id] = struct{}{}
			sourceIDs = append(sourceIDs, id)
		}
	}
	if amountTotal <= 0 {
		return r.failf("combined_damage requires positive total damage")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)

	if !targetIsAdversary {
		stateBefore, err := r.getCharacterState(ctx, state, targetID)
		if err != nil {
			return err
		}
		_, err = r.env.daggerheartClient.ApplyDamage(ctxWithSession, &daggerheartv1.DaggerheartApplyDamageRequest{
			CampaignId:    state.campaignID,
			SceneId:       state.activeSceneID,
			CharacterId:   targetID,
			ArmorReaction: buildDamageArmorReaction(step.Args, uint64(optionalInt(step.Args, "armor_reaction_seed", 42))),
			Damage: buildDamageRequestWithSources(
				step.Args,
				optionalString(step.Args, "source", "combined"),
				int32(amountTotal),
				sourceIDs,
			),
			RequireDamageRoll: false,
		})
		if err != nil {
			return fmt.Errorf("combined_damage apply damage: %w", err)
		}
		if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
			return err
		}
		if err := r.assertDamageFlags(ctx, state, before, targetID, step.Args); err != nil {
			return err
		}
		if adjustedDamageAmount(step.Args, int32(amountTotal)) > 0 {
			stateAfter, err := r.getCharacterState(ctx, state, targetID)
			if err != nil {
				return err
			}
			if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
				return r.assertf("expected damage to affect hp or armor for %s", name)
			}
		}
		return nil
	}

	expectedAdversarySpec, expectedAdversaryBefore, err := r.captureExpectedAdversaryDeltas(ctx, state, step.Args, name)
	if err != nil {
		return err
	}

	_, err = r.env.daggerheartClient.ApplyAdversaryDamage(ctxWithSession, &daggerheartv1.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  state.campaignID,
		SceneId:     state.activeSceneID,
		AdversaryId: targetID,
		Damage: buildDamageRequestWithSources(
			step.Args,
			optionalString(step.Args, "source", "combined"),
			int32(amountTotal),
			sourceIDs,
		),
		RequireDamageRoll: false,
	})
	if err != nil {
		return fmt.Errorf("combined_damage apply adversary damage: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryDamageApplied); err != nil {
		return err
	}
	return r.assertExpectedAdversaryDeltas(ctx, state, expectedAdversarySpec, expectedAdversaryBefore)
}

func (r *Runner) runAdversaryAttackStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	targetNames := uniqueNonEmptyStrings(readStringSlice(step.Args, "targets"))
	if len(targetNames) == 0 {
		if targetName := requiredString(step.Args, "target"); targetName != "" {
			targetNames = []string{targetName}
		}
	}
	if actorName == "" || len(targetNames) == 0 {
		return r.failf("adversary_attack requires actor and target or targets")
	}
	difficulty := optionalInt(step.Args, "difficulty", 10)
	featureID := normalizeScenarioKey(optionalString(step.Args, "feature_id", ""))
	adversaryIDValue, err := adversaryID(state, actorName)
	if err != nil {
		return err
	}
	targetCharacterIDs := make([]string, 0, len(targetNames))
	for _, name := range targetNames {
		targetCharacterID, err := actorID(state, name)
		if err != nil {
			return err
		}
		targetCharacterIDs = append(targetCharacterIDs, targetCharacterID)
	}
	contributorNames := uniqueNonEmptyStrings(readStringSlice(step.Args, "contributors"))
	contributorAdversaryIDs := make([]string, 0, len(contributorNames))
	for _, name := range contributorNames {
		contributorAdversaryID, err := adversaryID(state, name)
		if err != nil {
			return err
		}
		contributorAdversaryIDs = append(contributorAdversaryIDs, contributorAdversaryID)
	}
	expectationTargetName := targetNames[0]
	expectationTargetID := targetCharacterIDs[0]

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, expectationTargetName)
	if err != nil {
		return err
	}

	attackSeed := uint64(42)
	if seed := optionalInt(step.Args, "seed", 0); seed > 0 {
		attackSeed = uint64(seed)
	}
	damageSeed := attackSeed + 1

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	stateBefore, err := r.getCharacterState(ctx, state, expectationTargetID)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionAdversaryAttackFlow(ctx, &daggerheartv1.SessionAdversaryAttackFlowRequest{
		CampaignId:              state.campaignID,
		SessionId:               state.sessionID,
		SceneId:                 state.activeSceneID,
		AdversaryId:             adversaryIDValue,
		TargetId:                expectationTargetID,
		TargetIds:               targetCharacterIDs,
		FeatureId:               featureID,
		ContributorAdversaryIds: contributorAdversaryIDs,
		Difficulty:              int32(difficulty),
		Advantage:               int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage:            int32(optionalInt(step.Args, "disadvantage", 0)),
		Damage:                  buildDamageSpec(step.Args, "", "adversary_attack"),
		RequireDamageRoll:       true,
		TargetArmorReaction:     buildIncomingAttackArmorReaction(step.Args, uint64(optionalInt(step.Args, "armor_reaction_seed", int(damageSeed+1)))),
		AttackRng: &commonv1.RngRequest{
			Seed:     &attackSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
		DamageRng: &commonv1.RngRequest{
			Seed:     &damageSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("adversary attack flow: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved); err != nil {
		return err
	}
	if response.GetDamageApplied() == nil && hasAnyDamageExpectations(step.Args) {
		hasDamage := len(response.GetDamageApplications()) > 0
		if response.GetDamageApplied() != nil {
			hasDamage = true
		}
		if hasDamage {
			goto damageChecks
		}
		success := false
		if response.GetAttackOutcome() != nil && response.GetAttackOutcome().GetResult() != nil {
			success = response.GetAttackOutcome().GetResult().GetSuccess()
		}
		return r.failf("adversary_attack expected damage application but got none (roll_total=%d success=%t difficulty=%d)", response.GetAttackRoll().GetTotal(), success, difficulty)
	}
damageChecks:
	damageApplications := response.GetDamageApplications()
	if len(damageApplications) == 0 && response.GetDamageApplied() != nil {
		damageApplications = []*daggerheartv1.DaggerheartApplyDamageResponse{response.GetDamageApplied()}
	}
	if len(damageApplications) > 0 {
		if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
			return err
		}
		if err := r.assertDamageFlags(ctx, state, before, expectationTargetID, step.Args); err != nil {
			return err
		}
		if expectDamageEffect(step.Args, response.GetDamageRoll()) {
			stateAfter := damageApplications[0].GetState()
			if stateAfter == nil {
				var err error
				stateAfter, err = r.getCharacterState(ctx, state, expectationTargetID)
				if err != nil {
					return err
				}
			}
			if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
				if err := r.assertf("expected damage to affect hp or armor for %s", expectationTargetName); err != nil {
					return err
				}
			}
		}
	}
	if len(damageApplications) > 0 && damageApplications[0].GetState() != nil {
		return r.assertExpectedDeltasAfterState(expectedSpec, expectedBefore, damageApplications[0].GetState())
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func hasAnyDamageExpectations(args map[string]any) bool {
	for key := range args {
		if strings.HasPrefix(key, "expect_damage_") {
			return true
		}
	}
	return false
}

func (r *Runner) runAdversaryReactionStep(ctx context.Context, state *scenarioState, step Step) error {
	if featureID := normalizeScenarioKey(optionalString(step.Args, "feature_id", optionalString(step.Args, "feature", ""))); featureID != "" {
		featureArgs := maps.Clone(step.Args)
		featureArgs["feature_id"] = featureID
		return r.runAdversaryFeatureStep(ctx, state, Step{
			System: step.System,
			Kind:   "adversary_feature",
			Args:   featureArgs,
		})
	}
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("adversary_reaction requires actor")
	}
	adversaryIDValue, err := adversaryID(state, actorName)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)

	if optionalBool(step.Args, "refresh", false) {
		readyNote := optionalString(step.Args, "ready_note", "reaction_ready")
		update := &daggerheartv1.DaggerheartUpdateAdversaryRequest{
			CampaignId:  state.campaignID,
			AdversaryId: adversaryIDValue,
			Notes:       wrapperspb.String(readyNote),
		}
		if _, err := r.env.daggerheartClient.UpdateAdversary(ctxWithSession, update); err != nil {
			return fmt.Errorf("adversary_reaction refresh: %w", err)
		}
		return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryUpdated)
	}

	if !optionalBool(step.Args, "available", true) {
		return nil
	}
	targetName := requiredString(step.Args, "target")
	if targetName == "" {
		return r.failf("adversary_reaction requires target")
	}
	targetID, err := actorID(state, targetName)
	if err != nil {
		return err
	}
	damageAmount := optionalInt(step.Args, "damage", 0)
	if damageAmount <= 0 {
		return r.failf("adversary_reaction requires positive damage")
	}

	_, err = r.env.daggerheartClient.ApplyDamage(ctxWithSession, &daggerheartv1.DaggerheartApplyDamageRequest{
		CampaignId:  state.campaignID,
		SceneId:     state.activeSceneID,
		CharacterId: targetID,
		Damage:      buildDamageRequest(step.Args, "", optionalString(step.Args, "source", "reaction"), int32(damageAmount)),
	})
	if err != nil {
		return fmt.Errorf("adversary_reaction apply damage: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
		return err
	}

	cooldownNote := optionalString(step.Args, "cooldown_note", "reaction_cooldown")
	update := &daggerheartv1.DaggerheartUpdateAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryIDValue,
		Notes:       wrapperspb.String(cooldownNote),
	}
	if _, err := r.env.daggerheartClient.UpdateAdversary(ctxWithSession, update); err != nil {
		return fmt.Errorf("adversary_reaction cooldown: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryUpdated); err != nil {
		return err
	}
	return nil
}

func (r *Runner) runAdversaryFeatureStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("adversary_feature requires actor")
	}
	featureID := normalizeScenarioKey(optionalString(step.Args, "feature_id", optionalString(step.Args, "feature", "")))
	if featureID == "" {
		return r.failf("adversary_feature feature_id is required")
	}
	adversaryIDValue, err := adversaryID(state, actorName)
	if err != nil {
		return err
	}
	req := &daggerheartv1.DaggerheartApplyAdversaryFeatureRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		SceneId:     state.activeSceneID,
		AdversaryId: adversaryIDValue,
		FeatureId:   featureID,
	}
	if targetName := strings.TrimSpace(optionalString(step.Args, "target", "")); targetName != "" {
		targetID, isAdversary, err := resolveTargetID(state, targetName)
		if err != nil {
			return err
		}
		if isAdversary {
			req.TargetAdversaryId = targetID
		} else {
			req.TargetCharacterId = targetID
		}
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	if _, err := r.env.daggerheartClient.ApplyAdversaryFeature(withSessionID(ctx, state.sessionID), req); err != nil {
		return fmt.Errorf("adversary_feature: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryUpdated)
}

func (r *Runner) runAdversaryUpdateStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("adversary_update target is required")
	}
	adversaryIDValue, err := adversaryID(state, name)
	if err != nil {
		return err
	}

	_, hasEvasion := readInt(step.Args, "evasion")
	_, hasEvasionDelta := readInt(step.Args, "evasion_delta")
	_, hasStress := readInt(step.Args, "stress")
	_, hasStressDelta := readInt(step.Args, "stress_delta")
	_, hasNotes := step.Args["notes"]
	notesValue := optionalString(step.Args, "notes", "")
	if !hasEvasion && !hasEvasionDelta && !hasStress && !hasStressDelta && !hasNotes {
		return r.failf("adversary_update requires evasion, evasion_delta, stress, stress_delta, or notes")
	}
	if hasEvasion && hasEvasionDelta {
		return r.failf("adversary_update cannot set both evasion and evasion_delta")
	}
	if hasStress && hasStressDelta {
		return r.failf("adversary_update cannot set both stress and stress_delta")
	}
	if hasEvasion || hasEvasionDelta || hasStress || hasStressDelta {
		legacyNotes := make([]string, 0, 4)
		if value, ok := readInt(step.Args, "evasion"); ok {
			legacyNotes = append(legacyNotes, fmt.Sprintf("evasion=%d", value))
		}
		if value, ok := readInt(step.Args, "evasion_delta"); ok {
			legacyNotes = append(legacyNotes, fmt.Sprintf("evasion_delta=%d", value))
		}
		if value, ok := readInt(step.Args, "stress"); ok {
			legacyNotes = append(legacyNotes, fmt.Sprintf("stress=%d", value))
		}
		if value, ok := readInt(step.Args, "stress_delta"); ok {
			legacyNotes = append(legacyNotes, fmt.Sprintf("stress_delta=%d", value))
		}
		if notesValue != "" {
			legacyNotes = append(legacyNotes, notesValue)
		}
		notesValue = strings.Join(legacyNotes, "; ")
		hasNotes = true
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	update := &daggerheartv1.DaggerheartUpdateAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryIDValue,
	}
	if sceneID := strings.TrimSpace(optionalString(step.Args, "scene_id", "")); sceneID != "" {
		update.SceneId = sceneID
	}
	if hasNotes {
		update.Notes = wrapperspb.String(notesValue)
	}

	if _, err := r.env.daggerheartClient.UpdateAdversary(withSessionID(ctx, state.sessionID), update); err != nil {
		return fmt.Errorf("adversary_update: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryUpdated)
}

func (r *Runner) runSwapLoadoutStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("swap_loadout target is required")
	}
	cardID := requiredString(step.Args, "card_id")
	if cardID == "" {
		return r.failf("swap_loadout card_id is required")
	}
	recallCost := optionalInt(step.Args, "recall_cost", 0)
	inRest := optionalBool(step.Args, "in_rest", false)

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.SwapLoadout(ctxWithSession, &daggerheartv1.DaggerheartSwapLoadoutRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		Swap: &daggerheartv1.DaggerheartLoadoutSwapRequest{
			CardId:     cardID,
			RecallCost: int32(recallCost),
			InRest:     inRest,
		},
	})
	if err != nil {
		return fmt.Errorf("swap_loadout: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeLoadoutSwapped)
}

func (r *Runner) runCountdownCreateStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("countdown_create name is required")
	}
	maxValue := optionalInt(step.Args, "max", 0)
	if maxValue <= 0 {
		maxValue = 4
	}
	kindValue := optionalString(step.Args, "kind", "progress")
	parsedKind, err := parseCountdownKind(kindValue)
	if err != nil {
		return err
	}
	parsedDirection, err := parseCountdownDirection(optionalString(step.Args, "direction", "increase"))
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	looping := optionalBool(step.Args, "looping", false)
	if strings.EqualFold(kindValue, "loop") {
		looping = true
	}
	request := &daggerheartv1.DaggerheartCreateCountdownRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
		SceneId:    state.activeSceneID,
		Name:       name,
		Kind:       parsedKind,
		Current:    int32(optionalInt(step.Args, "current", 0)),
		Max:        int32(maxValue),
		Direction:  parsedDirection,
		Looping:    looping,
	}
	if countdownID := optionalString(step.Args, "countdown_id", ""); countdownID != "" {
		request.CountdownId = countdownID
	}
	response, err := r.env.daggerheartClient.CreateCountdown(ctx, request)
	if err != nil {
		return fmt.Errorf("countdown_create: %w", err)
	}
	if response.GetCountdown() == nil {
		return r.failf("expected countdown")
	}
	state.countdowns[name] = response.GetCountdown().GetCountdownId()
	r.logf("countdown created: name=%s id=%s", name, state.countdowns[name])
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCountdownCreated)
}

func (r *Runner) runCountdownUpdateStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	countdownID, err := resolveCountdownID(state, step.Args)
	if err != nil {
		return err
	}
	if countdownID == "" {
		return r.failf("countdown_update countdown_id or name is required")
	}

	delta := optionalInt(step.Args, "delta", 0)
	current, hasCurrent := readInt(step.Args, "current")
	if delta == 0 && !hasCurrent {
		return r.failf("countdown_update requires delta or current")
	}

	request := &daggerheartv1.DaggerheartUpdateCountdownRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		SceneId:     state.activeSceneID,
		CountdownId: countdownID,
		Delta:       int32(delta),
		Reason:      optionalString(step.Args, "reason", ""),
	}
	if hasCurrent {
		value := int32(current)
		request.Current = &value
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.UpdateCountdown(ctx, request)
	if err != nil {
		return fmt.Errorf("countdown_update: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCountdownUpdated)
}

func (r *Runner) runCountdownDeleteStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	countdownID, err := resolveCountdownID(state, step.Args)
	if err != nil {
		return err
	}
	if countdownID == "" {
		return r.failf("countdown_delete countdown_id or name is required")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.DeleteCountdown(ctx, &daggerheartv1.DaggerheartDeleteCountdownRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		SceneId:     state.activeSceneID,
		CountdownId: countdownID,
		Reason:      optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("countdown_delete: %w", err)
	}
	if err := r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCountdownDeleted); err != nil {
		return err
	}
	if name := optionalString(step.Args, "name", ""); name != "" {
		delete(state.countdowns, name)
	}
	return nil
}

func (r *Runner) runActionRollStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("action_roll requires actor")
	}
	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, actorName)
	if err != nil {
		return err
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	seed := uint64(optionalInt(step.Args, "seed", 0))
	if seed == 0 {
		seedValue, err := chooseActionSeed(step.Args, difficulty)
		if err != nil {
			return err
		}
		seed = seedValue
	}
	contextValue, err := actionRollContextFromScenario(optionalString(step.Args, "context", ""))
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	actorIDValue, err := actorID(state, actorName)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:           state.campaignID,
		SessionId:            state.sessionID,
		SceneId:              state.activeSceneID,
		CharacterId:          actorIDValue,
		Trait:                trait,
		RollKind:             daggerheartv1.RollKind_ROLL_KIND_ACTION,
		Difficulty:           int32(difficulty),
		Advantage:            int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage:         int32(optionalInt(step.Args, "disadvantage", 0)),
		Modifiers:            buildActionRollModifiers(step.Args, "modifiers"),
		ReplaceHopeWithArmor: optionalBool(step.Args, "replace_hope_with_armor", false),
		Context:              contextValue,
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("action_roll: %w", err)
	}
	ensureRollOutcomeState(state)
	state.rollOutcomes[response.GetRollSeq()] = actionRollResultFromResponse(response)
	state.lastRollSeq = response.GetRollSeq()
	if want, ok := readInt(step.Args, "expect_total"); ok && int(response.GetTotal()) != want {
		return r.failf("action_roll total = %d, want %d", response.GetTotal(), want)
	}
	r.logf("action roll: actor=%s roll_seq=%d", actorName, state.lastRollSeq)
	if err := r.requireEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func actionRollContextFromScenario(value string) (daggerheartv1.ActionRollContext, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return daggerheartv1.ActionRollContext_ACTION_ROLL_CONTEXT_UNSPECIFIED, nil
	case "move_silently", "move silently", "silent_movement", "silent movement":
		return daggerheartv1.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY, nil
	default:
		return daggerheartv1.ActionRollContext_ACTION_ROLL_CONTEXT_UNSPECIFIED, fmt.Errorf("unsupported action roll context %q", value)
	}
}

func (r *Runner) runReactionRollStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("reaction_roll requires actor")
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	difficulty := optionalInt(step.Args, "difficulty", 10)
	seed := uint64(optionalInt(step.Args, "seed", 0))

	if actorIDValue, err := actorID(state, actorName); err == nil {
		if seed == 0 {
			seedValue, err := chooseActionSeed(step.Args, difficulty)
			if err != nil {
				return err
			}
			seed = seedValue
		}
		response, err := r.env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
			CampaignId:   state.campaignID,
			SessionId:    state.sessionID,
			SceneId:      state.activeSceneID,
			CharacterId:  actorIDValue,
			Trait:        optionalString(step.Args, "trait", "instinct"),
			RollKind:     daggerheartv1.RollKind_ROLL_KIND_REACTION,
			Difficulty:   int32(difficulty),
			Advantage:    int32(optionalInt(step.Args, "advantage", 0)),
			Disadvantage: int32(optionalInt(step.Args, "disadvantage", 0)),
			Modifiers:    buildActionRollModifiers(step.Args, "modifiers"),
			Rng: &commonv1.RngRequest{
				Seed:     &seed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		})
		if err != nil {
			return fmt.Errorf("reaction_roll: %w", err)
		}
		ensureRollOutcomeState(state)
		state.rollOutcomes[response.GetRollSeq()] = actionRollResultFromResponse(response)
		state.lastRollSeq = response.GetRollSeq()
		r.logf("reaction roll: actor=%s roll_seq=%d", actorName, state.lastRollSeq)
		return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved)
	}

	adversaryIDValue, err := adversaryID(state, actorName)
	if err != nil {
		return err
	}
	if seed == 0 {
		seed = 42
	}
	response, err := r.env.daggerheartClient.SessionAdversaryAttackRoll(ctx, &daggerheartv1.SessionAdversaryAttackRollRequest{
		CampaignId:   state.campaignID,
		SessionId:    state.sessionID,
		SceneId:      state.activeSceneID,
		AdversaryId:  adversaryIDValue,
		Modifiers:    buildAdversaryRollModifiers(step.Args),
		Advantage:    int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage: int32(optionalInt(step.Args, "disadvantage", 0)),
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("reaction_roll adversary: %w", err)
	}
	state.lastAdversaryRollSeq = response.GetRollSeq()
	state.lastRollSeq = response.GetRollSeq()
	r.logf("reaction roll: actor=%s roll_seq=%d", actorName, state.lastRollSeq)
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved)
}

func (r *Runner) runDamageRollStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("damage_roll requires actor")
	}
	seed := optionalInt(step.Args, "seed", 0)
	modifier := optionalInt(step.Args, "modifier", optionalInt(step.Args, "damage_modifier", 0))
	critical := optionalBool(step.Args, "critical", false)

	actorIDValue, err := actorID(state, actorName)
	if err != nil {
		return err
	}
	request := &daggerheartv1.SessionDamageRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		SceneId:     state.activeSceneID,
		CharacterId: actorIDValue,
		Dice:        buildDamageDice(step.Args),
		Modifier:    int32(modifier),
		Critical:    critical,
	}
	if seed != 0 {
		seedValue := uint64(seed)
		request.Rng = &commonv1.RngRequest{
			Seed:     &seedValue,
			RollMode: commonv1.RollMode_REPLAY,
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionDamageRoll(ctx, request)
	if err != nil {
		return fmt.Errorf("damage_roll: %w", err)
	}
	state.lastDamageRollSeq = response.GetRollSeq()
	r.logf("damage roll: actor=%s roll_seq=%d", actorName, state.lastDamageRollSeq)
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved)
}

func (r *Runner) runAdversaryAttackRollStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("adversary_attack_roll requires actor")
	}
	seed := optionalInt(step.Args, "seed", 0)
	adversaryIDValue, err := adversaryID(state, actorName)
	if err != nil {
		return err
	}
	request := &daggerheartv1.SessionAdversaryAttackRollRequest{
		CampaignId:   state.campaignID,
		SessionId:    state.sessionID,
		SceneId:      state.activeSceneID,
		AdversaryId:  adversaryIDValue,
		Modifiers:    buildAdversaryRollModifiers(step.Args),
		Advantage:    int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage: int32(optionalInt(step.Args, "disadvantage", 0)),
	}
	if seed != 0 {
		seedValue := uint64(seed)
		request.Rng = &commonv1.RngRequest{
			Seed:     &seedValue,
			RollMode: commonv1.RollMode_REPLAY,
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionAdversaryAttackRoll(ctx, request)
	if err != nil {
		return fmt.Errorf("adversary_attack_roll: %w", err)
	}
	state.lastAdversaryRollSeq = response.GetRollSeq()
	r.logf("adversary attack roll: actor=%s roll_seq=%d", actorName, state.lastAdversaryRollSeq)
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved)
}

func (r *Runner) runApplyRollOutcomeStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastRollSeq
	}
	if rollSeq == 0 {
		return r.failf("apply_roll_outcome requires roll_seq")
	}
	request := &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollSeq,
	}
	targets, err := resolveOutcomeTargets(state, step.Args)
	if err != nil {
		return err
	}
	branches, err := resolveOutcomeBranches(step.Args, map[string]struct{}{
		"on_success":      {},
		"on_failure":      {},
		"on_hope":         {},
		"on_fear":         {},
		"on_success_hope": {},
		"on_failure_hope": {},
		"on_success_fear": {},
		"on_failure_fear": {},
		"on_critical":     {},
		"on_crit":         {},
	}, step.System)
	if err != nil {
		return err
	}
	if len(targets) > 0 {
		request.Targets = targets
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.ApplyRollOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), request)
	if err != nil {
		return fmt.Errorf("apply_roll_outcome: %w", err)
	}
	if response == nil {
		return r.failf("apply_roll_outcome: expected response")
	}
	if response.GetRequiresComplication() {
		if err := r.resolveOpenSessionGate(ctx, state, before); err != nil {
			return err
		}
	}
	if len(branches) > 0 {
		ensureRollOutcomeState(state)
		result, ok := state.rollOutcomes[rollSeq]
		if !ok {
			return r.failf("missing action roll outcome for roll_seq %d", rollSeq)
		}
		if err := runOutcomeBranchSteps(ctx, state, r, branches, []string{"on_success", "on_failure", "on_success_hope", "on_failure_hope", "on_success_fear", "on_failure_fear", "on_hope", "on_fear", "on_critical", "on_crit"}, func(branch string) bool {
			return evaluateActionOutcomeBranch(result, branch)
		}); err != nil {
			return err
		}
	}
	return r.requireAnyEventTypesAfterSeq(ctx, state, before, event.TypeOutcomeApplied, event.TypeOutcomeRejected)
}

func (r *Runner) runApplyAttackOutcomeStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastRollSeq
	}
	if rollSeq == 0 {
		return r.failf("apply_attack_outcome requires roll_seq")
	}
	targets, err := resolveAttackTargets(state, step.Args)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return r.failf("apply_attack_outcome requires targets")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.ApplyAttackOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyAttackOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollSeq,
		Targets:   targets,
	})
	if err != nil {
		return fmt.Errorf("apply_attack_outcome: %w", err)
	}
	return r.requireNoSessionEventsAfterSeq(ctx, state, before)
}

func (r *Runner) runApplyAdversaryAttackOutcomeStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastAdversaryRollSeq
	}
	if rollSeq == 0 {
		return r.failf("apply_adversary_attack_outcome requires roll_seq")
	}
	difficulty := optionalInt(step.Args, "difficulty", 10)
	targets, err := resolveOutcomeTargets(state, step.Args)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return r.failf("apply_adversary_attack_outcome requires targets")
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.ApplyAdversaryAttackOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  state.sessionID,
		SceneId:    state.activeSceneID,
		RollSeq:    rollSeq,
		Targets:    targets,
		Difficulty: int32(difficulty),
	})
	if err != nil {
		return fmt.Errorf("apply_adversary_attack_outcome: %w", err)
	}
	return r.requireNoSessionEventsAfterSeq(ctx, state, before)
}

func (r *Runner) runApplyReactionOutcomeStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastRollSeq
	}
	if rollSeq == 0 {
		return r.failf("apply_reaction_outcome requires roll_seq")
	}
	branches, err := resolveOutcomeBranches(step.Args, map[string]struct{}{
		"on_success":      {},
		"on_failure":      {},
		"on_hope":         {},
		"on_fear":         {},
		"on_success_hope": {},
		"on_failure_hope": {},
		"on_success_fear": {},
		"on_failure_fear": {},
		"on_critical":     {},
		"on_crit":         {},
	}, step.System)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.ApplyReactionOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyReactionOutcomeRequest{
		SessionId: state.sessionID,
		SceneId:   state.activeSceneID,
		RollSeq:   rollSeq,
	})
	if err != nil {
		return fmt.Errorf("apply_reaction_outcome: %w", err)
	}
	if response == nil {
		return r.failf("apply_reaction_outcome: expected response")
	}
	if len(branches) > 0 {
		if err := runOutcomeBranchSteps(ctx, state, r, branches, []string{"on_success", "on_failure", "on_success_hope", "on_failure_hope", "on_success_fear", "on_failure_fear", "on_hope", "on_fear", "on_critical", "on_crit"}, func(branch string) bool {
			return evaluateReactionOutcomeBranch(response.GetResult(), branch)
		}); err != nil {
			return err
		}
		return nil
	}
	return r.requireNoSessionEventsAfterSeq(ctx, state, before)
}

func (r *Runner) runMitigateDamageStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("mitigate_damage target is required")
	}
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	armor := optionalInt(step.Args, "armor", 0)
	if armor <= 0 {
		return nil
	}
	_, err = r.env.snapshotClient.PatchCharacterState(ctx, &gamev1.PatchCharacterStateRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Armor: int32(armor),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("patch character armor: %w", err)
	}
	return nil
}
