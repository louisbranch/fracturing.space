//go:build scenario

package game

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const scenarioLuaGlob = "internal/test/game/scenarios/*.lua"

type scenarioEnv struct {
	campaignClient    gamev1.CampaignServiceClient
	sessionClient     gamev1.SessionServiceClient
	characterClient   gamev1.CharacterServiceClient
	snapshotClient    gamev1.SnapshotServiceClient
	eventClient       gamev1.EventServiceClient
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	userID            string
}

type scenarioState struct {
	campaignID           string
	sessionID            string
	actors               map[string]string
	adversaries          map[string]string
	countdowns           map[string]string
	gmFear               int
	userID               string
	lastRollSeq          uint64
	lastDamageRollSeq    uint64
	lastAdversaryRollSeq uint64
}

func TestScenarioScripts(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
	defer stopServer()
	userID := createAuthUser(t, authAddr, "Scenario GM")

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	env := scenarioEnv{
		campaignClient:    gamev1.NewCampaignServiceClient(conn),
		sessionClient:     gamev1.NewSessionServiceClient(conn),
		characterClient:   gamev1.NewCharacterServiceClient(conn),
		snapshotClient:    gamev1.NewSnapshotServiceClient(conn),
		eventClient:       gamev1.NewEventServiceClient(conn),
		daggerheartClient: daggerheartv1.NewDaggerheartServiceClient(conn),
		userID:            userID,
	}

	paths := scenarioLuaPaths(t)
	for _, path := range paths {
		path := path
		scenario, err := loadScenarioFromFile(path)
		if err != nil {
			t.Fatalf("load scenario %s: %v", path, err)
		}
		name := scenario.Name
		if name == "" {
			name = filepath.Base(path)
		}
		t.Run(name, func(t *testing.T) {
			runScenario(t, env, scenario)
		})
	}
}

func scenarioLuaPaths(t *testing.T) []string {
	t.Helper()

	pattern := filepath.Join(repoRoot(t), scenarioLuaGlob)
	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob scenarios: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no scenarios found for %s", pattern)
	}
	sort.Strings(paths)
	return paths
}

func runScenario(t *testing.T, env scenarioEnv, scenario *Scenario) {
	t.Helper()

	state := &scenarioState{
		actors:      map[string]string{},
		adversaries: map[string]string{},
		countdowns:  map[string]string{},
		userID:      env.userID,
	}
	for index, step := range scenario.Steps {
		step := step
		t.Run(fmt.Sprintf("%02d_%s", index+1, step.Kind), func(t *testing.T) {
			runStep(t, env, state, step)
		})
	}
}

func runStep(t *testing.T, env scenarioEnv, state *scenarioState, step Step) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), scenarioTimeout())
	defer cancel()

	switch step.Kind {
	case "campaign":
		runCampaignStep(t, ctx, env, state, step)
	case "start_session":
		runStartSessionStep(t, ctx, env, state, step)
	case "end_session":
		runEndSessionStep(t, ctx, env, state)
	case "character":
		runCharacterStep(t, ctx, env, state, step)
	case "prefab":
		runPrefabStep(t, ctx, env, state, step)
	case "adversary":
		runAdversaryStep(t, ctx, env, state, step)
	case "gm_fear":
		runGMFearStep(t, ctx, env, state, step)
	case "reaction":
		runReactionStep(t, ctx, env, state, step)
	case "gm_spend_fear":
		runGMSpendFearStep(t, ctx, env, state, step)
	case "apply_condition":
		runApplyConditionStep(t, ctx, env, state, step)
	case "group_action":
		runGroupActionStep(t, ctx, env, state, step)
	case "tag_team":
		runTagTeamStep(t, ctx, env, state, step)
	case "rest":
		runRestStep(t, ctx, env, state, step)
	case "downtime_move":
		runDowntimeMoveStep(t, ctx, env, state, step)
	case "death_move":
		runDeathMoveStep(t, ctx, env, state, step)
	case "blaze_of_glory":
		runBlazeOfGloryStep(t, ctx, env, state, step)
	case "attack":
		runAttackStep(t, ctx, env, state, step)
	case "multi_attack":
		runMultiAttackStep(t, ctx, env, state, step)
	case "combined_damage":
		runCombinedDamageStep(t, ctx, env, state, step)
	case "adversary_attack":
		runAdversaryAttackStep(t, ctx, env, state, step)
	case "swap_loadout":
		runSwapLoadoutStep(t, ctx, env, state, step)
	case "countdown_create":
		runCountdownCreateStep(t, ctx, env, state, step)
	case "countdown_update":
		runCountdownUpdateStep(t, ctx, env, state, step)
	case "countdown_delete":
		runCountdownDeleteStep(t, ctx, env, state, step)
	case "action_roll":
		runActionRollStep(t, ctx, env, state, step)
	case "reaction_roll":
		runReactionRollStep(t, ctx, env, state, step)
	case "damage_roll":
		runDamageRollStep(t, ctx, env, state, step)
	case "adversary_attack_roll":
		runAdversaryAttackRollStep(t, ctx, env, state, step)
	case "apply_roll_outcome":
		runApplyRollOutcomeStep(t, ctx, env, state, step)
	case "apply_attack_outcome":
		runApplyAttackOutcomeStep(t, ctx, env, state, step)
	case "apply_adversary_attack_outcome":
		runApplyAdversaryAttackOutcomeStep(t, ctx, env, state, step)
	case "apply_reaction_outcome":
		runApplyReactionOutcomeStep(t, ctx, env, state, step)
	case "mitigate_damage":
		runMitigateDamageStep(t, ctx, env, state, step)
	default:
		t.Fatalf("unknown step kind %q", step.Kind)
	}
}

func runCampaignStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	if state.campaignID != "" {
		t.Fatalf("campaign already created")
	}
	name := requiredString(step.Args, "name")
	if name == "" {
		t.Fatal("campaign name is required")
	}
	system := optionalString(step.Args, "system", "DAGGERHEART")
	gmMode := optionalString(step.Args, "gm_mode", "HUMAN")

	request := &gamev1.CreateCampaignRequest{
		Name:   name,
		System: parseGameSystem(t, system),
		GmMode: parseGmMode(t, gmMode),
	}
	if theme := optionalString(step.Args, "theme", ""); theme != "" {
		request.ThemePrompt = theme
	}
	if creator := optionalString(step.Args, "creator_display_name", ""); creator != "" {
		request.CreatorDisplayName = creator
	}

	before := latestSeq(t, ctx, env, state)
	response, err := env.campaignClient.CreateCampaign(withUserID(ctx, state.userID), request)
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if response.GetCampaign() == nil {
		t.Fatal("expected campaign response")
	}
	state.campaignID = response.GetCampaign().GetId()
	requireEventTypesAfterSeq(t, ctx, env, state, before, event.TypeCampaignCreated)
}

func runStartSessionStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	if state.campaignID == "" {
		t.Fatal("campaign is required before session")
	}
	name := optionalString(step.Args, "name", "Scenario Session")
	request := &gamev1.StartSessionRequest{CampaignId: state.campaignID, Name: name}

	before := latestSeq(t, ctx, env, state)
	response, err := env.sessionClient.StartSession(ctx, request)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if response.GetSession() == nil {
		t.Fatal("expected session")
	}
	state.sessionID = response.GetSession().GetId()
	requireEventTypesAfterSeq(t, ctx, env, state, before, event.TypeSessionStarted)
}

func runEndSessionStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState) {
	if state.sessionID == "" {
		t.Fatal("session is required to end")
	}
	before := latestSeq(t, ctx, env, state)
	_, err := env.sessionClient.EndSession(ctx, &gamev1.EndSessionRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
	})
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, event.TypeSessionEnded)
	state.sessionID = ""
}

func runCharacterStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureCampaign(t, state)

	name := requiredString(step.Args, "name")
	if name == "" {
		t.Fatal("character name is required")
	}
	kind := optionalString(step.Args, "kind", "PC")
	request := &gamev1.CreateCharacterRequest{
		CampaignId: state.campaignID,
		Name:       name,
		Kind:       parseCharacterKind(t, kind),
	}

	before := latestSeq(t, ctx, env, state)
	response, err := env.characterClient.CreateCharacter(ctx, request)
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	if response.GetCharacter() == nil {
		t.Fatal("expected character")
	}
	characterID := response.GetCharacter().GetId()
	state.actors[name] = characterID

	applyDefaultDaggerheartProfile(t, ctx, env, state, characterID, step.Args)
	applyOptionalCharacterState(t, ctx, env, state, characterID, step.Args)
	requireEventTypesAfterSeq(t, ctx, env, state, before, event.TypeCharacterCreated)
}

func runPrefabStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureCampaign(t, state)
	name := requiredString(step.Args, "name")
	if name == "" {
		t.Fatal("prefab name is required")
	}
	options := prefabOptions(name)
	step.Args["name"] = name
	for key, value := range options {
		step.Args[key] = value
	}
	runCharacterStep(t, ctx, env, state, step)
}

func runAdversaryStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureCampaign(t, state)

	name := requiredString(step.Args, "name")
	if name == "" {
		t.Fatal("adversary name is required")
	}
	kind := optionalString(step.Args, "kind", "")
	before := latestSeq(t, ctx, env, state)
	request := &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId: state.campaignID,
		Name:       name,
		Kind:       kind,
	}
	if state.sessionID != "" {
		request.SessionId = wrapperspb.String(state.sessionID)
	}
	response, err := env.daggerheartClient.CreateAdversary(ctx, request)
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if response.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}
	state.adversaries[name] = response.GetAdversary().GetId()
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAdversaryCreated)
}

func runGMFearStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureCampaign(t, state)
	value, ok := readInt(step.Args, "value")
	if !ok {
		t.Fatal("gm_fear value is required")
	}
	_, err := env.snapshotClient.UpdateSnapshotState(ctx, &gamev1.UpdateSnapshotStateRequest{
		CampaignId: state.campaignID,
		SystemSnapshotUpdate: &gamev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: int32(value)},
		},
	})
	if err != nil {
		t.Fatalf("update snapshot: %v", err)
	}
	snapshot := getSnapshot(t, ctx, env, state)
	if snapshot.GetGmFear() != int32(value) {
		t.Fatalf("gm_fear = %d, want %d", snapshot.GetGmFear(), value)
	}
	state.gmFear = value
}

func runReactionStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		t.Fatal("reaction requires actor")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	seed := uint64(optionalInt(step.Args, "seed", 0))
	if seed == 0 {
		seed = chooseActionSeed(t, step.Args, difficulty)
	}

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, actorName)

	before := latestSeq(t, ctx, env, state)
	response, err := env.daggerheartClient.SessionReactionFlow(ctx, &daggerheartv1.SessionReactionFlowRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorID(t, state, actorName),
		Trait:       trait,
		Difficulty:  int32(difficulty),
		Modifiers:   buildActionRollModifiers(step.Args, "modifiers"),
		ReactionRng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("reaction flow: %v", err)
	}
	if response.GetActionRoll() == nil {
		t.Fatal("expected reaction action roll")
	}
	state.lastRollSeq = response.GetActionRoll().GetRollSeq()
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeReactionResolved)
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runGMSpendFearStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	amount, ok := readInt(step.Args, "amount")
	if !ok || amount <= 0 {
		t.Fatal("gm_spend_fear amount is required")
	}
	move := optionalString(step.Args, "move", "spotlight")
	description := optionalString(step.Args, "description", "")
	if target := optionalString(step.Args, "target", ""); target != "" {
		if description == "" {
			description = fmt.Sprintf("spotlight %s", target)
		}
	}

	before := latestSeq(t, ctx, env, state)
	response, err := env.daggerheartClient.ApplyGmMove(ctx, &daggerheartv1.DaggerheartApplyGmMoveRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		Move:        move,
		FearSpent:   int32(amount),
		Description: description,
	})
	if err != nil {
		t.Fatalf("apply gm move: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeGMFearChanged, daggerheart.EventTypeGMMoveApplied)
	state.gmFear = int(response.GetGmFearAfter())
}

func runApplyConditionStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("apply_condition target is required")
	}
	characterID := actorID(t, state, name)
	add := readStringSlice(step.Args, "add")
	remove := readStringSlice(step.Args, "remove")
	if len(add) == 0 && len(remove) == 0 {
		t.Fatal("apply_condition requires add or remove")
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.ApplyConditions(withSessionID(ctx, state.sessionID), &daggerheartv1.DaggerheartApplyConditionsRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		Add:         parseConditions(t, add),
		Remove:      parseConditions(t, remove),
		Source:      optionalString(step.Args, "source", ""),
	})
	if err != nil {
		t.Fatalf("apply conditions: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeConditionChanged)
}

func runGroupActionStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	leaderName := requiredString(step.Args, "leader")
	leaderTrait := requiredString(step.Args, "leader_trait")
	difficulty := optionalInt(step.Args, "difficulty", 0)
	if leaderName == "" || leaderTrait == "" || difficulty == 0 {
		t.Fatal("group_action requires leader, leader_trait, and difficulty")
	}

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, leaderName)

	supportersRaw, ok := step.Args["supporters"]
	if !ok {
		t.Fatal("group_action requires supporters")
	}
	supporterList, ok := supportersRaw.([]any)
	if !ok || len(supporterList) == 0 {
		t.Fatal("group_action supporters must be a list")
	}

	baseSeed := uint64(optionalInt(step.Args, "seed", 42))
	leaderSeed := resolveOutcomeSeed(t, step.Args, "outcome", difficulty, baseSeed)
	leaderModifiers := buildActionRollModifiers(step.Args, "leader_modifiers")

	supporters := make([]*daggerheartv1.GroupActionSupporter, 0, len(supporterList))
	for index, entry := range supporterList {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("group_action supporter %d must be an object", index)
		}
		name := requiredString(item, "name")
		trait := requiredString(item, "trait")
		if name == "" || trait == "" {
			t.Fatalf("group_action supporter %d requires name and trait", index)
		}
		seed := resolveOutcomeSeed(t, item, "outcome", difficulty, baseSeed+uint64(index)+1)
		supporters = append(supporters, &daggerheartv1.GroupActionSupporter{
			CharacterId: actorID(t, state, name),
			Trait:       trait,
			Modifiers:   buildActionRollModifiers(item, "modifiers"),
			Rng: &commonv1.RngRequest{
				Seed:     &seed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		})
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.SessionGroupActionFlow(ctx, &daggerheartv1.SessionGroupActionFlowRequest{
		CampaignId:        state.campaignID,
		SessionId:         state.sessionID,
		LeaderCharacterId: actorID(t, state, leaderName),
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
		t.Fatalf("group_action: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeGroupActionResolved)
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runTagTeamStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	firstName := requiredString(step.Args, "first")
	secondName := requiredString(step.Args, "second")
	selectedName := requiredString(step.Args, "selected")
	firstTrait := requiredString(step.Args, "first_trait")
	secondTrait := requiredString(step.Args, "second_trait")
	difficulty := optionalInt(step.Args, "difficulty", 0)
	if firstName == "" || secondName == "" || selectedName == "" || firstTrait == "" || secondTrait == "" || difficulty == 0 {
		t.Fatal("tag_team requires first, second, selected, first_trait, second_trait, and difficulty")
	}

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, selectedName)

	baseSeed := uint64(optionalInt(step.Args, "seed", 42))
	firstSeed := resolveOutcomeSeed(t, step.Args, "first_outcome", difficulty, baseSeed)
	secondSeed := resolveOutcomeSeed(t, step.Args, "second_outcome", difficulty, baseSeed+1)
	selectedOutcome := optionalString(step.Args, "outcome", "")
	if selectedOutcome != "" {
		if selectedName == firstName {
			firstSeed = resolveOutcomeSeed(t, map[string]any{"outcome": selectedOutcome}, "outcome", difficulty, firstSeed)
		} else if selectedName == secondName {
			secondSeed = resolveOutcomeSeed(t, map[string]any{"outcome": selectedOutcome}, "outcome", difficulty, secondSeed)
		}
	}

	firstID := actorID(t, state, firstName)
	secondID := actorID(t, state, secondName)
	selectedID := actorID(t, state, selectedName)

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.SessionTagTeamFlow(ctx, &daggerheartv1.SessionTagTeamFlowRequest{
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
		t.Fatalf("tag_team: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeTagTeamResolved)
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runRestStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	restType := optionalString(step.Args, "type", "")
	if restType == "" {
		restType = optionalString(step.Args, "rest_type", "")
	}
	if restType == "" {
		t.Fatal("rest type is required")
	}
	partySize := optionalInt(step.Args, "party_size", len(state.actors))
	if partySize <= 0 {
		partySize = len(state.actors)
	}
	interrupted := optionalBool(step.Args, "interrupted", false)
	seed := optionalInt(step.Args, "seed", 0)

	characterNames := readStringSlice(step.Args, "characters")
	characterIDs := resolveCharacterList(t, state, step.Args, "characters")
	if len(characterIDs) == 0 {
		characterIDs = allActorIDs(state)
	}

	fallbackName := ""
	if len(characterNames) == 1 {
		fallbackName = characterNames[0]
	}
	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, fallbackName)

	rest := &daggerheartv1.DaggerheartRestRequest{
		RestType:    parseRestType(t, restType),
		Interrupted: interrupted,
		PartySize:   int32(partySize),
	}
	if seed != 0 {
		seedValue := uint64(seed)
		rest.Rng = &commonv1.RngRequest{
			Seed:     &seedValue,
			RollMode: commonv1.RollMode_REPLAY,
		}
	}

	before := latestSeq(t, ctx, env, state)
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err := env.daggerheartClient.ApplyRest(ctxWithSession, &daggerheartv1.DaggerheartApplyRestRequest{
		CampaignId:   state.campaignID,
		CharacterIds: characterIDs,
		Rest:         rest,
	})
	if err != nil {
		t.Fatalf("rest: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeRestTaken)
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runDowntimeMoveStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("downtime_move target is required")
	}
	move := requiredString(step.Args, "move")
	if move == "" {
		t.Fatal("downtime_move move is required")
	}
	prepareWithGroup := optionalBool(step.Args, "prepare_with_group", false)

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, name)

	before := latestSeq(t, ctx, env, state)
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err := env.daggerheartClient.ApplyDowntimeMove(ctxWithSession, &daggerheartv1.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  state.campaignID,
		CharacterId: actorID(t, state, name),
		Move: &daggerheartv1.DaggerheartDowntimeRequest{
			Move:             parseDowntimeMove(t, move),
			PrepareWithGroup: prepareWithGroup,
		},
	})
	if err != nil {
		t.Fatalf("downtime_move: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDowntimeMoveApplied)
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runDeathMoveStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("death_move target is required")
	}
	move := requiredString(step.Args, "move")
	if move == "" {
		t.Fatal("death_move move is required")
	}
	hpClear, hpOk := readInt(step.Args, "hp_clear")
	stressClear, stressOk := readInt(step.Args, "stress_clear")
	seed := optionalInt(step.Args, "seed", 0)

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, name)

	request := &daggerheartv1.DaggerheartApplyDeathMoveRequest{
		CampaignId:  state.campaignID,
		CharacterId: actorID(t, state, name),
		Move:        parseDeathMove(t, move),
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

	before := latestSeq(t, ctx, env, state)
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err := env.daggerheartClient.ApplyDeathMove(ctxWithSession, request)
	if err != nil {
		t.Fatalf("death_move: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDeathMoveResolved)
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runBlazeOfGloryStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("blaze_of_glory target is required")
	}

	before := latestSeq(t, ctx, env, state)
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err := env.daggerheartClient.ResolveBlazeOfGlory(ctxWithSession, &daggerheartv1.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  state.campaignID,
		CharacterId: actorID(t, state, name),
	})
	if err != nil {
		t.Fatalf("blaze_of_glory: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeBlazeOfGloryResolved, event.TypeCharacterDeleted)
}

func runAttackStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	targetName := requiredString(step.Args, "target")
	if actorName == "" || targetName == "" {
		t.Fatal("attack requires actor and target")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	attackerID := actorID(t, state, actorName)
	targetID, targetIsAdversary := resolveTargetID(t, state, targetName)

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, actorName)

	actionSeed := chooseActionSeed(t, step.Args, difficulty)
	damageSeed := actionSeed + 1

	before := latestSeq(t, ctx, env, state)
	if !targetIsAdversary {
		stateBefore := getCharacterState(t, ctx, env, state, targetID)
		response, err := env.daggerheartClient.SessionAttackFlow(ctx, &daggerheartv1.SessionAttackFlowRequest{
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
			t.Fatalf("attack flow: %v", err)
		}
		requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAttackResolved)
		if response.GetDamageApplied() != nil {
			requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDamageApplied)
			assertDamageFlags(t, ctx, env, state, before, targetID, step.Args)
			if expectDamageEffect(step.Args, response.GetDamageRoll()) {
				stateAfter := getCharacterState(t, ctx, env, state, targetID)
				if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
					t.Fatalf("expected damage to affect hp or armor for %s", targetName)
				}
			}
		}
		assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
		return
	}

	ctxWithMeta := withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID)
	rollResp, err := env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
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
		t.Fatalf("attack action roll: %v", err)
	}

	_, err = env.daggerheartClient.ApplyRollOutcome(ctxWithMeta, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		t.Fatalf("attack roll outcome: %v", err)
	}

	attackOutcome, err := env.daggerheartClient.ApplyAttackOutcome(ctxWithMeta, &daggerheartv1.DaggerheartApplyAttackOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   []string{targetID},
	})
	if err != nil {
		t.Fatalf("attack outcome: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAttackResolved)

	if attackOutcome.GetResult() != nil && attackOutcome.GetResult().GetSuccess() {
		dice := buildDamageDice(step.Args)
		if len(dice) == 0 {
			t.Fatal("attack requires damage_dice")
		}
		critical := attackOutcome.GetResult().GetCrit()
		damageRoll, err := env.daggerheartClient.SessionDamageRoll(ctx, &daggerheartv1.SessionDamageRollRequest{
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
			t.Fatalf("attack damage roll: %v", err)
		}
		if applyAdversaryDamage(t, ctx, env, state, targetID, targetName, damageRoll, step.Args) {
			requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAdversaryUpdated)
		}
	}
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runMultiAttackStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		t.Fatal("multi_attack requires actor")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	attackerID := actorID(t, state, actorName)
	if attackerID == "" {
		t.Fatalf("multi_attack actor %q not found", actorName)
	}

	targetNames := readStringSlice(step.Args, "targets")
	if len(targetNames) == 0 {
		t.Fatal("multi_attack requires targets")
	}
	type attackTarget struct {
		id        string
		name      string
		adversary bool
	}
	targets := make([]attackTarget, 0, len(targetNames))
	targetIDs := make([]string, 0, len(targetNames))
	for _, name := range targetNames {
		id, isAdversary := resolveTargetID(t, state, name)
		targets = append(targets, attackTarget{id: id, name: name, adversary: isAdversary})
		targetIDs = append(targetIDs, id)
	}

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, actorName)

	actionSeed := chooseActionSeed(t, step.Args, difficulty)
	damageSeed := actionSeed + 1

	before := latestSeq(t, ctx, env, state)
	rollResp, err := env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
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
		t.Fatalf("multi_attack action roll: %v", err)
	}

	ctxWithMeta := withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID)
	_, err = env.daggerheartClient.ApplyRollOutcome(ctxWithMeta, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		t.Fatalf("multi_attack roll outcome: %v", err)
	}

	attackOutcome, err := env.daggerheartClient.ApplyAttackOutcome(ctxWithMeta, &daggerheartv1.DaggerheartApplyAttackOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollResp.GetRollSeq(),
		Targets:   targetIDs,
	})
	if err != nil {
		t.Fatalf("multi_attack outcome: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAttackResolved)

	if attackOutcome.GetResult() != nil && attackOutcome.GetResult().GetSuccess() {
		dice := buildDamageDice(step.Args)
		requireDamageDice(t, step.Args, "multi_attack")
		critical := attackOutcome.GetResult().GetCrit()
		damageRoll, err := env.daggerheartClient.SessionDamageRoll(ctx, &daggerheartv1.SessionDamageRollRequest{
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
			t.Fatalf("multi_attack damage roll: %v", err)
		}

		expectedChange := adjustedDamageAmount(step.Args, damageRoll.GetTotal()) > 0
		for _, target := range targets {
			if target.adversary {
				if applyAdversaryDamage(t, ctx, env, state, target.id, target.name, damageRoll, step.Args) {
					requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAdversaryUpdated)
				}
				continue
			}
			stateBefore := getCharacterState(t, ctx, env, state, target.id)
			_, err := env.daggerheartClient.ApplyDamage(ctxWithMeta, &daggerheartv1.DaggerheartApplyDamageRequest{
				CampaignId:        state.campaignID,
				CharacterId:       target.id,
				Damage:            buildDamageRequest(step.Args, attackerID, "attack", damageRoll.GetTotal()),
				RollSeq:           &damageRoll.RollSeq,
				RequireDamageRoll: true,
			})
			if err != nil {
				t.Fatalf("multi_attack apply damage: %v", err)
			}
			requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDamageApplied)
			assertDamageFlags(t, ctx, env, state, before, target.id, step.Args)
			if expectedChange {
				stateAfter := getCharacterState(t, ctx, env, state, target.id)
				if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
					t.Fatalf("expected damage to affect hp or armor for %s", target.name)
				}
			}
		}
	}
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runCombinedDamageStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("combined_damage target is required")
	}
	targetID := actorID(t, state, name)

	sourcesRaw, ok := step.Args["sources"]
	if !ok {
		t.Fatal("combined_damage sources are required")
	}
	sourceList, ok := sourcesRaw.([]any)
	if !ok || len(sourceList) == 0 {
		t.Fatal("combined_damage sources must be a list")
	}

	amountTotal := 0
	sourceIDs := make([]string, 0, len(sourceList))
	for index, entry := range sourceList {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("combined_damage source %d must be an object", index)
		}
		amount, ok := readInt(item, "amount")
		if !ok || amount <= 0 {
			t.Fatalf("combined_damage source %d requires amount", index)
		}
		amountTotal += amount
		if sourceName := optionalString(item, "character", ""); sourceName != "" {
			sourceIDs = append(sourceIDs, actorID(t, state, sourceName))
		}
	}
	if amountTotal <= 0 {
		t.Fatal("combined_damage requires positive total damage")
	}

	before := latestSeq(t, ctx, env, state)
	stateBefore := getCharacterState(t, ctx, env, state, targetID)
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err := env.daggerheartClient.ApplyDamage(ctxWithSession, &daggerheartv1.DaggerheartApplyDamageRequest{
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
		t.Fatalf("combined_damage apply damage: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDamageApplied)
	assertDamageFlags(t, ctx, env, state, before, targetID, step.Args)
	if adjustedDamageAmount(step.Args, int32(amountTotal)) > 0 {
		stateAfter := getCharacterState(t, ctx, env, state, targetID)
		if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
			t.Fatalf("expected damage to affect hp or armor for %s", name)
		}
	}
}

func runAdversaryAttackStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	targetName := requiredString(step.Args, "target")
	if actorName == "" || targetName == "" {
		t.Fatal("adversary_attack requires actor and target")
	}
	difficulty := optionalInt(step.Args, "difficulty", 10)
	adversaryID := adversaryID(t, state, actorName)
	targetCharacterID := actorID(t, state, targetName)

	expectedSpec, expectedBefore := captureExpectedDeltas(t, ctx, env, state, step.Args, targetName)

	attackSeed := uint64(42)
	if seed := optionalInt(step.Args, "seed", 0); seed > 0 {
		attackSeed = uint64(seed)
	}
	damageSeed := attackSeed + 1

	before := latestSeq(t, ctx, env, state)
	stateBefore := getCharacterState(t, ctx, env, state, targetCharacterID)
	response, err := env.daggerheartClient.SessionAdversaryAttackFlow(ctx, &daggerheartv1.SessionAdversaryAttackFlowRequest{
		CampaignId:        state.campaignID,
		SessionId:         state.sessionID,
		AdversaryId:       adversaryID,
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
		t.Fatalf("adversary attack flow: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAdversaryAttackResolved)
	if response.GetDamageApplied() != nil {
		requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDamageApplied)
		assertDamageFlags(t, ctx, env, state, before, targetCharacterID, step.Args)
		if expectDamageEffect(step.Args, response.GetDamageRoll()) {
			stateAfter := getCharacterState(t, ctx, env, state, targetCharacterID)
			if stateAfter.GetHp() >= stateBefore.GetHp() && stateAfter.GetArmor() >= stateBefore.GetArmor() {
				t.Fatalf("expected damage to affect hp or armor for %s", targetName)
			}
		}
	}
	assertExpectedDeltas(t, ctx, env, state, expectedSpec, expectedBefore)
}

func runSwapLoadoutStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("swap_loadout target is required")
	}
	cardID := requiredString(step.Args, "card_id")
	if cardID == "" {
		t.Fatal("swap_loadout card_id is required")
	}
	recallCost := optionalInt(step.Args, "recall_cost", 0)
	inRest := optionalBool(step.Args, "in_rest", false)

	before := latestSeq(t, ctx, env, state)
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err := env.daggerheartClient.SwapLoadout(ctxWithSession, &daggerheartv1.DaggerheartSwapLoadoutRequest{
		CampaignId:  state.campaignID,
		CharacterId: actorID(t, state, name),
		Swap: &daggerheartv1.DaggerheartLoadoutSwapRequest{
			CardId:     cardID,
			RecallCost: int32(recallCost),
			InRest:     inRest,
		},
	})
	if err != nil {
		t.Fatalf("swap_loadout: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeLoadoutSwapped)
}

func runCountdownCreateStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	name := requiredString(step.Args, "name")
	if name == "" {
		t.Fatal("countdown_create name is required")
	}
	maxValue := optionalInt(step.Args, "max", 0)
	if maxValue <= 0 {
		maxValue = 4
	}

	before := latestSeq(t, ctx, env, state)
	request := &daggerheartv1.DaggerheartCreateCountdownRequest{
		CampaignId: state.campaignID,
		SessionId:  state.sessionID,
		Name:       name,
		Kind:       parseCountdownKind(t, optionalString(step.Args, "kind", "progress")),
		Current:    int32(optionalInt(step.Args, "current", 0)),
		Max:        int32(maxValue),
		Direction:  parseCountdownDirection(t, optionalString(step.Args, "direction", "increase")),
		Looping:    optionalBool(step.Args, "looping", false),
	}
	if countdownID := optionalString(step.Args, "countdown_id", ""); countdownID != "" {
		request.CountdownId = countdownID
	}
	response, err := env.daggerheartClient.CreateCountdown(ctx, request)
	if err != nil {
		t.Fatalf("countdown_create: %v", err)
	}
	if response.GetCountdown() == nil {
		t.Fatal("expected countdown")
	}
	state.countdowns[name] = response.GetCountdown().GetCountdownId()
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeCountdownCreated)
}

func runCountdownUpdateStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	countdownID := resolveCountdownID(t, state, step.Args)
	if countdownID == "" {
		t.Fatal("countdown_update countdown_id or name is required")
	}

	delta := optionalInt(step.Args, "delta", 0)
	current, hasCurrent := readInt(step.Args, "current")
	if delta == 0 && !hasCurrent {
		t.Fatal("countdown_update requires delta or current")
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

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.UpdateCountdown(ctx, request)
	if err != nil {
		t.Fatalf("countdown_update: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeCountdownUpdated)
}

func runCountdownDeleteStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	countdownID := resolveCountdownID(t, state, step.Args)
	if countdownID == "" {
		t.Fatal("countdown_delete countdown_id or name is required")
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.DeleteCountdown(ctx, &daggerheartv1.DaggerheartDeleteCountdownRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CountdownId: countdownID,
		Reason:      optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		t.Fatalf("countdown_delete: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeCountdownDeleted)
	if name := optionalString(step.Args, "name", ""); name != "" {
		delete(state.countdowns, name)
	}
}

func runActionRollStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		t.Fatal("action_roll requires actor")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	seed := uint64(optionalInt(step.Args, "seed", 0))
	if seed == 0 {
		seed = chooseActionSeed(t, step.Args, difficulty)
	}

	before := latestSeq(t, ctx, env, state)
	response, err := env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorID(t, state, actorName),
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
		t.Fatalf("action_roll: %v", err)
	}
	state.lastRollSeq = response.GetRollSeq()
	requireEventTypesAfterSeq(t, ctx, env, state, before, event.TypeRollResolved)
}

func runReactionRollStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		t.Fatal("reaction_roll requires actor")
	}
	trait := optionalString(step.Args, "trait", "instinct")
	difficulty := optionalInt(step.Args, "difficulty", 10)
	seed := uint64(optionalInt(step.Args, "seed", 0))
	if seed == 0 {
		seed = chooseActionSeed(t, step.Args, difficulty)
	}

	before := latestSeq(t, ctx, env, state)
	response, err := env.daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorID(t, state, actorName),
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
		t.Fatalf("reaction_roll: %v", err)
	}
	state.lastRollSeq = response.GetRollSeq()
	requireEventTypesAfterSeq(t, ctx, env, state, before, event.TypeRollResolved)
}

func runDamageRollStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		t.Fatal("damage_roll requires actor")
	}
	seed := optionalInt(step.Args, "seed", 0)
	modifier := optionalInt(step.Args, "modifier", optionalInt(step.Args, "damage_modifier", 0))
	critical := optionalBool(step.Args, "critical", false)

	request := &daggerheartv1.SessionDamageRollRequest{
		CampaignId:  state.campaignID,
		SessionId:   state.sessionID,
		CharacterId: actorID(t, state, actorName),
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

	before := latestSeq(t, ctx, env, state)
	response, err := env.daggerheartClient.SessionDamageRoll(ctx, request)
	if err != nil {
		t.Fatalf("damage_roll: %v", err)
	}
	state.lastDamageRollSeq = response.GetRollSeq()
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeDamageRollResolved)
}

func runAdversaryAttackRollStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	actorName := requiredString(step.Args, "actor")
	if actorName == "" {
		t.Fatal("adversary_attack_roll requires actor")
	}
	seed := optionalInt(step.Args, "seed", 0)
	request := &daggerheartv1.SessionAdversaryAttackRollRequest{
		CampaignId:     state.campaignID,
		SessionId:      state.sessionID,
		AdversaryId:    adversaryID(t, state, actorName),
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

	before := latestSeq(t, ctx, env, state)
	response, err := env.daggerheartClient.SessionAdversaryAttackRoll(ctx, request)
	if err != nil {
		t.Fatalf("adversary_attack_roll: %v", err)
	}
	state.lastAdversaryRollSeq = response.GetRollSeq()
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAdversaryRollResolved)
}

func runApplyRollOutcomeStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastRollSeq
	}
	if rollSeq == 0 {
		t.Fatal("apply_roll_outcome requires roll_seq")
	}
	request := &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollSeq,
	}
	if targets := resolveOutcomeTargets(t, state, step.Args); len(targets) > 0 {
		request.Targets = targets
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.ApplyRollOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), request)
	if err != nil {
		t.Fatalf("apply_roll_outcome: %v", err)
	}
	requireAnyEventTypesAfterSeq(t, ctx, env, state, before, event.TypeOutcomeApplied, event.TypeOutcomeRejected)
}

func runApplyAttackOutcomeStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastRollSeq
	}
	if rollSeq == 0 {
		t.Fatal("apply_attack_outcome requires roll_seq")
	}
	targets := resolveAttackTargets(t, state, step.Args)
	if len(targets) == 0 {
		t.Fatal("apply_attack_outcome requires targets")
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.ApplyAttackOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyAttackOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollSeq,
		Targets:   targets,
	})
	if err != nil {
		t.Fatalf("apply_attack_outcome: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAttackResolved)
}

func runApplyAdversaryAttackOutcomeStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastAdversaryRollSeq
	}
	if rollSeq == 0 {
		t.Fatal("apply_adversary_attack_outcome requires roll_seq")
	}
	difficulty := optionalInt(step.Args, "difficulty", 10)
	targets := resolveOutcomeTargets(t, state, step.Args)
	if len(targets) == 0 {
		t.Fatal("apply_adversary_attack_outcome requires targets")
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.ApplyAdversaryAttackOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  state.sessionID,
		RollSeq:    rollSeq,
		Targets:    targets,
		Difficulty: int32(difficulty),
	})
	if err != nil {
		t.Fatalf("apply_adversary_attack_outcome: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeAdversaryAttackResolved)
}

func runApplyReactionOutcomeStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureSession(t, ctx, env, state)
	rollSeq := uint64(optionalInt(step.Args, "roll_seq", 0))
	if rollSeq == 0 {
		rollSeq = state.lastRollSeq
	}
	if rollSeq == 0 {
		t.Fatal("apply_reaction_outcome requires roll_seq")
	}

	before := latestSeq(t, ctx, env, state)
	_, err := env.daggerheartClient.ApplyReactionOutcome(withCampaignID(withSessionID(ctx, state.sessionID), state.campaignID), &daggerheartv1.DaggerheartApplyReactionOutcomeRequest{
		SessionId: state.sessionID,
		RollSeq:   rollSeq,
	})
	if err != nil {
		t.Fatalf("apply_reaction_outcome: %v", err)
	}
	requireEventTypesAfterSeq(t, ctx, env, state, before, daggerheart.EventTypeReactionResolved)
}

