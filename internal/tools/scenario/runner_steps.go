package scenario

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (r *Runner) runStep(ctx context.Context, state *scenarioState, step Step) error {
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
	case "adversary":
		return r.runAdversaryStep(ctx, state, step)
	case "gm_fear":
		return r.runGMFearStep(ctx, state, step)
	case "reaction":
		return r.runReactionStep(ctx, state, step)
	case "gm_spend_fear":
		return r.runGMSpendFearStep(ctx, state, step)
	case "set_spotlight":
		return r.runSetSpotlightStep(ctx, state, step)
	case "clear_spotlight":
		return r.runClearSpotlightStep(ctx, state, step)
	case "apply_condition":
		return r.runApplyConditionStep(ctx, state, step)
	case "group_action":
		return r.runGroupActionStep(ctx, state, step)
	case "tag_team":
		return r.runTagTeamStep(ctx, state, step)
	case "rest":
		return r.runRestStep(ctx, state, step)
	case "downtime_move":
		return r.runDowntimeMoveStep(ctx, state, step)
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
	default:
		return r.failf("unknown step kind %q", step.Kind)
	}
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
		CampaignId:  state.campaignID,
		DisplayName: name,
		Role:        roleValue,
		Controller:  controllerValue,
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
	system := optionalString(step.Args, "system", "DAGGERHEART")
	gmMode := optionalString(step.Args, "gm_mode", "HUMAN")

	systemValue, err := parseGameSystem(system)
	if err != nil {
		return err
	}
	gmModeValue, err := parseGmMode(gmMode)
	if err != nil {
		return err
	}

	creator := optionalString(step.Args, "creator_display_name", "")
	if creator == "" {
		creator = "Scenario GM"
	}
	request := &gamev1.CreateCampaignRequest{
		Name:               name,
		System:             systemValue,
		GmMode:             gmModeValue,
		CreatorDisplayName: creator,
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
	state.ownerParticipantID = response.GetOwnerParticipant().GetId()
	r.logf("campaign created: id=%s owner_participant=%s", state.campaignID, state.ownerParticipantID)
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeCampaignCreated)
}

