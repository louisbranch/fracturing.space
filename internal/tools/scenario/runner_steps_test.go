package scenario

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// --- helpers ---

// testEnv returns a scenarioEnv wired to all fakes with reasonable defaults.
func testEnv() (scenarioEnv, *fakeEventClient, *fakeSessionClient, *fakeDaggerheartClient) {
	eventClient := &fakeEventClient{}
	sessionClient := &fakeSessionClient{
		startSession: func(_ context.Context, req *gamev1.StartSessionRequest, _ ...grpc.CallOption) (*gamev1.StartSessionResponse, error) {
			return &gamev1.StartSessionResponse{Session: &gamev1.Session{Id: "session-1"}}, nil
		},
		endSession: func(_ context.Context, _ *gamev1.EndSessionRequest, _ ...grpc.CallOption) (*gamev1.EndSessionResponse, error) {
			return &gamev1.EndSessionResponse{}, nil
		},
		setSpotlight: func(_ context.Context, _ *gamev1.SetSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.SetSessionSpotlightResponse, error) {
			return &gamev1.SetSessionSpotlightResponse{}, nil
		},
		clearSpotlight: func(_ context.Context, _ *gamev1.ClearSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
			return &gamev1.ClearSessionSpotlightResponse{}, nil
		},
	}
	dhClient := &fakeDaggerheartClient{}
	env := scenarioEnv{
		campaignClient: &fakeCampaignClient{
			create: func(_ context.Context, req *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
				return &gamev1.CreateCampaignResponse{
					Campaign:         &gamev1.Campaign{Id: "campaign-1"},
					OwnerParticipant: &gamev1.Participant{Id: "owner-1"},
				}, nil
			},
		},
		participantClient: &fakeParticipantClient{
			create: func(_ context.Context, req *gamev1.CreateParticipantRequest, _ ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error) {
				return &gamev1.CreateParticipantResponse{
					Participant: &gamev1.Participant{Id: "participant-" + req.GetDisplayName()},
				}, nil
			},
		},
		characterClient: &fakeCharacterClient{
			create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
				return &gamev1.CreateCharacterResponse{
					Character: &gamev1.Character{Id: "char-" + req.GetName()},
				}, nil
			},
			patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
				return &gamev1.PatchCharacterProfileResponse{}, nil
			},
			setDefaultControl: func(_ context.Context, _ *gamev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error) {
				return &gamev1.SetDefaultControlResponse{}, nil
			},
			getSheet: func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
				return &gamev1.GetCharacterSheetResponse{
					State: &gamev1.CharacterState{
						SystemState: &gamev1.CharacterState_Daggerheart{
							Daggerheart: &daggerheartv1.DaggerheartCharacterState{},
						},
					},
				}, nil
			},
		},
		sessionClient: sessionClient,
		eventClient:   eventClient,
		snapshotClient: &fakeSnapshotClient{
			patchState: func(_ context.Context, _ *gamev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
				return &gamev1.PatchCharacterStateResponse{}, nil
			},
		},
		daggerheartClient: dhClient,
	}
	return env, eventClient, sessionClient, dhClient
}

// testState returns a scenarioState with an active campaign and session.
func testState() *scenarioState {
	return &scenarioState{
		campaignID:         "campaign-1",
		ownerParticipantID: "owner-1",
		sessionID:          "session-1",
		actors:             map[string]string{},
		adversaries:        map[string]string{},
		countdowns:         map[string]string{},
		participants:       map[string]string{},
	}
}

func quietRunner(env scenarioEnv) *Runner {
	return &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		logger:     log.New(io.Discard, "", 0),
		env:        env,
	}
}

// --- campaign step tests ---

func TestRunParticipantStepDefaults(t *testing.T) {
	var gotRequest *gamev1.CreateParticipantRequest
	participantClient := &fakeParticipantClient{
		create: func(_ context.Context, req *gamev1.CreateParticipantRequest, _ ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error) {
			gotRequest = req
			return &gamev1.CreateParticipantResponse{
				Participant: &gamev1.Participant{Id: "participant-1"},
			}, nil
		},
	}

	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		env: scenarioEnv{
			participantClient: participantClient,
			eventClient:       &fakeEventClient{},
		},
	}
	state := &scenarioState{campaignID: "campaign-1", participants: map[string]string{}}
	step := Step{Kind: "participant", Args: map[string]any{"name": "Alice"}}

	if err := runner.runParticipantStep(context.Background(), state, step); err != nil {
		t.Fatalf("runParticipantStep: %v", err)
	}
	if gotRequest == nil {
		t.Fatal("expected create participant request")
	}
	if gotRequest.GetRole() != gamev1.ParticipantRole_PLAYER {
		t.Fatalf("role = %s, want PLAYER", gotRequest.GetRole().String())
	}
	if gotRequest.GetController() != gamev1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("controller = %s, want HUMAN", gotRequest.GetController().String())
	}
}

func TestRunCharacterStepControlParticipant(t *testing.T) {
	var controlRequest *gamev1.SetDefaultControlRequest
	characterClient := &fakeCharacterClient{
		create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			return &gamev1.CreateCharacterResponse{
				Character: &gamev1.Character{Id: "character-1"},
			}, nil
		},
		patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			return &gamev1.PatchCharacterProfileResponse{}, nil
		},
		setDefaultControl: func(_ context.Context, req *gamev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error) {
			controlRequest = req
			return &gamev1.SetDefaultControlResponse{}, nil
		},
	}

	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		env: scenarioEnv{
			characterClient: characterClient,
			snapshotClient:  &fakeSnapshotClient{},
			eventClient:     &fakeEventClient{},
		},
	}
	state := &scenarioState{
		campaignID:         "campaign-1",
		ownerParticipantID: "owner-1",
		participants:       map[string]string{"John": "participant-1"},
		actors:             map[string]string{},
	}
	step := Step{Kind: "character", Args: map[string]any{
		"name":        "Frodo",
		"control":     "participant",
		"participant": "John",
	}}

	if err := runner.runCharacterStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCharacterStep: %v", err)
	}
	if controlRequest == nil {
		t.Fatal("expected SetDefaultControl request")
	}
	if got := controlRequest.GetParticipantId(); got == nil || got.GetValue() != "participant-1" {
		t.Fatalf("participant_id = %v, want participant-1", got)
	}
}