func runMitigateDamageStep(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, step Step) {
	ensureCampaign(t, state)
	name := requiredString(step.Args, "target")
	if name == "" {
		t.Fatal("mitigate_damage target is required")
	}
	characterID := actorID(t, state, name)
	armor := optionalInt(step.Args, "armor", 0)
	if armor <= 0 {
		return
	}
	_, err := env.snapshotClient.PatchCharacterState(ctx, &gamev1.PatchCharacterStateRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Armor: int32(armor),
			},
		},
	})
	if err != nil {
		t.Fatalf("patch character armor: %v", err)
	}
}

func ensureCampaign(t *testing.T, state *scenarioState) {
	t.Helper()
	if state.campaignID == "" {
		t.Fatal("campaign is required")
	}
}

func ensureSession(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState) {
	t.Helper()
	if state.campaignID == "" {
		t.Fatal("campaign is required")
	}
	if state.sessionID != "" {
		return
	}
	response, err := env.sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: state.campaignID,
		Name:       "Scenario Session",
	})
	if err != nil {
		t.Fatalf("auto start session: %v", err)
	}
	if response.GetSession() == nil {
		t.Fatal("expected session")
	}
	state.sessionID = response.GetSession().GetId()
}

func latestSeq(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState) uint64 {
	if state.campaignID == "" {
		return 0
	}
	response, err := env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(response.GetEvents()) == 0 {
		return 0
	}
	return response.GetEvents()[0].GetSeq()
}

func requireEventTypesAfterSeq(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, before uint64, types ...event.Type) {
	t.Helper()
	for _, eventType := range types {
		filter := fmt.Sprintf("type = \"%s\"", eventType)
		if state.sessionID != "" && isSessionEvent(string(eventType)) {
			filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
		}
		response, err := env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
			CampaignId: state.campaignID,
			PageSize:   1,
			OrderBy:    "seq desc",
			Filter:     filter,
		})
		if err != nil {
			t.Fatalf("list events for %s: %v", eventType, err)
		}
		if len(response.GetEvents()) == 0 {
			t.Fatalf("expected event %s", eventType)
		}
		if response.GetEvents()[0].GetSeq() <= before {
			t.Fatalf("expected %s after seq %d", eventType, before)
		}
	}
}

func requireAnyEventTypesAfterSeq(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, before uint64, types ...event.Type) {
	t.Helper()
	for _, eventType := range types {
		if hasEventTypeAfterSeq(t, ctx, env, state, before, eventType) {
			return
		}
	}
	labels := make([]string, 0, len(types))
	for _, eventType := range types {
		labels = append(labels, string(eventType))
	}
	t.Fatalf("expected event after seq %d: %s", before, strings.Join(labels, ", "))
}

func hasEventTypeAfterSeq(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, before uint64, eventType event.Type) bool {
	t.Helper()
	filter := fmt.Sprintf("type = \"%s\"", eventType)
	if state.sessionID != "" && isSessionEvent(string(eventType)) {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	response, err := env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		t.Fatalf("list events for %s: %v", eventType, err)
	}
	if len(response.GetEvents()) == 0 {
		return false
	}
	return response.GetEvents()[0].GetSeq() > before
}

func isSessionEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "action.") || strings.HasPrefix(eventType, "session.")
}

func applyDefaultDaggerheartProfile(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, characterID string, args map[string]any) {
	armorValue := optionalInt(args, "armor", 0)
	armorMaxValue := optionalInt(args, "armor_max", 0)
	profile := &daggerheartv1.DaggerheartProfile{
		Level:           int32(optionalInt(args, "level", 1)),
		HpMax:           int32(optionalInt(args, "hp_max", 6)),
		StressMax:       wrapperspb.Int32(int32(optionalInt(args, "stress_max", 6))),
		Evasion:         wrapperspb.Int32(int32(optionalInt(args, "evasion", 10))),
		MajorThreshold:  wrapperspb.Int32(int32(optionalInt(args, "major_threshold", 3))),
		SevereThreshold: wrapperspb.Int32(int32(optionalInt(args, "severe_threshold", 6))),
	}
	if armorMaxValue > 0 {
		profile.ArmorMax = wrapperspb.Int32(int32(armorMaxValue))
	} else if armorValue > 0 {
		profile.ArmorMax = wrapperspb.Int32(int32(armorValue))
	}
	if value := optionalInt(args, "armor_score", 0); value > 0 {
		profile.ArmorScore = wrapperspb.Int32(int32(value))
	}
	applyTraitValue(profile, "agility", args)
	applyTraitValue(profile, "strength", args)
	applyTraitValue(profile, "finesse", args)
	applyTraitValue(profile, "instinct", args)
	applyTraitValue(profile, "presence", args)
	applyTraitValue(profile, "knowledge", args)

	_, err := env.characterClient.PatchCharacterProfile(ctx, &gamev1.PatchCharacterProfileRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemProfilePatch: &gamev1.PatchCharacterProfileRequest_Daggerheart{
			Daggerheart: profile,
		},
	})
	if err != nil {
		t.Fatalf("patch character profile: %v", err)
	}
}

func applyOptionalCharacterState(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, characterID string, args map[string]any) {
	patch := &daggerheartv1.DaggerheartCharacterState{}
	hasPatch := false
	if armor, ok := readInt(args, "armor"); ok {
		patch.Armor = int32(armor)
		hasPatch = true
	}
	if hp, ok := readInt(args, "hp"); ok {
		patch.Hp = int32(hp)
		hasPatch = true
	}
	if stress, ok := readInt(args, "stress"); ok {
		patch.Stress = int32(stress)
		hasPatch = true
	}
	if lifeState := optionalString(args, "life_state", ""); lifeState != "" {
		patch.LifeState = parseLifeState(t, lifeState)
		hasPatch = true
	}
	if !hasPatch {
		return
	}
	_, err := env.snapshotClient.PatchCharacterState(ctx, &gamev1.PatchCharacterStateRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: patch,
		},
	})
	if err != nil {
		t.Fatalf("patch character state: %v", err)
	}
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

func getSnapshot(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState) *daggerheartv1.DaggerheartSnapshot {
	response, err := env.snapshotClient.GetSnapshot(ctx, &gamev1.GetSnapshotRequest{CampaignId: state.campaignID})
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if response.GetSnapshot() == nil || response.GetSnapshot().GetDaggerheart() == nil {
		t.Fatal("expected daggerheart snapshot")
	}
	return response.GetSnapshot().GetDaggerheart()
}