func (r *Runner) runStartSessionStep(ctx context.Context, state *scenarioState, step Step) error {
	if state.campaignID == "" {
		return r.failf("campaign is required before session")
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

	if err := r.applyDefaultDaggerheartProfile(ctx, state, characterID, step.Args); err != nil {
		return err
	}
	if err := r.applyOptionalCharacterState(ctx, state, characterID, step.Args); err != nil {
		return err
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
			_, err := r.env.characterClient.SetDefaultControl(withParticipantID(ctx, state.ownerParticipantID), &gamev1.SetDefaultControlRequest{
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

func (r *Runner) runAdversaryStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}

	name := requiredString(step.Args, "name")
	if name == "" {
		return r.failf("adversary name is required")
	}
	kind := optionalString(step.Args, "kind", "")
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	request := &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId: state.campaignID,
		Name:       name,
		Kind:       kind,
	}
	if state.sessionID != "" {
		request.SessionId = wrapperspb.String(state.sessionID)
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
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryCreated)
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
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorIDValue,
		Trait:       trait,
		Difficulty:  int32(difficulty),
		Modifiers:   buildActionRollModifiers(step.Args, "modifiers"),
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
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeReactionResolved); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runGMSpendFearStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	amount, ok := readInt(step.Args, "amount")
	if !ok {
		return r.failf("gm_spend_fear amount is required")
	}
	if amount < 0 {
		return r.failf("gm_spend_fear amount must be non-negative")
	}
	move := optionalString(step.Args, "move", "spotlight")
	description := optionalString(step.Args, "description", "")
	if target := optionalString(step.Args, "target", ""); target != "" {
		if description == "" {
			description = fmt.Sprintf("spotlight %s", target)
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.ApplyGmMove(ctx, &daggerheartv1.DaggerheartApplyGmMoveRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		Move:        move,
		FearSpent:   int32(amount),
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("apply gm move: %w", err)
	}
	if amount > 0 {
		if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeGMFearChanged, daggerheart.EventTypeGMMoveApplied); err != nil {
			return err
		}
	} else {
		if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeGMMoveApplied); err != nil {
			return err
		}
	}
	state.gmFear = int(response.GetGmFearAfter())
	return nil
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
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	add := readStringSlice(step.Args, "add")
	remove := readStringSlice(step.Args, "remove")
	lifeState := optionalString(step.Args, "life_state", "")
	if len(add) == 0 && len(remove) == 0 && lifeState == "" {
		return r.failf("apply_condition requires add, remove, or life_state")
	}
	addValues, err := parseConditions(add)
	if err != nil {
		return err
	}
	removeValues, err := parseConditions(remove)
	if err != nil {
		return err
	}
	if lifeState != "" {
		if _, err := parseLifeState(lifeState); err != nil {
			return err
		}
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	request := &daggerheartv1.DaggerheartApplyConditionsRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		Add:         addValues,
		Remove:      removeValues,
		Source:      optionalString(step.Args, "source", ""),
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
		eventTypes = append(eventTypes, daggerheart.EventTypeConditionChanged)
	}
	if lifeState != "" {
		eventTypes = append(eventTypes, daggerheart.EventTypeCharacterStatePatched)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, eventTypes...)
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
		actorIDValue, err := actorID(state, name)
		if err != nil {
			return err
		}
		supporters = append(supporters, &daggerheartv1.GroupActionSupporter{
			CharacterId: actorIDValue,
			Trait:       trait,
			Modifiers:   buildActionRollModifiers(item, "modifiers"),
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
		LeaderCharacterId: leaderID,
		LeaderTrait:       leaderTrait,
		Difficulty:        int32(difficulty),
		LeaderModifiers:   leaderModifiers,
		LeaderRng: &commonv1.RngRequest{
			Seed:     &leaderSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
		Supporters: supporters,
	})
	if err != nil {
		return fmt.Errorf("group_action: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeGroupActionResolved); err != nil {
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
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeTagTeamResolved); err != nil {
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
	partySize := optionalInt(step.Args, "party_size", len(state.actors))
	if partySize <= 0 {
		partySize = len(state.actors)
	}
	interrupted := optionalBool(step.Args, "interrupted", false)
	seed := optionalInt(step.Args, "seed", 0)

	characterNames := readStringSlice(step.Args, "characters")
	characterIDs, err := resolveCharacterList(state, step.Args, "characters")
	if err != nil {
		return err
	}
	if len(characterIDs) == 0 {
		characterIDs = allActorIDs(state)
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
	psi := int32(partySize)
	rest := &daggerheartv1.DaggerheartRestRequest{
		RestType:    parsedRestType,
		Interrupted: interrupted,
		PartySize:   psi,
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
		CampaignId:   state.campaignID,
		CharacterIds: characterIDs,
		Rest:         rest,
	})
	if err != nil {
		return fmt.Errorf("rest: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeRestTaken); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
}

func (r *Runner) runDowntimeMoveStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("downtime_move target is required")
	}
	move := requiredString(step.Args, "move")
	if move == "" {
		return r.failf("downtime_move move is required")
	}
	prepareWithGroup := optionalBool(step.Args, "prepare_with_group", false)

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, name)
	if err != nil {
		return err
	}

	parsedMove, err := parseDowntimeMove(move)
	if err != nil {
		return err
	}
	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.ApplyDowntimeMove(ctxWithSession, &daggerheartv1.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		Move: &daggerheartv1.DaggerheartDowntimeRequest{
			Move:             parsedMove,
			PrepareWithGroup: prepareWithGroup,
		},
	})
	if err != nil {
		return fmt.Errorf("downtime_move: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDowntimeMoveApplied); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
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
	_, err = r.env.daggerheartClient.ApplyDeathMove(ctxWithSession, request)
	if err != nil {
		return fmt.Errorf("death_move: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDeathMoveResolved); err != nil {
		return err
	}
	return r.assertExpectedDeltas(ctx, state, expectedSpec, expectedBefore)
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
		CharacterId: characterID,
	})
	if err != nil {
		return fmt.Errorf("blaze_of_glory: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeBlazeOfGloryResolved, event.TypeCharacterDeleted)
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

	actionSeed, err := chooseActionSeed(step.Args, difficulty)
	if err != nil {
		return err
	}
	damageSeed := actionSeed + 1

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
			CampaignId:        state.campaignID,
			SessionId:         state.sessionID,
			CharacterId:       attackerID,
			Trait:             trait,
			Difficulty:        int32(difficulty),
			Modifiers:         buildActionRollModifiers(step.Args, "modifiers"),
			TargetId:          targetID,
			DamageDice:        buildDamageDice(step.Args),
			Damage:            buildDamageSpec(step.Args, attackerID, "attack"),
			RequireDamageRoll: true,
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
		if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAttackResolved); err != nil {
			return err
		}
		if response.GetDamageApplied() != nil {
			if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
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
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
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
		return fmt.Errorf("attack action roll: %w", err)
	}

	rollOutcomeResponse, err := r.env.daggerheartClient.ApplyRollOutcome(ctxWithMeta, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
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
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   []string{targetID},
	})
	if err != nil {
		return fmt.Errorf("attack outcome: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAttackResolved); err != nil {
		return err
	}

	if attackOutcome.GetResult() != nil && attackOutcome.GetResult().GetSuccess() {
		dice := buildDamageDice(step.Args)
		critical := attackOutcome.GetResult().GetCrit()
		damageRoll, err := r.env.daggerheartClient.SessionDamageRoll(ctx, &daggerheartv1.SessionDamageRollRequest{
			CampaignId:  state.campaignID,
			SessionId:   state.sessionID,
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
			if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryUpdated); err != nil {
				return err
			}
		}
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

	targetNames := readStringSlice(step.Args, "targets")
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
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   targetIDs,
	})
	if err != nil {
		return fmt.Errorf("multi_attack outcome: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAttackResolved); err != nil {
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
					if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryUpdated); err != nil {
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
				CharacterId:       target.id,
				Damage:            buildDamageRequest(step.Args, attackerID, "attack", damageRoll.GetTotal()),
				RollSeq:           &damageRoll.RollSeq,
				RequireDamageRoll: true,
			})
			if err != nil {
				return fmt.Errorf("multi_attack apply damage: %w", err)
			}
			if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
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
	targetID, err := actorID(state, name)
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
	stateBefore, err := r.getCharacterState(ctx, state, targetID)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.ApplyDamage(ctxWithSession, &daggerheartv1.DaggerheartApplyDamageRequest{
		CampaignId:  state.campaignID,
		CharacterId: targetID,
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
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
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

func (r *Runner) runAdversaryAttackStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	targetName := requiredString(step.Args, "target")
	if actorName == "" || targetName == "" {
		return r.failf("adversary_attack requires actor and target")
	}
	difficulty := optionalInt(step.Args, "difficulty", 10)
	adversaryIDValue, err := adversaryID(state, actorName)
	if err != nil {
		return err
	}
	targetCharacterID, err := actorID(state, targetName)
	if err != nil {
		return err
	}

	expectedSpec, expectedBefore, err := r.captureExpectedDeltas(ctx, state, step.Args, targetName)
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
	stateBefore, err := r.getCharacterState(ctx, state, targetCharacterID)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionAdversaryAttackFlow(ctx, &daggerheartv1.SessionAdversaryAttackFlowRequest{
		CampaignId:        state.campaignID,
		SessionId:         state.sessionID,
		AdversaryId:       adversaryIDValue,
		TargetId:          targetCharacterID,
		Difficulty:        int32(difficulty),
		AttackModifier:    int32(optionalInt(step.Args, "attack_modifier", 0)),
		Advantage:         int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage:      int32(optionalInt(step.Args, "disadvantage", 0)),
		DamageDice:        buildDamageDice(step.Args),
		Damage:            buildDamageSpec(step.Args, "", "adversary_attack"),
		RequireDamageRoll: true,
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
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryAttackResolved); err != nil {
		return err
	}
	if response.GetDamageApplied() != nil {
		if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageApplied); err != nil {
			return err
		}
		if err := r.assertDamageFlags(ctx, state, before, targetCharacterID, step.Args); err != nil {
			return err
		}
		if expectDamageEffect(step.Args, response.GetDamageRoll()) {
			stateAfter, err := r.getCharacterState(ctx, state, targetCharacterID)
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
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeLoadoutSwapped)
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
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCountdownCreated)
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
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCountdownUpdated)
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
		CountdownId: countdownID,
		Reason:      optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("countdown_delete: %w", err)
	}
	if err := r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeCountdownDeleted); err != nil {
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

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	actorIDValue, err := actorID(state, actorName)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorIDValue,
		Trait:       trait,
		RollKind:    daggerheartv1.RollKind_ROLL_KIND_ACTION,
		Difficulty:  int32(difficulty),
		Modifiers:   buildActionRollModifiers(step.Args, "modifiers"),
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("action_roll: %w", err)
	}
	state.lastRollSeq = response.GetRollSeq()
	r.logf("action roll: actor=%s roll_seq=%d", actorName, state.lastRollSeq)
	return r.requireEventTypesAfterSeq(ctx, state, before, event.TypeRollResolved)
}

func (r *Runner) runReactionRollStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureSession(ctx, state); err != nil {
		return err
	}
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		return r.failf("reaction_roll requires actor")
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

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	actorIDValue, err := actorID(state, actorName)
	if err != nil {
		return err
	}
	response, err := r.env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorIDValue,
		Trait:       trait,
		RollKind:    daggerheartv1.RollKind_ROLL_KIND_REACTION,
		Difficulty:  int32(difficulty),
		Modifiers:   buildActionRollModifiers(step.Args, "modifiers"),
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		return fmt.Errorf("reaction_roll: %w", err)
	}
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
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDamageRollResolved)
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
		CampaignId:     state.campaignID,
		SessionId:      state.sessionID,
		AdversaryId:    adversaryIDValue,
		AttackModifier: int32(optionalInt(step.Args, "attack_modifier", 0)),
		Advantage:      int32(optionalInt(step.Args, "advantage", 0)),
		Disadvantage:   int32(optionalInt(step.Args, "disadvantage", 0)),
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
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryRollResolved)
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
		RollSeq:   rollSeq,
	}
	targets, err := resolveOutcomeTargets(state, step.Args)
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
	_, err = r.env.daggerheartClient.ApplyRollOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), request)
	if err != nil {
		return fmt.Errorf("apply_roll_outcome: %w", err)
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
		RollSeq:   rollSeq,
		Targets:   targets,
	})
	if err != nil {
		return fmt.Errorf("apply_attack_outcome: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAttackResolved)
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
		RollSeq:    rollSeq,
		Targets:    targets,
		Difficulty: int32(difficulty),
	})
	if err != nil {
		return fmt.Errorf("apply_adversary_attack_outcome: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeAdversaryAttackResolved)
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

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	_, err = r.env.daggerheartClient.ApplyReactionOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyReactionOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollSeq,
	})
	if err != nil {
		return fmt.Errorf("apply_reaction_outcome: %w", err)
	}
	return r.requireEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeReactionResolved)
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