func TestRunCharacterStepControlGM(t *testing.T) {
	var controlRequest *gamev1.SetDefaultControlRequest
	characterClient := &fakeCharacterClient{
		create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			return &gamev1.CreateCharacterResponse{
				Character: &gamev1.Character{Id: "character-1"},
			}, nil
		},
		patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			return &gamev1.PatchCharacterProfileResponse{}, nil
		},
		setDefaultControl: func(_ context.Context, req *gamev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*gamev1.SetDefaultControlResponse, error) {
			controlRequest = req
			return &gamev1.SetDefaultControlResponse{}, nil
		},
	}

	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		env: scenarioEnv{
			characterClient: characterClient,
			snapshotClient:  &fakeSnapshotClient{},
			eventClient:     &fakeEventClient{},
		},
	}
	state := &scenarioState{
		campaignID:         "campaign-1",
		ownerParticipantID: "owner-1",
		actors:             map[string]string{},
	}
	step := Step{Kind: "character", Args: map[string]any{
		"name":    "Frodo",
		"control": "gm",
	}}

	if err := runner.runCharacterStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCharacterStep: %v", err)
	}
	if controlRequest == nil {
		t.Fatal("expected SetDefaultControl request")
	}
	if controlRequest.GetParticipantId() != nil {
		t.Fatalf("participant_id = %v, want nil", controlRequest.GetParticipantId())
	}
}

func TestRunCampaignStepVerboseLogging(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := log.New(buffer, "", 0)

	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		logger:     logger,
		verbose:    true,
		env: scenarioEnv{
			campaignClient: &fakeCampaignClient{
				create: func(_ context.Context, req *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
					return &gamev1.CreateCampaignResponse{
						Campaign:         &gamev1.Campaign{Id: "campaign-1"},
						OwnerParticipant: &gamev1.Participant{Id: "participant-1"},
					}, nil
				},
			},
			eventClient: &fakeEventClient{},
		},
	}

	state := &scenarioState{}
	step := Step{Kind: "campaign", Args: map[string]any{"name": "Test", "system": "DAGGERHEART"}}
	if err := runner.runCampaignStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCampaignStep: %v", err)
	}
	if !strings.Contains(buffer.String(), "campaign created") {
		t.Fatalf("expected verbose log to include campaign created")
	}
}

func TestRunCampaignStepRequiresOwnerParticipant(t *testing.T) {
	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		env: scenarioEnv{
			campaignClient: &fakeCampaignClient{
				create: func(_ context.Context, req *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
					return &gamev1.CreateCampaignResponse{
						Campaign: &gamev1.Campaign{Id: "campaign-1"},
					}, nil
				},
			},
			eventClient: &fakeEventClient{},
		},
	}

	state := &scenarioState{}
	step := Step{Kind: "campaign", Args: map[string]any{"name": "Test", "system": "DAGGERHEART"}}
	if err := runner.runCampaignStep(context.Background(), state, step); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseControlInvalid(t *testing.T) {
	_, err := parseControl("invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseParticipantRoleController(t *testing.T) {
	role, err := parseParticipantRole("GM")
	if err != nil {
		t.Fatalf("parseParticipantRole: %v", err)
	}
	if role != gamev1.ParticipantRole_GM {
		t.Fatalf("role = %s, want GM", role.String())
	}
	controller, err := parseController("AI")
	if err != nil {
		t.Fatalf("parseController: %v", err)
	}
	if controller != gamev1.Controller_CONTROLLER_AI {
		t.Fatalf("controller = %s, want AI", controller.String())
	}
}

func TestSetDefaultControlRequestWithoutParticipant(t *testing.T) {
	request := &gamev1.SetDefaultControlRequest{}
	if request.GetParticipantId() != nil {
		t.Fatalf("participant_id = %v, want nil", request.GetParticipantId())
	}
	request.ParticipantId = wrapperspb.String("participant-1")
	if got := request.GetParticipantId().GetValue(); got != "participant-1" {
		t.Fatalf("participant_id = %s, want participant-1", got)
	}
}

// --- runStep dispatch tests ---

func TestRunStepUnknown(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{Kind: "bogus"})
	if err == nil || !strings.Contains(err.Error(), "unknown step kind") {
		t.Fatalf("expected unknown step kind error, got %v", err)
	}
}

func TestRunStepDispatchCampaign(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := &scenarioState{} // no campaign yet
	err := runner.runStep(context.Background(), state, Step{
		Kind: "campaign",
		Args: map[string]any{"name": "Test", "system": "DAGGERHEART"},
	})
	if err != nil {
		t.Fatalf("runStep(campaign): %v", err)
	}
	if state.campaignID != "campaign-1" {
		t.Fatalf("campaignID = %q, want campaign-1", state.campaignID)
	}
}

// --- campaign step edge cases ---

func TestRunCampaignStepDuplicate(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState() // already has campaign
	err := runner.runCampaignStep(context.Background(), state, Step{
		Kind: "campaign",
		Args: map[string]any{"name": "Test", "system": "DAGGERHEART"},
	})
	if err == nil || !strings.Contains(err.Error(), "already created") {
		t.Fatalf("expected duplicate campaign error, got %v", err)
	}
}