func getCharacterState(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, characterID string) *daggerheartv1.DaggerheartCharacterState {
	response, err := env.characterClient.GetCharacterSheet(ctx, &gamev1.GetCharacterSheetRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		t.Fatalf("get character sheet: %v", err)
	}
	if response.GetState() == nil || response.GetState().GetDaggerheart() == nil {
		t.Fatal("expected daggerheart character state")
	}
	return response.GetState().GetDaggerheart()
}

func getAdversary(t *testing.T, ctx context.Context, env scenarioEnv, state *scenarioState, adversaryID string) *daggerheartv1.DaggerheartAdversary {
	response, err := env.daggerheartClient.GetAdversary(ctx, &daggerheartv1.DaggerheartGetAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryID,
	})
	if err != nil {
		t.Fatalf("get adversary: %v", err)
	}
	if response.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}
	return response.GetAdversary()
}

func chooseActionSeed(t *testing.T, args map[string]any, difficulty int) uint64 {
	hint := strings.ToLower(optionalString(args, "outcome", ""))
	if hint == "" {
		return 42
	}
	for seed := uint64(1); seed < 50000; seed++ {
		result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
			Modifier:   0,
			Difficulty: &difficulty,
			Seed:       int64(seed),
		})
		if err != nil {
			continue
		}
		if matchesOutcomeHint(result, hint) {
			return seed
		}
	}
	t.Fatalf("no seed found for outcome %q", hint)
	return 0
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
	default:
		return false
	}
}

func resolveOutcomeSeed(t *testing.T, args map[string]any, key string, difficulty int, fallback uint64) uint64 {
	hint := optionalString(args, key, "")
	if hint == "" {
		return fallback
	}
	return chooseActionSeed(t, map[string]any{"outcome": hint}, difficulty)
}

func buildActionRollModifiers(args map[string]any, key string) []*daggerheartv1.ActionRollModifier {
	value, ok := args[key]
	if !ok {
		return nil
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	modifiers := make([]*daggerheartv1.ActionRollModifier, 0, len(list))
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
	return modifiers
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

func applyAdversaryDamage(
	t *testing.T,
	ctx context.Context,
	env scenarioEnv,
	state *scenarioState,
	adversaryID string,
	name string,
	damageRoll *daggerheartv1.SessionDamageRollResponse,
	args map[string]any,
) bool {
	t.Helper()
	before := getAdversary(t, ctx, env, state, adversaryID)
	hpBefore := int(before.GetHp())
	armorBefore := int(before.GetArmor())
	majorThreshold := int(before.GetMajorThreshold())
	severeThreshold := int(before.GetSevereThreshold())

	amount := int(damageRoll.GetTotal())
	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: optionalBool(args, "resist_physical", false),
		ResistMagic:    optionalBool(args, "resist_magic", false),
		ImmunePhysical: optionalBool(args, "immune_physical", false),
		ImmuneMagic:    optionalBool(args, "immune_magic", false),
	}
	adjusted := daggerheart.ApplyResistance(amount, damageTypesForArgs(args), resistance)
	if adjusted <= 0 {
		return false
	}
	options := daggerheart.DamageOptions{EnableMassiveDamage: optionalBool(args, "massive_damage", false)}

	result, err := daggerheart.EvaluateDamage(adjusted, majorThreshold, severeThreshold, options)
	if err != nil {
		t.Fatalf("adversary damage: %v", err)
	}

	var app daggerheart.DamageApplication
	if optionalBool(args, "direct", false) {
		app, err = daggerheart.ApplyDamage(hpBefore, adjusted, majorThreshold, severeThreshold, options)
		if err != nil {
			t.Fatalf("adversary damage: %v", err)
		}
	} else {
		app = daggerheart.ApplyDamageWithArmor(hpBefore, armorBefore, result)
	}
	if app.HPAfter >= hpBefore && app.ArmorAfter >= armorBefore {
		t.Fatalf("expected damage to affect hp or armor for %s", name)
	}

	update := &daggerheartv1.DaggerheartUpdateAdversaryRequest{
		CampaignId:  state.campaignID,
		AdversaryId: adversaryID,
	}
	if state.sessionID != "" {
		update.SessionId = wrapperspb.String(state.sessionID)
	}
	if app.HPAfter != hpBefore {
		update.Hp = wrapperspb.Int32(int32(app.HPAfter))
	}
	if app.ArmorAfter != armorBefore {
		update.Armor = wrapperspb.Int32(int32(app.ArmorAfter))
	}
	if update.Hp == nil && update.Armor == nil {
		t.Fatalf("expected adversary damage to change hp or armor for %s", name)
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	if _, err := env.daggerheartClient.UpdateAdversary(ctxWithSession, update); err != nil {
		t.Fatalf("update adversary damage: %v", err)
	}
	after := getAdversary(t, ctx, env, state, adversaryID)
	if after.GetHp() >= before.GetHp() && after.GetArmor() >= before.GetArmor() {
		t.Fatalf("expected damage to affect hp or armor for %s", name)
	}
	return true
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

func damageTypesForArgs(args map[string]any) daggerheart.DamageTypes {
	switch parseDamageType(optionalString(args, "damage_type", "physical")) {
	case daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC:
		return daggerheart.DamageTypes{Magic: true}
	case daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED:
		return daggerheart.DamageTypes{Physical: true, Magic: true}
	default:
		return daggerheart.DamageTypes{Physical: true}
	}
}

func adjustedDamageAmount(args map[string]any, amount int32) int {
	resistance := daggerheart.ResistanceProfile{
		ResistPhysical: optionalBool(args, "resist_physical", false),
		ResistMagic:    optionalBool(args, "resist_magic", false),
		ImmunePhysical: optionalBool(args, "immune_physical", false),
		ImmuneMagic:    optionalBool(args, "immune_magic", false),
	}
	return daggerheart.ApplyResistance(int(amount), damageTypesForArgs(args), resistance)
}

func expectDamageEffect(args map[string]any, roll *daggerheartv1.SessionDamageRollResponse) bool {
	if roll == nil {
		return false
	}
	return adjustedDamageAmount(args, roll.GetTotal()) > 0
}

func parseConditions(t *testing.T, values []string) []daggerheartv1.DaggerheartCondition {
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
			t.Fatalf("unknown condition %q", value)
		}
	}
	return result
}

func parseGameSystem(t *testing.T, value string) commonv1.GameSystem {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DAGGERHEART":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		t.Fatalf("unsupported system %q", value)
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

func parseGmMode(t *testing.T, value string) gamev1.GmMode {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HUMAN":
		return gamev1.GmMode_HUMAN
	case "AI":
		return gamev1.GmMode_AI
	default:
		t.Fatalf("unsupported gm_mode %q", value)
		return gamev1.GmMode_GM_MODE_UNSPECIFIED
	}
}

func parseCharacterKind(t *testing.T, value string) gamev1.CharacterKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC":
		return gamev1.CharacterKind_PC
	case "NPC":
		return gamev1.CharacterKind_NPC
	default:
		t.Fatalf("unsupported character kind %q", value)
		return gamev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

func prefabOptions(name string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "quinn":
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

func actorID(t *testing.T, state *scenarioState, name string) string {
	id, ok := state.actors[name]
	if !ok {
		t.Fatalf("unknown actor %q", name)
	}
	return id
}

func adversaryID(t *testing.T, state *scenarioState, name string) string {
	id, ok := state.adversaries[name]
	if !ok {
		t.Fatalf("unknown adversary %q", name)
	}
	return id
}

func resolveTargetID(t *testing.T, state *scenarioState, name string) (string, bool) {
	if id, ok := state.actors[name]; ok {
		return id, false
	}
	if id, ok := state.adversaries[name]; ok {
		return id, true
	}
	t.Fatalf("unknown target %q", name)
	return "", false
}

func resolveCountdownID(t *testing.T, state *scenarioState, args map[string]any) string {
	if countdownID := optionalString(args, "countdown_id", ""); countdownID != "" {
		return countdownID
	}
	name := optionalString(args, "name", "")
	if name == "" {
		return ""
	}
	countdownID, ok := state.countdowns[name]
	if !ok {
		t.Fatalf("unknown countdown %q", name)
	}
	return countdownID
}

func resolveOutcomeTargets(t *testing.T, state *scenarioState, args map[string]any) []string {
	list := readStringSlice(args, "targets")
	if len(list) == 0 {
		if name := optionalString(args, "target", ""); name != "" {
			list = []string{name}
		}
	}
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		ids = append(ids, actorID(t, state, name))
	}
	return ids
}

func resolveAttackTargets(t *testing.T, state *scenarioState, args map[string]any) []string {
	list := readStringSlice(args, "targets")
	if len(list) == 0 {
		if name := optionalString(args, "target", ""); name != "" {
			list = []string{name}
		}
	}
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		id, _ := resolveTargetID(t, state, name)
		ids = append(ids, id)
	}
	return ids
}