func TestRunCampaignStepMissingName(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := &scenarioState{}
	err := runner.runCampaignStep(context.Background(), state, Step{
		Kind: "campaign",
		Args: map[string]any{"system": "DAGGERHEART"},
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name required error, got %v", err)
	}
}

// --- participant step ---

func TestRunParticipantStepMissingCampaign(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := &scenarioState{participants: map[string]string{}}
	err := runner.runParticipantStep(context.Background(), state, Step{
		Kind: "participant",
		Args: map[string]any{"name": "Alice"},
	})
	if err == nil || !strings.Contains(err.Error(), "campaign is required") {
		t.Fatalf("expected campaign required error, got %v", err)
	}
}

func TestRunParticipantStepMissingName(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runParticipantStep(context.Background(), state, Step{
		Kind: "participant",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name required error, got %v", err)
	}
}

func TestRunParticipantStepWithGMRole(t *testing.T) {
	var gotRequest *gamev1.CreateParticipantRequest
	env, _, _, _ := testEnv()
	env.participantClient = &fakeParticipantClient{
		create: func(_ context.Context, req *gamev1.CreateParticipantRequest, _ ...grpc.CallOption) (*gamev1.CreateParticipantResponse, error) {
			gotRequest = req
			return &gamev1.CreateParticipantResponse{
				Participant: &gamev1.Participant{Id: "p-gm"},
			}, nil
		},
	}
	runner := quietRunner(env)
	state := testState()
	err := runner.runParticipantStep(context.Background(), state, Step{
		Kind: "participant",
		Args: map[string]any{"name": "GM Player", "role": "GM", "controller": "AI"},
	})
	if err != nil {
		t.Fatalf("runParticipantStep: %v", err)
	}
	if gotRequest.GetRole() != gamev1.ParticipantRole_GM {
		t.Fatalf("role = %s, want GM", gotRequest.GetRole())
	}
	if gotRequest.GetController() != gamev1.Controller_CONTROLLER_AI {
		t.Fatalf("controller = %s, want AI", gotRequest.GetController())
	}
}

// --- session steps ---

func TestRunStartSessionStep(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.sessionID = "" // no session yet
	err := runner.runStartSessionStep(context.Background(), state, Step{
		Kind: "start_session",
		Args: map[string]any{"name": "Session 1"},
	})
	if err != nil {
		t.Fatalf("runStartSessionStep: %v", err)
	}
	if state.sessionID != "session-1" {
		t.Fatalf("sessionID = %q, want session-1", state.sessionID)
	}
}

func TestRunStartSessionStepRequiresCampaign(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := &scenarioState{}
	err := runner.runStartSessionStep(context.Background(), state, Step{
		Kind: "start_session",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "campaign is required") {
		t.Fatalf("expected campaign required error, got %v", err)
	}
}

func TestRunEndSessionStep(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runEndSessionStep(context.Background(), state)
	if err != nil {
		t.Fatalf("runEndSessionStep: %v", err)
	}
	if state.sessionID != "" {
		t.Fatalf("sessionID = %q, want empty", state.sessionID)
	}
}

func TestRunEndSessionStepRequiresSession(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.sessionID = ""
	err := runner.runEndSessionStep(context.Background(), state)
	if err == nil || !strings.Contains(err.Error(), "session is required") {
		t.Fatalf("expected session required error, got %v", err)
	}
}

// --- character step ---

func TestRunCharacterStepBasic(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runCharacterStep(context.Background(), state, Step{
		Kind: "character",
		Args: map[string]any{"name": "Frodo", "kind": "PC"},
	})
	if err != nil {
		t.Fatalf("runCharacterStep: %v", err)
	}
	if _, ok := state.actors["Frodo"]; !ok {
		t.Fatal("expected Frodo in actors")
	}
}

func TestRunCharacterStepMissingCampaign(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := &scenarioState{actors: map[string]string{}}
	err := runner.runCharacterStep(context.Background(), state, Step{
		Kind: "character",
		Args: map[string]any{"name": "Frodo"},
	})
	if err == nil || !strings.Contains(err.Error(), "campaign is required") {
		t.Fatalf("expected campaign required, got %v", err)
	}
}

func TestRunCharacterStepMissingName(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runCharacterStep(context.Background(), state, Step{
		Kind: "character",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name required, got %v", err)
	}
}

// --- prefab step ---

func TestRunPrefabStep(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runPrefabStep(context.Background(), state, Step{
		Kind: "prefab",
		Args: map[string]any{"name": "Frodo"},
	})
	if err != nil {
		t.Fatalf("runPrefabStep: %v", err)
	}
	if _, ok := state.actors["Frodo"]; !ok {
		t.Fatal("expected Frodo in actors")
	}
}

func TestRunPrefabStepMissingName(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runPrefabStep(context.Background(), state, Step{
		Kind: "prefab",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name required, got %v", err)
	}
}

// --- adversary step ---

func TestRunAdversaryStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.createAdversary = func(_ context.Context, req *daggerheartv1.DaggerheartCreateAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateAdversaryResponse, error) {
		return &daggerheartv1.DaggerheartCreateAdversaryResponse{
			Adversary: &daggerheartv1.DaggerheartAdversary{Id: "adv-1"},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	err := runner.runAdversaryStep(context.Background(), state, Step{
		Kind: "adversary",
		Args: map[string]any{"name": "Goblin"},
	})
	if err != nil {
		t.Fatalf("runAdversaryStep: %v", err)
	}
	if state.adversaries["Goblin"] != "adv-1" {
		t.Fatalf("adversary = %v, want adv-1", state.adversaries["Goblin"])
	}
}

func TestRunAdversaryStepMissingName(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runAdversaryStep(context.Background(), state, Step{
		Kind: "adversary",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name required, got %v", err)
	}
}

// --- gm_fear step ---

func TestRunGMFearStep(t *testing.T) {
	env, _, _, _ := testEnv()
	env.snapshotClient = &fakeSnapshotClient{
		updateSnapshot: func(_ context.Context, _ *gamev1.UpdateSnapshotStateRequest, _ ...grpc.CallOption) (*gamev1.UpdateSnapshotStateResponse, error) {
			return &gamev1.UpdateSnapshotStateResponse{}, nil
		},
		getSnapshot: func(_ context.Context, _ *gamev1.GetSnapshotRequest, _ ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
			return &gamev1.GetSnapshotResponse{
				Snapshot: &gamev1.Snapshot{
					SystemSnapshot: &gamev1.Snapshot_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 3},
					},
				},
			}, nil
		},
	}
	runner := quietRunner(env)
	state := testState()
	err := runner.runGMFearStep(context.Background(), state, Step{
		Kind: "gm_fear",
		Args: map[string]any{"value": 3},
	})
	if err != nil {
		t.Fatalf("runGMFearStep: %v", err)
	}
	if state.gmFear != 3 {
		t.Fatalf("gmFear = %d, want 3", state.gmFear)
	}
}

func TestRunGMFearStepMissingValue(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runGMFearStep(context.Background(), state, Step{
		Kind: "gm_fear",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "value is required") {
		t.Fatalf("expected value required, got %v", err)
	}
}

// --- countdown steps ---

func TestRunCountdownCreateStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.createCountdown = func(_ context.Context, req *daggerheartv1.DaggerheartCreateCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateCountdownResponse, error) {
		return &daggerheartv1.DaggerheartCreateCountdownResponse{
			Countdown: &daggerheartv1.DaggerheartCountdown{CountdownId: "cd-1"},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	err := runner.runCountdownCreateStep(context.Background(), state, Step{
		Kind: "countdown_create",
		Args: map[string]any{"name": "Ritual", "max": 4, "kind": "progress", "direction": "increase"},
	})
	if err != nil {
		t.Fatalf("runCountdownCreateStep: %v", err)
	}
	if state.countdowns["Ritual"] != "cd-1" {
		t.Fatalf("countdown = %v, want cd-1", state.countdowns["Ritual"])
	}
}

func TestRunCountdownUpdateStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.updateCountdown = func(_ context.Context, _ *daggerheartv1.DaggerheartUpdateCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateCountdownResponse, error) {
		return &daggerheartv1.DaggerheartUpdateCountdownResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.countdowns["Ritual"] = "cd-1"
	err := runner.runCountdownUpdateStep(context.Background(), state, Step{
		Kind: "countdown_update",
		Args: map[string]any{"name": "Ritual", "delta": 1},
	})
	if err != nil {
		t.Fatalf("runCountdownUpdateStep: %v", err)
	}
}

func TestRunCountdownDeleteStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.deleteCountdown = func(_ context.Context, _ *daggerheartv1.DaggerheartDeleteCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteCountdownResponse, error) {
		return &daggerheartv1.DaggerheartDeleteCountdownResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.countdowns["Ritual"] = "cd-1"
	err := runner.runCountdownDeleteStep(context.Background(), state, Step{
		Kind: "countdown_delete",
		Args: map[string]any{"name": "Ritual"},
	})
	if err != nil {
		t.Fatalf("runCountdownDeleteStep: %v", err)
	}
	if _, ok := state.countdowns["Ritual"]; ok {
		t.Fatal("expected countdown to be removed from state")
	}
}

// --- mitigate_damage step ---

func TestRunMitigateDamageStep(t *testing.T) {
	env, _, _, _ := testEnv()
	env.snapshotClient = &fakeSnapshotClient{
		patchState: func(_ context.Context, _ *gamev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
			return &gamev1.PatchCharacterStateResponse{}, nil
		},
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runMitigateDamageStep(context.Background(), state, Step{
		Kind: "mitigate_damage",
		Args: map[string]any{"target": "Frodo", "armor": 2},
	})
	if err != nil {
		t.Fatalf("runMitigateDamageStep: %v", err)
	}
}

func TestRunMitigateDamageStepZeroArmor(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	// armor <= 0 should return nil without calling snapshot
	err := runner.runMitigateDamageStep(context.Background(), state, Step{
		Kind: "mitigate_damage",
		Args: map[string]any{"target": "Frodo", "armor": 0},
	})
	if err != nil {
		t.Fatalf("runMitigateDamageStep: %v", err)
	}
}

// --- action_roll step ---

func TestRunActionRollStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 42}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runActionRollStep(context.Background(), state, Step{
		Kind: "action_roll",
		Args: map[string]any{"actor": "Frodo", "trait": "agility", "difficulty": 12, "seed": 1},
	})
	if err != nil {
		t.Fatalf("runActionRollStep: %v", err)
	}
	if state.lastRollSeq != 42 {
		t.Fatalf("lastRollSeq = %d, want 42", state.lastRollSeq)
	}
}

func TestRunActionRollStepMissingActor(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runActionRollStep(context.Background(), state, Step{
		Kind: "action_roll",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "requires actor") {
		t.Fatalf("expected requires actor error, got %v", err)
	}
}

// --- reaction_roll step ---

func TestRunReactionRollStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		if req.GetRollKind() != daggerheartv1.RollKind_ROLL_KIND_REACTION {
			return nil, fmt.Errorf("expected REACTION roll kind, got %v", req.GetRollKind())
		}
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 55}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runReactionRollStep(context.Background(), state, Step{
		Kind: "reaction_roll",
		Args: map[string]any{"actor": "Frodo", "seed": 1},
	})
	if err != nil {
		t.Fatalf("runReactionRollStep: %v", err)
	}
	if state.lastRollSeq != 55 {
		t.Fatalf("lastRollSeq = %d, want 55", state.lastRollSeq)
	}
}

// --- damage_roll step ---

func TestRunDamageRollStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.sessionDamageRoll = func(_ context.Context, _ *daggerheartv1.SessionDamageRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionDamageRollResponse, error) {
		return &daggerheartv1.SessionDamageRollResponse{RollSeq: 77}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runDamageRollStep(context.Background(), state, Step{
		Kind: "damage_roll",
		Args: map[string]any{"actor": "Frodo", "seed": 1, "damage_dice": []any{map[string]any{"sides": 6, "count": 2}}},
	})
	if err != nil {
		t.Fatalf("runDamageRollStep: %v", err)
	}
	if state.lastDamageRollSeq != 77 {
		t.Fatalf("lastDamageRollSeq = %d, want 77", state.lastDamageRollSeq)
	}
}

// --- adversary_attack_roll step ---

func TestRunAdversaryAttackRollStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.sessionAdversaryAttackRoll = func(_ context.Context, _ *daggerheartv1.SessionAdversaryAttackRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackRollResponse, error) {
		return &daggerheartv1.SessionAdversaryAttackRollResponse{RollSeq: 88}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.adversaries["Goblin"] = "adv-1"
	err := runner.runAdversaryAttackRollStep(context.Background(), state, Step{
		Kind: "adversary_attack_roll",
		Args: map[string]any{"actor": "Goblin", "seed": 1},
	})
	if err != nil {
		t.Fatalf("runAdversaryAttackRollStep: %v", err)
	}
	if state.lastAdversaryRollSeq != 88 {
		t.Fatalf("lastAdversaryRollSeq = %d, want 88", state.lastAdversaryRollSeq)
	}
}

// --- apply_roll_outcome step ---

func TestRunApplyRollOutcomeStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyRollOutcome = func(_ context.Context, req *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{},
	})
	if err != nil {
		t.Fatalf("runApplyRollOutcomeStep: %v", err)
	}
}

func TestRunApplyRollOutcomeStepMissingRollSeq(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 0
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "requires roll_seq") {
		t.Fatalf("expected requires roll_seq error, got %v", err)
	}
}

// --- apply_attack_outcome step ---

func TestRunApplyAttackOutcomeStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyAttackOutcome = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyAttackOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAttackOutcomeResponse, error) {
		return &daggerheartv1.DaggerheartApplyAttackOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	state.adversaries["Goblin"] = "adv-1"
	err := runner.runApplyAttackOutcomeStep(context.Background(), state, Step{
		Kind: "apply_attack_outcome",
		Args: map[string]any{"targets": []any{"Goblin"}},
	})
	if err != nil {
		t.Fatalf("runApplyAttackOutcomeStep: %v", err)
	}
}

func TestRunApplyAttackOutcomeStepMissingTargets(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyAttackOutcomeStep(context.Background(), state, Step{
		Kind: "apply_attack_outcome",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "requires targets") {
		t.Fatalf("expected requires targets, got %v", err)
	}
}

// --- apply_adversary_attack_outcome step ---

func TestRunApplyAdversaryAttackOutcomeStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyAdversaryAttackOutcome = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
		return &daggerheartv1.DaggerheartApplyAdversaryAttackOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastAdversaryRollSeq = 88
	state.actors["Frodo"] = "char-frodo"
	err := runner.runApplyAdversaryAttackOutcomeStep(context.Background(), state, Step{
		Kind: "apply_adversary_attack_outcome",
		Args: map[string]any{"targets": []any{"Frodo"}},
	})
	if err != nil {
		t.Fatalf("runApplyAdversaryAttackOutcomeStep: %v", err)
	}
}

func TestRunApplyAdversaryAttackOutcomeStepMissingRollSeq(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runApplyAdversaryAttackOutcomeStep(context.Background(), state, Step{
		Kind: "apply_adversary_attack_outcome",
		Args: map[string]any{"targets": []any{"Frodo"}},
	})
	if err == nil || !strings.Contains(err.Error(), "requires roll_seq") {
		t.Fatalf("expected requires roll_seq, got %v", err)
	}
}

// --- apply_reaction_outcome step ---

func TestRunApplyReactionOutcomeStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyReactionOutcome = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
		return &daggerheartv1.DaggerheartApplyReactionOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyReactionOutcomeStep(context.Background(), state, Step{
		Kind: "apply_reaction_outcome",
		Args: map[string]any{},
	})
	if err != nil {
		t.Fatalf("runApplyReactionOutcomeStep: %v", err)
	}
}

// --- gm_spend_fear step ---

func TestRunGMSpendFearStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 1}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 3
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{"amount": 2, "move": "spotlight"},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
	if state.gmFear != 1 {
		t.Fatalf("gmFear = %d, want 1", state.gmFear)
	}
}

func TestRunGMSpendFearStepZeroAmount(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 3}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 3
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{"amount": 0, "move": "spotlight"},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
}

// --- set_spotlight step ---

func TestRunSetSpotlightStepGM(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runSetSpotlightStep(context.Background(), state, Step{
		Kind: "set_spotlight",
		Args: map[string]any{"type": "gm"},
	})
	if err != nil {
		t.Fatalf("runSetSpotlightStep: %v", err)
	}
}

func TestRunSetSpotlightStepCharacter(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runSetSpotlightStep(context.Background(), state, Step{
		Kind: "set_spotlight",
		Args: map[string]any{"target": "Frodo"},
	})
	if err != nil {
		t.Fatalf("runSetSpotlightStep: %v", err)
	}
}

// --- clear_spotlight step ---

func TestRunClearSpotlightStep(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runClearSpotlightStep(context.Background(), state, Step{
		Kind: "clear_spotlight",
		Args: map[string]any{},
	})
	if err != nil {
		t.Fatalf("runClearSpotlightStep: %v", err)
	}
}