func requireDamageDice(t *testing.T, args map[string]any, context string) {
	value, ok := args["damage_dice"]
	if !ok {
		t.Fatalf("%s requires damage_dice", context)
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("%s damage_dice must be a list", context)
	}
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

func requiredInt(args map[string]any, key string) int {
	value, ok := args[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
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
	hopeDelta   *int
	stressDelta *int
}

func captureExpectedDeltas(
	t *testing.T,
	ctx context.Context,
	env scenarioEnv,
	state *scenarioState,
	args map[string]any,
	fallbackName string,
) (*expectedDeltas, *daggerheartv1.DaggerheartCharacterState) {
	hopeDelta, hopeOk := readInt(args, "expect_hope_delta")
	stressDelta, stressOk := readInt(args, "expect_stress_delta")
	if !hopeOk && !stressOk {
		return nil, nil
	}
	name := optionalString(args, "expect_target", fallbackName)
	if strings.TrimSpace(name) == "" {
		t.Fatal("expect_*_delta requires expect_target or a default character")
	}
	characterID := actorID(t, state, name)
	before := getCharacterState(t, ctx, env, state, characterID)
	spec := &expectedDeltas{name: name, characterID: characterID}
	if hopeOk {
		spec.hopeDelta = &hopeDelta
	}
	if stressOk {
		spec.stressDelta = &stressDelta
	}
	return spec, before
}

func assertExpectedDeltas(
	t *testing.T,
	ctx context.Context,
	env scenarioEnv,
	state *scenarioState,
	spec *expectedDeltas,
	before *daggerheartv1.DaggerheartCharacterState,
) {
	if spec == nil || before == nil {
		return
	}
	after := getCharacterState(t, ctx, env, state, spec.characterID)
	if spec.hopeDelta != nil {
		delta := int(after.GetHope()) - int(before.GetHope())
		if delta != *spec.hopeDelta {
			t.Fatalf("hope delta for %s = %d, want %d", spec.name, delta, *spec.hopeDelta)
		}
	}
	if spec.stressDelta != nil {
		delta := int(after.GetStress()) - int(before.GetStress())
		if delta != *spec.stressDelta {
			t.Fatalf("stress delta for %s = %d, want %d", spec.name, delta, *spec.stressDelta)
		}
	}
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

func assertDamageFlags(
	t *testing.T,
	ctx context.Context,
	env scenarioEnv,
	state *scenarioState,
	before uint64,
	targetID string,
	args map[string]any,
) {
	expect, ok := readDamageFlagExpect(args)
	if !ok {
		return
	}
	filter := fmt.Sprintf("type = \"%s\"", daggerheart.EventTypeDamageApplied)
	if state.sessionID != "" {
		filter = filter + fmt.Sprintf(" AND session_id = \"%s\"", state.sessionID)
	}
	response, err := env.eventClient.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: state.campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		t.Fatalf("list damage events: %v", err)
	}
	var payload daggerheart.DamageAppliedPayload
	for _, evt := range response.GetEvents() {
		if evt.GetSeq() <= before {
			continue
		}
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			t.Fatalf("decode damage payload: %v", err)
		}
		if targetID != "" && payload.CharacterID != targetID {
			continue
		}
		if expect.resistPhysical != nil && payload.ResistPhysical != *expect.resistPhysical {
			t.Fatalf("resist_physical = %v, want %v", payload.ResistPhysical, *expect.resistPhysical)
		}
		if expect.resistMagic != nil && payload.ResistMagic != *expect.resistMagic {
			t.Fatalf("resist_magic = %v, want %v", payload.ResistMagic, *expect.resistMagic)
		}
		if expect.immunePhysical != nil && payload.ImmunePhysical != *expect.immunePhysical {
			t.Fatalf("immune_physical = %v, want %v", payload.ImmunePhysical, *expect.immunePhysical)
		}
		if expect.immuneMagic != nil && payload.ImmuneMagic != *expect.immuneMagic {
			t.Fatalf("immune_magic = %v, want %v", payload.ImmuneMagic, *expect.immuneMagic)
		}
		return
	}
	t.Fatalf("expected damage_applied after seq %d", before)
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
	list, ok := value.([]any)
	if !ok {
		return nil
	}
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

func resolveCharacterList(t *testing.T, state *scenarioState, args map[string]any, key string) []string {
	list := readStringSlice(args, key)
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for _, name := range list {
		ids = append(ids, actorID(t, state, name))
	}
	return ids
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

func parseRestType(t *testing.T, value string) daggerheartv1.DaggerheartRestType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "short":
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT
	case "long":
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG
	default:
		t.Fatalf("unsupported rest type %q", value)
		return daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED
	}
}

func parseCountdownKind(t *testing.T, value string) daggerheartv1.DaggerheartCountdownKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "progress":
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS
	case "consequence":
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE
	default:
		t.Fatalf("unsupported countdown kind %q", value)
		return daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED
	}
}

func parseCountdownDirection(t *testing.T, value string) daggerheartv1.DaggerheartCountdownDirection {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "increase":
		return daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE
	case "decrease":
		return daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE
	default:
		t.Fatalf("unsupported countdown direction %q", value)
		return daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED
	}
}

func parseDowntimeMove(t *testing.T, value string) daggerheartv1.DaggerheartDowntimeMove {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "clear_all_stress":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS
	case "repair_all_armor":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_REPAIR_ALL_ARMOR
	case "prepare":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE
	case "work_on_project":
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_WORK_ON_PROJECT
	default:
		t.Fatalf("unsupported downtime move %q", value)
		return daggerheartv1.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED
	}
}

func parseDeathMove(t *testing.T, value string) daggerheartv1.DaggerheartDeathMove {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "blaze_of_glory":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY
	case "avoid_death":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH
	case "risk_it_all":
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL
	default:
		t.Fatalf("unsupported death move %q", value)
		return daggerheartv1.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED
	}
}

func parseLifeState(t *testing.T, value string) daggerheartv1.DaggerheartLifeState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alive":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE
	case "unconscious":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS
	case "blaze_of_glory":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY
	case "dead":
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD
	default:
		t.Fatalf("unsupported life_state %q", value)
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
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