// --- apply_condition step ---

func TestRunApplyConditionStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyConditions = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyConditionsRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyConditionsResponse, error) {
		return &daggerheartv1.DaggerheartApplyConditionsResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runApplyConditionStep(context.Background(), state, Step{
		Kind: "apply_condition",
		Args: map[string]any{"target": "Frodo", "add": []any{"VULNERABLE"}},
	})
	if err != nil {
		t.Fatalf("runApplyConditionStep: %v", err)
	}
}

func TestRunApplyConditionStepMissingTarget(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runApplyConditionStep(context.Background(), state, Step{
		Kind: "apply_condition",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "target is required") {
		t.Fatalf("expected target required, got %v", err)
	}
}

func TestRunApplyConditionStepMissingActions(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runApplyConditionStep(context.Background(), state, Step{
		Kind: "apply_condition",
		Args: map[string]any{"target": "Frodo"},
	})
	if err == nil || !strings.Contains(err.Error(), "requires add, remove, or life_state") {
		t.Fatalf("expected requires add/remove/life_state, got %v", err)
	}
}

// --- rest step ---

func TestRunRestStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyRest = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyRestRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyRestResponse, error) {
		return &daggerheartv1.DaggerheartApplyRestResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runRestStep(context.Background(), state, Step{
		Kind: "rest",
		Args: map[string]any{"type": "short"},
	})
	if err != nil {
		t.Fatalf("runRestStep: %v", err)
	}
}

func TestRunRestStepMissingType(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runRestStep(context.Background(), state, Step{
		Kind: "rest",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "rest type is required") {
		t.Fatalf("expected rest type required, got %v", err)
	}
}

// --- downtime_move step ---

func TestRunDowntimeMoveStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyDowntimeMove = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyDowntimeMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDowntimeMoveResponse, error) {
		return &daggerheartv1.DaggerheartApplyDowntimeMoveResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runDowntimeMoveStep(context.Background(), state, Step{
		Kind: "downtime_move",
		Args: map[string]any{"target": "Frodo", "move": "clear_all_stress"},
	})
	if err != nil {
		t.Fatalf("runDowntimeMoveStep: %v", err)
	}
}

// --- death_move step ---

func TestRunDeathMoveStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.applyDeathMove = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyDeathMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDeathMoveResponse, error) {
		return &daggerheartv1.DaggerheartApplyDeathMoveResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runDeathMoveStep(context.Background(), state, Step{
		Kind: "death_move",
		Args: map[string]any{"target": "Frodo", "move": "avoid_death"},
	})
	if err != nil {
		t.Fatalf("runDeathMoveStep: %v", err)
	}
}

func TestRunDeathMoveStepMissingTarget(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runDeathMoveStep(context.Background(), state, Step{
		Kind: "death_move",
		Args: map[string]any{"move": "avoid_death"},
	})
	if err == nil || !strings.Contains(err.Error(), "target is required") {
		t.Fatalf("expected target required, got %v", err)
	}
}

// --- blaze_of_glory step ---

func TestRunBlazeOfGloryStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.resolveBlazeOfGlory = func(_ context.Context, _ *daggerheartv1.DaggerheartResolveBlazeOfGloryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartResolveBlazeOfGloryResponse, error) {
		return &daggerheartv1.DaggerheartResolveBlazeOfGloryResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runBlazeOfGloryStep(context.Background(), state, Step{
		Kind: "blaze_of_glory",
		Args: map[string]any{"target": "Frodo"},
	})
	if err != nil {
		t.Fatalf("runBlazeOfGloryStep: %v", err)
	}
}

// --- swap_loadout step ---

func TestRunSwapLoadoutStep(t *testing.T) {
	env, _, _, dhClient := testEnv()
	dhClient.swapLoadout = func(_ context.Context, _ *daggerheartv1.DaggerheartSwapLoadoutRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartSwapLoadoutResponse, error) {
		return &daggerheartv1.DaggerheartSwapLoadoutResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runSwapLoadoutStep(context.Background(), state, Step{
		Kind: "swap_loadout",
		Args: map[string]any{"target": "Frodo", "card_id": "card-1"},
	})
	if err != nil {
		t.Fatalf("runSwapLoadoutStep: %v", err)
	}
}

func TestRunSwapLoadoutStepMissingTarget(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	err := runner.runSwapLoadoutStep(context.Background(), state, Step{
		Kind: "swap_loadout",
		Args: map[string]any{"card_id": "card-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "target is required") {
		t.Fatalf("expected target required, got %v", err)
	}
}

func TestRunSwapLoadoutStepMissingCardID(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runSwapLoadoutStep(context.Background(), state, Step{
		Kind: "swap_loadout",
		Args: map[string]any{"target": "Frodo"},
	})
	if err == nil || !strings.Contains(err.Error(), "card_id is required") {
		t.Fatalf("expected card_id required, got %v", err)
	}
}

// --- RunScenario integration ---

func TestRunScenarioNil(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	err := runner.RunScenario(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "scenario is required") {
		t.Fatalf("expected scenario required, got %v", err)
	}
}

func TestRunScenarioStepError(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	err := runner.RunScenario(context.Background(), &Scenario{
		Name: "test",
		Steps: []Step{
			{Kind: "bogus"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown step kind") {
		t.Fatalf("expected step error, got %v", err)
	}
}

func TestRunScenarioCampaignAndSession(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := quietRunner(env)
	runner.verbose = true
	runner.logger = log.New(io.Discard, "", 0)
	runner.timeout = 5 * time.Second
	err := runner.RunScenario(context.Background(), &Scenario{
		Name: "test",
		Steps: []Step{
			{Kind: "campaign", Args: map[string]any{"name": "Test", "system": "DAGGERHEART"}},
			{Kind: "start_session", Args: map[string]any{"name": "Session 1"}},
			{Kind: "end_session", Args: map[string]any{}},
		},
	})
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
}

// --- helper function tests ---

func TestChooseActionSeedDefault(t *testing.T) {
	seed, err := chooseActionSeed(map[string]any{}, 10)
	if err != nil {
		t.Fatalf("chooseActionSeed: %v", err)
	}
	if seed != 42 {
		t.Fatalf("seed = %d, want 42", seed)
	}
}

func TestChooseActionSeedWithOutcome(t *testing.T) {
	// "fear" outcome should find a valid seed
	seed, err := chooseActionSeed(map[string]any{"outcome": "fear"}, 10)
	if err != nil {
		t.Fatalf("chooseActionSeed: %v", err)
	}
	if seed == 0 {
		t.Fatal("expected non-zero seed")
	}
}

func TestMatchesOutcomeHint(t *testing.T) {
	if matchesOutcomeHint(daggerheartdomain.ActionResult{}, "bogus") {
		t.Fatal("expected false for unknown hint")
	}
}

func TestParseFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{"parseGameSystem valid", func() error { _, err := parseGameSystem("DAGGERHEART"); return err }},
		{"parseGameSystem invalid", func() error { _, err := parseGameSystem("BOGUS"); return err }},
		{"parseGmMode valid", func() error { _, err := parseGmMode("HUMAN"); return err }},
		{"parseGmMode invalid", func() error { _, err := parseGmMode("BOGUS"); return err }},
		{"parseCharacterKind valid", func() error { _, err := parseCharacterKind("NPC"); return err }},
		{"parseCharacterKind invalid", func() error { _, err := parseCharacterKind("BOGUS"); return err }},
		{"parseRestType valid", func() error { _, err := parseRestType("long"); return err }},
		{"parseRestType invalid", func() error { _, err := parseRestType("BOGUS"); return err }},
		{"parseCountdownKind valid", func() error { _, err := parseCountdownKind("consequence"); return err }},
		{"parseCountdownKind invalid", func() error { _, err := parseCountdownKind("BOGUS"); return err }},
		{"parseCountdownDirection valid", func() error { _, err := parseCountdownDirection("decrease"); return err }},
		{"parseCountdownDirection invalid", func() error { _, err := parseCountdownDirection("BOGUS"); return err }},
		{"parseDowntimeMove valid", func() error { _, err := parseDowntimeMove("prepare"); return err }},
		{"parseDowntimeMove invalid", func() error { _, err := parseDowntimeMove("BOGUS"); return err }},
		{"parseDeathMove valid", func() error { _, err := parseDeathMove("risk_it_all"); return err }},
		{"parseDeathMove invalid", func() error { _, err := parseDeathMove("BOGUS"); return err }},
		{"parseLifeState valid", func() error { _, err := parseLifeState("dead"); return err }},
		{"parseLifeState invalid", func() error { _, err := parseLifeState("BOGUS"); return err }},
		{"parseConditions valid", func() error { _, err := parseConditions([]string{"HIDDEN", "RESTRAINED"}); return err }},
		{"parseConditions invalid", func() error { _, err := parseConditions([]string{"BOGUS"}); return err }},
		{"parseDamageType", func() error {
			parseDamageType("magic")
			parseDamageType("mixed")
			parseDamageType("physical")
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if strings.HasSuffix(tt.name, "invalid") {
				if err == nil {
					t.Fatal("expected error")
				}
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("actorID case insensitive", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{"Frodo": "char-1"}}
		id, err := actorID(state, "frodo")
		if err != nil {
			t.Fatalf("actorID: %v", err)
		}
		if id != "char-1" {
			t.Fatalf("got %q, want char-1", id)
		}
	})
	t.Run("actorID missing", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{}}
		_, err := actorID(state, "nobody")
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("adversaryID case insensitive", func(t *testing.T) {
		state := &scenarioState{adversaries: map[string]string{"Goblin": "adv-1"}}
		id, err := adversaryID(state, "goblin")
		if err != nil {
			t.Fatalf("adversaryID: %v", err)
		}
		if id != "adv-1" {
			t.Fatalf("got %q, want adv-1", id)
		}
	})
	t.Run("adversaryID missing", func(t *testing.T) {
		state := &scenarioState{adversaries: map[string]string{}}
		_, err := adversaryID(state, "nobody")
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("resolveTargetID actor", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{"Frodo": "char-1"}, adversaries: map[string]string{}}
		id, isAdv, err := resolveTargetID(state, "Frodo")
		if err != nil || isAdv || id != "char-1" {
			t.Fatalf("resolveTargetID = (%q, %v, %v)", id, isAdv, err)
		}
	})
	t.Run("resolveTargetID adversary", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{}, adversaries: map[string]string{"Goblin": "adv-1"}}
		id, isAdv, err := resolveTargetID(state, "Goblin")
		if err != nil || !isAdv || id != "adv-1" {
			t.Fatalf("resolveTargetID = (%q, %v, %v)", id, isAdv, err)
		}
	})
	t.Run("resolveTargetID missing", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{}, adversaries: map[string]string{}}
		_, _, err := resolveTargetID(state, "nobody")
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("allActorIDs", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{"Frodo": "c1", "Sam": "c2"}}
		ids := allActorIDs(state)
		if len(ids) != 2 {
			t.Fatalf("got %d, want 2", len(ids))
		}
	})
	t.Run("allActorIDs empty", func(t *testing.T) {
		state := &scenarioState{actors: map[string]string{}}
		ids := allActorIDs(state)
		if ids != nil {
			t.Fatalf("got %v, want nil", ids)
		}
	})
	t.Run("uniqueNonEmptyStrings", func(t *testing.T) {
		result := uniqueNonEmptyStrings([]string{"a", "b", "a", "", " "})
		if len(result) != 2 {
			t.Fatalf("got %v, want [a b]", result)
		}
	})
	t.Run("uniqueNonEmptyStrings nil", func(t *testing.T) {
		result := uniqueNonEmptyStrings(nil)
		if result != nil {
			t.Fatalf("got %v, want nil", result)
		}
	})
	t.Run("readStringSlice", func(t *testing.T) {
		result := readStringSlice(map[string]any{"k": []any{"a", "b", 42, ""}}, "k")
		if len(result) != 2 {
			t.Fatalf("got %v, want [a b]", result)
		}
	})
	t.Run("readStringSlice missing", func(t *testing.T) {
		result := readStringSlice(map[string]any{}, "k")
		if result != nil {
			t.Fatalf("got %v, want nil", result)
		}
	})
	t.Run("readStringSlice not list", func(t *testing.T) {
		result := readStringSlice(map[string]any{"k": "single"}, "k")
		if result != nil {
			t.Fatalf("got %v, want nil", result)
		}
	})
	t.Run("optionalBool string values", func(t *testing.T) {
		if !optionalBool(map[string]any{"k": "true"}, "k", false) {
			t.Fatal("expected true")
		}
		if optionalBool(map[string]any{"k": "false"}, "k", true) {
			t.Fatal("expected false")
		}
		if optionalBool(map[string]any{"k": "bogus"}, "k", false) {
			t.Fatal("expected fallback")
		}
	})
	t.Run("readBool", func(t *testing.T) {
		v, ok := readBool(map[string]any{"k": "yes"}, "k")
		if !ok || !v {
			t.Fatal("expected true/true")
		}
		v, ok = readBool(map[string]any{"k": "no"}, "k")
		if !ok || v {
			t.Fatal("expected false/true")
		}
		_, ok = readBool(map[string]any{}, "k")
		if ok {
			t.Fatal("expected not ok")
		}
		_, ok = readBool(map[string]any{"k": "bogus"}, "k")
		if ok {
			t.Fatal("expected not ok for bogus")
		}
	})
	t.Run("readInt", func(t *testing.T) {
		v, ok := readInt(map[string]any{"k": float64(42)}, "k")
		if !ok || v != 42 {
			t.Fatalf("expected 42, got %d", v)
		}
		v, ok = readInt(map[string]any{"k": 7}, "k")
		if !ok || v != 7 {
			t.Fatalf("expected 7, got %d", v)
		}
		_, ok = readInt(map[string]any{"k": "str"}, "k")
		if ok {
			t.Fatal("expected not ok")
		}
	})
	t.Run("withUserID empty", func(t *testing.T) {
		ctx := withUserID(context.Background(), "")
		if ctx == nil {
			t.Fatal("expected non-nil ctx")
		}
	})
	t.Run("isSessionEvent", func(t *testing.T) {
		if !isSessionEvent("action.roll") {
			t.Fatal("expected true")
		}
		if !isSessionEvent("session.started") {
			t.Fatal("expected true")
		}
		if isSessionEvent("campaign.created") {
			t.Fatal("expected false")
		}
	})
	t.Run("prefabOptions unknown", func(t *testing.T) {
		opts := prefabOptions("Unknown")
		if opts["kind"] != "PC" {
			t.Fatalf("got %v", opts)
		}
	})
	t.Run("normalizeModifierSource", func(t *testing.T) {
		if normalizeModifierSource("Hope Feature") != "hope_feature" {
			t.Fatal("expected hope_feature")
		}
		if normalizeModifierSource("") != "" {
			t.Fatal("expected empty")
		}
	})
	t.Run("isHopeSpendSource", func(t *testing.T) {
		if !isHopeSpendSource("experience") {
			t.Fatal("expected true for experience")
		}
		if isHopeSpendSource("attack") {
			t.Fatal("expected false for attack")
		}
	})
	t.Run("buildDamageDice default", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{})
		if len(dice) != 1 || dice[0].Sides != 6 {
			t.Fatalf("got %v", dice)
		}
	})
	t.Run("buildDamageDice empty list", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{"damage_dice": []any{}})
		if len(dice) != 1 {
			t.Fatalf("got %d dice, want default", len(dice))
		}
	})
	t.Run("buildDamageDice not list", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{"damage_dice": "wrong"})
		if len(dice) != 1 {
			t.Fatalf("got %d dice, want default", len(dice))
		}
	})
	t.Run("buildDamageDice with non-map", func(t *testing.T) {
		dice := buildDamageDice(map[string]any{"damage_dice": []any{"not-a-map"}})
		if len(dice) != 1 {
			t.Fatalf("got %d dice, want default (non-map entries skipped)", len(dice))
		}
	})
	t.Run("buildActionRollModifiers", func(t *testing.T) {
		mods := buildActionRollModifiers(map[string]any{
			"mods": []any{
				map[string]any{"source": "trait", "value": 2},
				map[string]any{"source": "experience"}, // hope spend, value 0
				map[string]any{"source": "x"},          // missing value, skip
				"not-a-map",                            // skip
			},
		}, "mods")
		if len(mods) != 2 {
			t.Fatalf("got %d modifiers, want 2", len(mods))
		}
	})
	t.Run("buildActionRollModifiers missing", func(t *testing.T) {
		mods := buildActionRollModifiers(map[string]any{}, "mods")
		if mods != nil {
			t.Fatalf("got %v, want nil", mods)
		}
	})
	t.Run("buildActionRollModifiers not list", func(t *testing.T) {
		mods := buildActionRollModifiers(map[string]any{"mods": "wrong"}, "mods")
		if mods != nil {
			t.Fatalf("got %v, want nil", mods)
		}
	})
	t.Run("requireDamageDice", func(t *testing.T) {
		if err := requireDamageDice(map[string]any{}, "test"); err == nil {
			t.Fatal("expected error")
		}
		if err := requireDamageDice(map[string]any{"damage_dice": "wrong"}, "test"); err == nil {
			t.Fatal("expected error")
		}
		if err := requireDamageDice(map[string]any{"damage_dice": []any{map[string]any{}}}, "test"); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("resolveCountdownID by name", func(t *testing.T) {
		state := &scenarioState{countdowns: map[string]string{"Ritual": "cd-1"}}
		id, err := resolveCountdownID(state, map[string]any{"name": "Ritual"})
		if err != nil || id != "cd-1" {
			t.Fatalf("got (%q, %v)", id, err)
		}
	})
	t.Run("resolveCountdownID by id", func(t *testing.T) {
		state := &scenarioState{countdowns: map[string]string{}}
		id, err := resolveCountdownID(state, map[string]any{"countdown_id": "cd-99"})
		if err != nil || id != "cd-99" {
			t.Fatalf("got (%q, %v)", id, err)
		}
	})
	t.Run("resolveCountdownID unknown", func(t *testing.T) {
		state := &scenarioState{countdowns: map[string]string{}}
		_, err := resolveCountdownID(state, map[string]any{"name": "bogus"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("damageTypesForArgs", func(t *testing.T) {
		dt := damageTypesForArgs(map[string]any{"damage_type": "magic"})
		if !dt.Magic || dt.Physical {
			t.Fatalf("expected magic only, got %+v", dt)
		}
		dt = damageTypesForArgs(map[string]any{"damage_type": "mixed"})
		if !dt.Magic || !dt.Physical {
			t.Fatalf("expected both, got %+v", dt)
		}
	})
	t.Run("expectDamageEffect nil roll", func(t *testing.T) {
		if expectDamageEffect(map[string]any{}, nil) {
			t.Fatal("expected false for nil roll")
		}
	})
}

// Test newTestRunner constructor
func TestNewTestRunner(t *testing.T) {
	env, _, _, _ := testEnv()
	runner := newTestRunner(env)
	if runner.auth == nil {
		t.Fatal("expected auth")
	}
	if runner.userID == "" {
		t.Fatal("expected userID")
	}
	if runner.timeout == 0 {
		t.Fatal("expected timeout")
	}
}

// Test NewRunner error paths
func TestNewRunnerEmptyAddr(t *testing.T) {
	_, err := NewRunner(context.Background(), Config{})
	if err == nil || !strings.Contains(err.Error(), "grpc address is required") {
		t.Fatalf("expected grpc address required, got %v", err)
	}
}

// Test Close on nil conn
func TestCloseNilConn(t *testing.T) {
	r := &Runner{}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// Test logf non-verbose
func TestLogfNonVerbose(t *testing.T) {
	r := &Runner{verbose: false}
	r.logf("should not panic %s", "test")
}
