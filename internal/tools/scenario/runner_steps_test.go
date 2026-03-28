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

type testEnvFixture struct {
	env               scenarioEnv
	eventClient       *fakeEventClient
	sessionClient     *fakeSessionClient
	interactionClient *fakeInteractionClient
	daggerheartClient *fakeDaggerheartClient
}

// --- helpers ---

// testEnv returns a scenarioEnv wired to all fakes with reasonable defaults.
func testEnv() testEnvFixture {
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
					Participant: &gamev1.Participant{Id: "participant-" + req.GetName()},
				}, nil
			},
		},
		characterClient: &fakeCharacterClient{
			create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
				return &gamev1.CreateCharacterResponse{
					Character: &gamev1.Character{Id: "char-" + req.GetName()},
				}, nil
			},
			update: func(context.Context, *gamev1.UpdateCharacterRequest, ...grpc.CallOption) (*gamev1.UpdateCharacterResponse, error) {
				return &gamev1.UpdateCharacterResponse{}, nil
			},
			patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
				return &gamev1.PatchCharacterProfileResponse{}, nil
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
		interactionClient: &fakeInteractionClient{
			getState: func(_ context.Context, _ *gamev1.GetInteractionStateRequest, _ ...grpc.CallOption) (*gamev1.GetInteractionStateResponse, error) {
				return &gamev1.GetInteractionStateResponse{
					State: &gamev1.InteractionState{
						PlayerPhase: &gamev1.ScenePlayerPhase{
							Status:               gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM,
							ActingCharacterIds:   []string{},
							ActingParticipantIds: []string{},
							Slots:                []*gamev1.ScenePlayerSlot{},
						},
						Ooc: &gamev1.OOCState{
							Posts:                       []*gamev1.OOCPost{},
							ReadyToResumeParticipantIds: []string{},
						},
					},
				}, nil
			},
		},
		eventClient: eventClient,
		snapshotClient: &fakeSnapshotClient{
			patchState: func(_ context.Context, _ *gamev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*gamev1.PatchCharacterStateResponse, error) {
				return &gamev1.PatchCharacterStateResponse{}, nil
			},
		},
		daggerheartClient: dhClient,
		resolveDaggerheartAdversaryEntryID: func(_ context.Context, name string) (string, error) {
			switch name {
			case "Goblin":
				return "adversary.goblin", nil
			case "Shadow Hound":
				return "adversary.shadow-hound", nil
			default:
				return "adversary." + strings.ToLower(strings.ReplaceAll(name, " ", "-")), nil
			}
		},
	}
	return testEnvFixture{
		env:               env,
		eventClient:       eventClient,
		sessionClient:     sessionClient,
		interactionClient: env.interactionClient.(*fakeInteractionClient),
		daggerheartClient: dhClient,
	}
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
		rollOutcomes:       map[uint64]actionRollResult{},
	}
}

func quietRunner(env scenarioEnv) *Runner {
	return &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		logger:     log.New(io.Discard, "", 0),
		env:        env,
	}
}

func readyDaggerheartSheetResponse() *gamev1.GetCharacterSheetResponse {
	return &gamev1.GetCharacterSheetResponse{
		Profile: &gamev1.CharacterProfile{
			SystemProfile: &gamev1.CharacterProfile_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartProfile{
					ClassId:    "class.guardian",
					SubclassId: "subclass.stalwart",
				},
			},
		},
		State: &gamev1.CharacterState{
			SystemState: &gamev1.CharacterState_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartCharacterState{
					Hp:      6,
					HopeMax: 2,
				},
			},
		},
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

func TestRunCharacterStepOwnerParticipant(t *testing.T) {
	var updateRequest *gamev1.UpdateCharacterRequest
	characterClient := &fakeCharacterClient{
		create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			return &gamev1.CreateCharacterResponse{
				Character: &gamev1.Character{Id: "character-1"},
			}, nil
		},
		update: func(_ context.Context, req *gamev1.UpdateCharacterRequest, _ ...grpc.CallOption) (*gamev1.UpdateCharacterResponse, error) {
			updateRequest = req
			return &gamev1.UpdateCharacterResponse{}, nil
		},
		patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			return &gamev1.PatchCharacterProfileResponse{}, nil
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
		"owner":       "participant",
		"participant": "John",
	}}

	if err := runner.runCharacterStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCharacterStep: %v", err)
	}
	if updateRequest == nil {
		t.Fatal("expected UpdateCharacter request")
	}
	if got := updateRequest.GetOwnerParticipantId(); got == nil || got.GetValue() != "participant-1" {
		t.Fatalf("owner_participant_id = %v, want participant-1", got)
	}
}

func TestRunCharacterStepOwnerUnassigned(t *testing.T) {
	var updateRequest *gamev1.UpdateCharacterRequest
	characterClient := &fakeCharacterClient{
		create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			return &gamev1.CreateCharacterResponse{
				Character: &gamev1.Character{Id: "character-1"},
			}, nil
		},
		update: func(_ context.Context, req *gamev1.UpdateCharacterRequest, _ ...grpc.CallOption) (*gamev1.UpdateCharacterResponse, error) {
			updateRequest = req
			return &gamev1.UpdateCharacterResponse{}, nil
		},
		patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			return &gamev1.PatchCharacterProfileResponse{}, nil
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
		"name":  "Frodo",
		"owner": "unassigned",
	}}

	if err := runner.runCharacterStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCharacterStep: %v", err)
	}
	if updateRequest == nil {
		t.Fatal("expected UpdateCharacter request")
	}
	if got := updateRequest.GetOwnerParticipantId(); got == nil || got.GetValue() != "" {
		t.Fatalf("owner_participant_id = %v, want empty", got)
	}
}

func TestRunCharacterStepSkipSystemReadiness(t *testing.T) {
	characterClient := &fakeCharacterClient{
		create: func(_ context.Context, req *gamev1.CreateCharacterRequest, _ ...grpc.CallOption) (*gamev1.CreateCharacterResponse, error) {
			return &gamev1.CreateCharacterResponse{
				Character: &gamev1.Character{Id: "character-1"},
			}, nil
		},
		patchProfile: func(context.Context, *gamev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*gamev1.PatchCharacterProfileResponse, error) {
			return &gamev1.PatchCharacterProfileResponse{}, nil
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
		"name":                  "Frodo",
		"skip_system_readiness": true,
	}}

	if err := runner.runCharacterStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCharacterStep: %v", err)
	}
	if got := state.actors["Frodo"]; got != "character-1" {
		t.Fatalf("actor id = %q, want character-1", got)
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

func TestRunCampaignStepDefaults(t *testing.T) {
	var gotRequest *gamev1.CreateCampaignRequest
	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		env: scenarioEnv{
			campaignClient: &fakeCampaignClient{
				create: func(_ context.Context, req *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
					gotRequest = req
					return &gamev1.CreateCampaignResponse{
						Campaign:         &gamev1.Campaign{Id: "campaign-1"},
						OwnerParticipant: &gamev1.Participant{Id: "owner-1"},
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
	if gotRequest == nil {
		t.Fatal("expected create campaign request")
	}
	if got := gotRequest.GetGmMode(); got != gamev1.GmMode_HUMAN {
		t.Fatalf("gm mode = %s, want HUMAN", got.String())
	}
	if got := gotRequest.GetIntent(); got != gamev1.CampaignIntent_SANDBOX {
		t.Fatalf("intent = %s, want SANDBOX", got.String())
	}
	if got := gotRequest.GetAccessPolicy(); got != gamev1.CampaignAccessPolicy_PRIVATE {
		t.Fatalf("access policy = %s, want PRIVATE", got.String())
	}
}

func TestRunCampaignStepOverridesDefaults(t *testing.T) {
	var gotRequest *gamev1.CreateCampaignRequest
	runner := &Runner{
		assertions: Assertions{Mode: AssertionStrict},
		env: scenarioEnv{
			campaignClient: &fakeCampaignClient{
				create: func(_ context.Context, req *gamev1.CreateCampaignRequest, _ ...grpc.CallOption) (*gamev1.CreateCampaignResponse, error) {
					gotRequest = req
					return &gamev1.CreateCampaignResponse{
						Campaign:         &gamev1.Campaign{Id: "campaign-1"},
						OwnerParticipant: &gamev1.Participant{Id: "owner-1"},
					}, nil
				},
			},
			eventClient: &fakeEventClient{},
		},
	}

	state := &scenarioState{}
	step := Step{Kind: "campaign", Args: map[string]any{
		"name":          "Test",
		"system":        "DAGGERHEART",
		"intent":        "STANDARD",
		"access_policy": "PUBLIC",
	}}
	if err := runner.runCampaignStep(context.Background(), state, step); err != nil {
		t.Fatalf("runCampaignStep: %v", err)
	}
	if gotRequest == nil {
		t.Fatal("expected create campaign request")
	}
	if got := gotRequest.GetIntent(); got != gamev1.CampaignIntent_STANDARD {
		t.Fatalf("intent = %s, want STANDARD", got.String())
	}
	if got := gotRequest.GetAccessPolicy(); got != gamev1.CampaignAccessPolicy_PUBLIC {
		t.Fatalf("access policy = %s, want PUBLIC", got.String())
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

func TestParseOwnershipInvalid(t *testing.T) {
	_, err := parseOwnership("invalid")
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

func TestUpdateCharacterRequestWithoutOwner(t *testing.T) {
	request := &gamev1.UpdateCharacterRequest{}
	if request.GetOwnerParticipantId() != nil {
		t.Fatalf("owner_participant_id = %v, want nil", request.GetOwnerParticipantId())
	}
	request.OwnerParticipantId = wrapperspb.String("participant-1")
	if got := request.GetOwnerParticipantId().GetValue(); got != "participant-1" {
		t.Fatalf("owner_participant_id = %s, want participant-1", got)
	}
}

// --- runStep dispatch tests ---

func TestRunStepUnknown(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{Kind: "bogus"})
	if err == nil || !strings.Contains(err.Error(), "unknown step kind") {
		t.Fatalf("expected unknown step kind error, got %v", err)
	}
}

func TestRunStepDispatchCampaign(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunCampaignStepMissingSystem(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := &scenarioState{}
	err := runner.runCampaignStep(context.Background(), state, Step{
		Kind: "campaign",
		Args: map[string]any{"name": "Test"},
	})
	if err == nil || !strings.Contains(err.Error(), "campaign system is required") {
		t.Fatalf("expected campaign system required error, got %v", err)
	}
}

// --- participant step ---

func TestRunParticipantStepMissingCampaign(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunStartSessionStepAdoptsImplicitSession(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := testState()
	state.sessionID = "session-implicit"
	state.sessionImplicit = true
	err := runner.runStartSessionStep(context.Background(), state, Step{
		Kind: "start_session",
		Args: map[string]any{"name": "Session 1"},
	})
	if err != nil {
		t.Fatalf("runStartSessionStep: %v", err)
	}
	if state.sessionID != "session-implicit" {
		t.Fatalf("sessionID = %q, want session-implicit", state.sessionID)
	}
	if state.sessionImplicit {
		t.Fatal("expected implicit session to be adopted")
	}
}

func TestRunStartSessionStepRejectsDuplicateExplicitSession(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := testState()
	state.sessionID = "session-existing"
	err := runner.runStartSessionStep(context.Background(), state, Step{
		Kind: "start_session",
		Args: map[string]any{"name": "Session 1"},
	})
	if err == nil || !strings.Contains(err.Error(), "session is already started") {
		t.Fatalf("expected duplicate session error, got %v", err)
	}
}

func TestRunStartSessionStepRequiresCampaign(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.createAdversary = func(_ context.Context, req *daggerheartv1.DaggerheartCreateAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateAdversaryResponse, error) {
		if req.GetAdversaryEntryId() != "adversary.goblin" {
			t.Fatalf("adversary_entry_id = %q", req.GetAdversaryEntryId())
		}
		if req.GetSessionId() != "session-1" || req.GetSceneId() != "scene-1" {
			t.Fatalf("request = %+v", req)
		}
		return &daggerheartv1.DaggerheartCreateAdversaryResponse{
			Adversary: &daggerheartv1.DaggerheartAdversary{Id: "adv-1"},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.activeSceneID = "scene-1"
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunExpectGMFearStep(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	env.snapshotClient = &fakeSnapshotClient{
		getSnapshot: func(_ context.Context, _ *gamev1.GetSnapshotRequest, _ ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
			return &gamev1.GetSnapshotResponse{
				Snapshot: &gamev1.Snapshot{
					SystemSnapshot: &gamev1.Snapshot_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 2},
					},
				},
			}, nil
		},
	}
	runner := quietRunner(env)
	state := testState()
	if err := runner.runExpectGMFearStep(context.Background(), state, Step{
		Kind: "expect_gm_fear",
		Args: map[string]any{"value": 2},
	}); err != nil {
		t.Fatalf("runExpectGMFearStep: %v", err)
	}
	if state.gmFear != 2 {
		t.Fatalf("gmFear = %d, want 2", state.gmFear)
	}
}

func TestRunExpectGMFearStepMismatch(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	env.snapshotClient = &fakeSnapshotClient{
		getSnapshot: func(_ context.Context, _ *gamev1.GetSnapshotRequest, _ ...grpc.CallOption) (*gamev1.GetSnapshotResponse, error) {
			return &gamev1.GetSnapshotResponse{
				Snapshot: &gamev1.Snapshot{
					SystemSnapshot: &gamev1.Snapshot_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 1},
					},
				},
			}, nil
		},
	}
	runner := quietRunner(env)
	state := testState()
	err := runner.runExpectGMFearStep(context.Background(), state, Step{
		Kind: "expect_gm_fear",
		Args: map[string]any{"value": 2},
	})
	if err == nil || !strings.Contains(err.Error(), "expect_gm_fear") {
		t.Fatalf("expected gm fear assertion, got %v", err)
	}
}

func TestRunCreationWorkflowStepAppliesWorkflowAndAssertsProfile(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	var applyReq *gamev1.ApplyCharacterCreationWorkflowRequest
	env.characterClient = &fakeCharacterClient{
		applyWorkflow: func(_ context.Context, req *gamev1.ApplyCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*gamev1.ApplyCharacterCreationWorkflowResponse, error) {
			applyReq = req
			return &gamev1.ApplyCharacterCreationWorkflowResponse{}, nil
		},
		getSheet: func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				Profile: &gamev1.CharacterProfile{
					SystemProfile: &gamev1.CharacterProfile_Daggerheart{
						Daggerheart: &daggerheartv1.DaggerheartProfile{
							ClassId:    "class.ranger",
							SubclassId: "subclass.beastbound",
							Heritage: &daggerheartv1.DaggerheartHeritageSelection{
								AncestryLabel:           "Stoneleaf",
								FirstFeatureAncestryId:  "heritage.dwarf",
								SecondFeatureAncestryId: "heritage.elf",
								CommunityId:             "heritage.highborne",
							},
							CompanionSheet: &daggerheartv1.DaggerheartCompanionSheet{
								Name:       "Rocket",
								AnimalKind: "Raccoon",
								DamageType: "physical",
							},
						},
					},
				},
			}, nil
		},
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Mira"] = "char-mira"
	err := runner.runCreationWorkflowStep(context.Background(), state, Step{
		Kind: "creation_workflow",
		Args: map[string]any{
			"target":      "Mira",
			"class_id":    "class.ranger",
			"subclass_id": "subclass.beastbound",
			"heritage": map[string]any{
				"first_feature_ancestry_id":  "heritage.dwarf",
				"second_feature_ancestry_id": "heritage.elf",
				"ancestry_label":             "Stoneleaf",
				"community_id":               "heritage.highborne",
			},
			"companion": map[string]any{
				"animal_kind":        "Raccoon",
				"name":               "Rocket",
				"experience_ids":     []any{"companion-experience.scout", "companion-experience.vigilant"},
				"attack_description": "Short range concussion blast",
				"damage_type":        "physical",
			},
			"expect_class_id":                   "class.ranger",
			"expect_subclass_id":                "subclass.beastbound",
			"expect_heritage_label":             "Stoneleaf",
			"expect_first_feature_ancestry_id":  "heritage.dwarf",
			"expect_second_feature_ancestry_id": "heritage.elf",
			"expect_community_id":               "heritage.highborne",
			"expect_companion_present":          true,
			"expect_companion_name":             "Rocket",
			"expect_companion_animal_kind":      "Raccoon",
			"expect_companion_damage_type":      "physical",
		},
	})
	if err != nil {
		t.Fatalf("runCreationWorkflowStep: %v", err)
	}
	if applyReq == nil {
		t.Fatal("expected ApplyCharacterCreationWorkflow request")
	}
	got := applyReq.GetDaggerheart()
	if got == nil || got.GetClassSubclassInput() == nil || got.GetClassSubclassInput().GetCompanion() == nil {
		t.Fatalf("workflow input = %+v, want companion-backed class/subclass input", got)
	}
	if got.GetHeritageInput().GetHeritage().GetSecondFeatureAncestryId() != "heritage.elf" {
		t.Fatalf("second ancestry = %q, want heritage.elf", got.GetHeritageInput().GetHeritage().GetSecondFeatureAncestryId())
	}
}

func TestRunCreationWorkflowStepMissingTarget(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := testState()
	err := runner.runCreationWorkflowStep(context.Background(), state, Step{
		Kind: "creation_workflow",
		Args: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "target is required") {
		t.Fatalf("expected target required, got %v", err)
	}
}

// --- countdown steps ---

func TestRunCountdownCreateStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.createCountdown = func(_ context.Context, req *daggerheartv1.DaggerheartCreateSceneCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateSceneCountdownResponse, error) {
		return &daggerheartv1.DaggerheartCreateSceneCountdownResponse{
			Countdown: &daggerheartv1.DaggerheartSceneCountdown{
				CountdownId:       "cd-1",
				Name:              req.GetName(),
				Tone:              req.GetTone(),
				AdvancementPolicy: req.GetAdvancementPolicy(),
				StartingValue:     4,
				RemainingValue:    4,
				LoopBehavior:      req.GetLoopBehavior(),
				Status:            daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
				LinkedCountdownId: req.GetLinkedCountdownId(),
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.activeSceneID = "scene-1"
	state.countdowns["Consequence Clock"] = "cd-linked"
	err := runner.runCountdownCreateStep(context.Background(), state, Step{
		Kind: "scene_countdown_create",
		Args: map[string]any{
			"name":                       "Ritual",
			"tone":                       "progress",
			"advancement_policy":         "manual",
			"fixed_starting_value":       4,
			"loop_behavior":              "none",
			"linked_countdown_id":        "Consequence Clock",
			"expect_tone":                "progress",
			"expect_advancement_policy":  "manual",
			"expect_starting_value":      4,
			"expect_remaining_value":     4,
			"expect_loop_behavior":       "none",
			"expect_status":              "active",
			"expect_linked_countdown_id": "Consequence Clock",
		},
	})
	if err != nil {
		t.Fatalf("runCountdownCreateStep: %v", err)
	}
	if state.countdowns["Ritual"] != "cd-1" {
		t.Fatalf("countdown = %v, want cd-1", state.countdowns["Ritual"])
	}
}

func TestRunCountdownUpdateStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.advanceCountdown = func(_ context.Context, _ *daggerheartv1.DaggerheartAdvanceSceneCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartAdvanceSceneCountdownResponse, error) {
		return &daggerheartv1.DaggerheartAdvanceSceneCountdownResponse{
			Countdown: &daggerheartv1.DaggerheartSceneCountdown{
				CountdownId:    "cd-1",
				Name:           "Ritual",
				StartingValue:  4,
				RemainingValue: 3,
				Status:         daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
			},
			Advance: &daggerheartv1.DaggerheartCountdownAdvance{
				RemainingBefore: 4,
				RemainingAfter:  3,
				AdvancedBy:      1,
				Triggered:       false,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.activeSceneID = "scene-1"
	state.countdowns["Ritual"] = "cd-1"
	err := runner.runCountdownUpdateStep(context.Background(), state, Step{
		Kind: "scene_countdown_update",
		Args: map[string]any{
			"name":                    "Ritual",
			"amount":                  1,
			"expect_remaining_value":  3,
			"expect_before_remaining": 4,
			"expect_after_remaining":  3,
			"expect_advanced_by":      1,
			"expect_triggered":        false,
		},
	})
	if err != nil {
		t.Fatalf("runCountdownUpdateStep: %v", err)
	}
}

func TestRunCountdownResolveTriggerStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.resolveCountdownTrigger = func(_ context.Context, _ *daggerheartv1.DaggerheartResolveSceneCountdownTriggerRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartResolveSceneCountdownTriggerResponse, error) {
		return &daggerheartv1.DaggerheartResolveSceneCountdownTriggerResponse{
			Countdown: &daggerheartv1.DaggerheartSceneCountdown{
				CountdownId:    "cd-1",
				Name:           "Ritual",
				StartingValue:  4,
				RemainingValue: 4,
				LoopBehavior:   daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET,
				Status:         daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.activeSceneID = "scene-1"
	state.countdowns["Ritual"] = "cd-1"
	err := runner.runCountdownResolveTriggerStep(context.Background(), state, Step{
		Kind: "scene_countdown_resolve_trigger",
		Args: map[string]any{"name": "Ritual", "expect_remaining_value": 4, "expect_status": "active", "expect_loop_behavior": "reset"},
	})
	if err != nil {
		t.Fatalf("runCountdownResolveTriggerStep: %v", err)
	}
}

func TestRunCountdownDeleteStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.deleteCountdown = func(_ context.Context, _ *daggerheartv1.DaggerheartDeleteSceneCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteSceneCountdownResponse, error) {
		return &daggerheartv1.DaggerheartDeleteSceneCountdownResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.activeSceneID = "scene-1"
	state.countdowns["Ritual"] = "cd-1"
	err := runner.runCountdownDeleteStep(context.Background(), state, Step{
		Kind: "scene_countdown_delete",
		Args: map[string]any{"name": "Ritual"},
	})
	if err != nil {
		t.Fatalf("runCountdownDeleteStep: %v", err)
	}
	if _, ok := state.countdowns["Ritual"]; ok {
		t.Fatal("expected countdown to be removed from state")
	}
}

func TestRunCampaignCountdownCreateStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.createCampaignCountdown = func(_ context.Context, req *daggerheartv1.DaggerheartCreateCampaignCountdownRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateCampaignCountdownResponse, error) {
		return &daggerheartv1.DaggerheartCreateCampaignCountdownResponse{
			Countdown: &daggerheartv1.DaggerheartCampaignCountdown{
				CountdownId:       "camp-cd-1",
				Name:              req.GetName(),
				Tone:              req.GetTone(),
				AdvancementPolicy: req.GetAdvancementPolicy(),
				StartingValue:     6,
				RemainingValue:    6,
				LoopBehavior:      req.GetLoopBehavior(),
				Status:            daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	err := runner.runCampaignCountdownCreateStep(context.Background(), state, Step{
		Kind: "campaign_countdown_create",
		Args: map[string]any{
			"name":                      "Long Project",
			"tone":                      "progress",
			"advancement_policy":        "long_rest",
			"fixed_starting_value":      6,
			"loop_behavior":             "none",
			"expect_advancement_policy": "long_rest",
			"expect_starting_value":     6,
			"expect_remaining_value":    6,
		},
	})
	if err != nil {
		t.Fatalf("runCampaignCountdownCreateStep: %v", err)
	}
	if state.countdowns["Long Project"] != "camp-cd-1" {
		t.Fatalf("countdown = %v, want camp-cd-1", state.countdowns["Long Project"])
	}
}

func TestRunCampaignCountdownResolveTriggerStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.resolveCampaignCountdownTrigger = func(_ context.Context, _ *daggerheartv1.DaggerheartResolveCampaignCountdownTriggerRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartResolveCampaignCountdownTriggerResponse, error) {
		return &daggerheartv1.DaggerheartResolveCampaignCountdownTriggerResponse{
			Countdown: &daggerheartv1.DaggerheartCampaignCountdown{
				CountdownId:    "camp-cd-1",
				Name:           "Long Project",
				StartingValue:  3,
				RemainingValue: 3,
				Status:         daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.countdowns["Long Project"] = "camp-cd-1"
	err := runner.runCampaignCountdownResolveTriggerStep(context.Background(), state, Step{
		Kind: "campaign_countdown_resolve_trigger",
		Args: map[string]any{"name": "Long Project", "expect_remaining_value": 3, "expect_status": "active"},
	})
	if err != nil {
		t.Fatalf("runCampaignCountdownResolveTriggerStep: %v", err)
	}
}

// --- mitigate_damage step ---

func TestRunMitigateDamageStep(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.SessionActionRollRequest
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		request = req
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 42, Total: 15}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runActionRollStep(context.Background(), state, Step{
		Kind: "action_roll",
		Args: map[string]any{
			"actor":        "Frodo",
			"trait":        "agility",
			"difficulty":   12,
			"seed":         1,
			"advantage":    2,
			"disadvantage": 1,
			"context":      "move_silently",
			"expect_total": 15,
		},
	})
	if err != nil {
		t.Fatalf("runActionRollStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected request")
	}
	if request.GetAdvantage() != 2 || request.GetDisadvantage() != 1 {
		t.Fatalf("advantage/disadvantage mismatch: %d/%d", request.GetAdvantage(), request.GetDisadvantage())
	}
	if request.GetContext() != daggerheartv1.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY {
		t.Fatalf("context = %v, want MOVE_SILENTLY", request.GetContext())
	}
	if state.lastRollSeq != 42 {
		t.Fatalf("lastRollSeq = %d, want 42", state.lastRollSeq)
	}
}

func TestRunActionRollStepForwardsFlatModifier(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.SessionActionRollRequest
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		request = req
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 42}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runActionRollStep(context.Background(), state, Step{
		Kind: "action_roll",
		Args: map[string]any{
			"actor":      "Frodo",
			"trait":      "agility",
			"seed":       1,
			"modifier":   3,
			"difficulty": 12,
		},
	})
	if err != nil {
		t.Fatalf("runActionRollStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected request")
	}
	mods := request.GetModifiers()
	if len(mods) != 1 {
		t.Fatalf("len(modifiers) = %d, want 1", len(mods))
	}
	if mods[0].GetSource() != "modifier" || mods[0].GetValue() != 3 {
		t.Fatalf("unexpected modifier: source=%s value=%d", mods[0].GetSource(), mods[0].GetValue())
	}
}

func TestRunActionRollStepMissingActor(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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

func TestRunActionRollStepRejectsUnknownContext(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runActionRollStep(context.Background(), state, Step{
		Kind: "action_roll",
		Args: map[string]any{
			"actor":   "Frodo",
			"context": "unknown",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported action roll context") {
		t.Fatalf("expected unsupported context error, got %v", err)
	}
}

// --- reaction_roll step ---

func TestRunReactionRollStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.SessionActionRollRequest
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		if req.GetRollKind() != daggerheartv1.RollKind_ROLL_KIND_REACTION {
			return nil, fmt.Errorf("expected REACTION roll kind, got %v", req.GetRollKind())
		}
		request = req
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 55}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runReactionRollStep(context.Background(), state, Step{
		Kind: "reaction_roll",
		Args: map[string]any{
			"actor":        "Frodo",
			"seed":         1,
			"advantage":    1,
			"disadvantage": 2,
		},
	})
	if err != nil {
		t.Fatalf("runReactionRollStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected request")
	}
	if request.GetAdvantage() != 1 || request.GetDisadvantage() != 2 {
		t.Fatalf("advantage/disadvantage mismatch: %d/%d", request.GetAdvantage(), request.GetDisadvantage())
	}
	if state.lastRollSeq != 55 {
		t.Fatalf("lastRollSeq = %d, want 55", state.lastRollSeq)
	}
}

func TestRunReactionRollStepForwardsFlatModifier(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.SessionActionRollRequest
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		request = req
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 55}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runReactionRollStep(context.Background(), state, Step{
		Kind: "reaction_roll",
		Args: map[string]any{
			"actor":    "Frodo",
			"seed":     1,
			"modifier": 4,
		},
	})
	if err != nil {
		t.Fatalf("runReactionRollStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected request")
	}
	mods := request.GetModifiers()
	if len(mods) != 1 {
		t.Fatalf("len(modifiers) = %d, want 1", len(mods))
	}
	if mods[0].GetSource() != "modifier" || mods[0].GetValue() != 4 {
		t.Fatalf("unexpected modifier: source=%s value=%d", mods[0].GetSource(), mods[0].GetValue())
	}
}

func TestRunReactionRollStepAdversaryActorUsesAdversaryRoll(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.SessionAdversaryAttackRollRequest
	dhClient.sessionAdversaryAttackRoll = func(_ context.Context, req *daggerheartv1.SessionAdversaryAttackRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackRollResponse, error) {
		request = req
		return &daggerheartv1.SessionAdversaryAttackRollResponse{RollSeq: 144}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Golum"] = "adv-golum"
	err := runner.runReactionRollStep(context.Background(), state, Step{
		Kind: "reaction_roll",
		Args: map[string]any{
			"actor":        "Golum",
			"seed":         5,
			"modifier":     3,
			"advantage":    1,
			"disadvantage": 0,
		},
	})
	if err != nil {
		t.Fatalf("runReactionRollStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected adversary roll request")
	}
	if request.GetAdversaryId() != "adv-golum" {
		t.Fatalf("adversary_id = %q, want adv-golum", request.GetAdversaryId())
	}
	if len(request.GetModifiers()) != 1 || request.GetModifiers()[0].GetValue() != 3 {
		t.Fatalf("modifiers = %+v, want one modifier with value 3", request.GetModifiers())
	}
	if request.GetRng() == nil || request.GetRng().GetSeed() != 5 {
		t.Fatalf("expected replay seed 5, got %+v", request.GetRng())
	}
	if state.lastAdversaryRollSeq != 144 {
		t.Fatalf("lastAdversaryRollSeq = %d, want 144", state.lastAdversaryRollSeq)
	}
}

func TestRunReactionStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.SessionReactionFlowRequest
	dhClient.sessionReactionFlow = func(_ context.Context, req *daggerheartv1.SessionReactionFlowRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionReactionFlowResponse, error) {
		request = req
		return &daggerheartv1.SessionReactionFlowResponse{
			ActionRoll: &daggerheartv1.SessionActionRollResponse{RollSeq: 99},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runReactionStep(context.Background(), state, Step{
		Kind: "reaction",
		Args: map[string]any{
			"actor":        "Frodo",
			"trait":        "agility",
			"difficulty":   9,
			"seed":         4,
			"advantage":    1,
			"disadvantage": 2,
		},
	})
	if err != nil {
		t.Fatalf("runReactionStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected request")
	}
	if request.GetAdvantage() != 1 || request.GetDisadvantage() != 2 {
		t.Fatalf("advantage/disadvantage mismatch: %d/%d", request.GetAdvantage(), request.GetDisadvantage())
	}
	if request.GetReactionRng() == nil {
		t.Fatal("expected replay rng")
	}
	if state.lastRollSeq != 99 {
		t.Fatalf("lastRollSeq = %d, want 99", state.lastRollSeq)
	}
}

// --- adversary_attack step ---

func TestRunAdversaryAttackStepForwardsFeatureTargetsAndContributors(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	callOrder := make([]string, 0, 1)

	var attackReq *daggerheartv1.SessionAdversaryAttackFlowRequest
	dhClient.sessionAdversaryAttackFlow = func(_ context.Context, req *daggerheartv1.SessionAdversaryAttackFlowRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackFlowResponse, error) {
		callOrder = append(callOrder, "attack")
		attackReq = req
		return &daggerheartv1.SessionAdversaryAttackFlowResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Skulk"] = "adv-skulk"
	state.adversaries["Packmate"] = "adv-packmate"
	state.actors["Frodo"] = "char-frodo"
	state.actors["Sam"] = "char-sam"

	err := runner.runAdversaryAttackStep(context.Background(), state, Step{
		Kind: "adversary_attack",
		Args: map[string]any{
			"actor":        "Skulk",
			"targets":      []any{"Frodo", "Sam"},
			"difficulty":   10,
			"damage_type":  "physical",
			"feature_id":   "group_attack",
			"contributors": []any{"Packmate"},
		},
	})
	if err != nil {
		t.Fatalf("runAdversaryAttackStep: %v", err)
	}
	if attackReq == nil {
		t.Fatal("expected SessionAdversaryAttackFlow request")
	}
	if got := attackReq.GetFeatureId(); got != "group_attack" {
		t.Fatalf("feature_id = %q, want group_attack", got)
	}
	if got := attackReq.GetTargetId(); got != "char-frodo" {
		t.Fatalf("target_id = %q, want char-frodo", got)
	}
	if got := attackReq.GetTargetIds(); len(got) != 2 || got[0] != "char-frodo" || got[1] != "char-sam" {
		t.Fatalf("target_ids = %v, want [char-frodo char-sam]", got)
	}
	if got := attackReq.GetContributorAdversaryIds(); len(got) != 1 || got[0] != "adv-packmate" {
		t.Fatalf("contributor_adversary_ids = %v, want [adv-packmate]", got)
	}
	if attackReq.GetTargetArmorReaction() != nil {
		t.Fatalf("target armor reaction = %v, want nil", attackReq.GetTargetArmorReaction())
	}
	if got := strings.Join(callOrder, ","); got != "attack" {
		t.Fatalf("call order = %q, want attack", got)
	}
}

func TestRunAdversaryAttackStepForwardsTimeslowingReaction(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var attackReq *daggerheartv1.SessionAdversaryAttackFlowRequest

	dhClient.sessionAdversaryAttackFlow = func(_ context.Context, req *daggerheartv1.SessionAdversaryAttackFlowRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackFlowResponse, error) {
		attackReq = req
		return &daggerheartv1.SessionAdversaryAttackFlowResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Ranger"] = "adv-ranger"
	state.actors["Frodo"] = "char-frodo"

	err := runner.runAdversaryAttackStep(context.Background(), state, Step{
		Kind: "adversary_attack",
		Args: map[string]any{
			"actor":               "Ranger",
			"target":              "Frodo",
			"difficulty":          10,
			"damage_type":         "physical",
			"armor_reaction":      "timeslowing",
			"armor_reaction_seed": 17,
		},
	})
	if err != nil {
		t.Fatalf("runAdversaryAttackStep: %v", err)
	}
	if attackReq == nil {
		t.Fatal("expected SessionAdversaryAttackFlow request")
	}
	if attackReq.GetTargetArmorReaction() == nil || attackReq.GetTargetArmorReaction().GetTimeslowing() == nil {
		t.Fatalf("target armor reaction = %v, want timeslowing", attackReq.GetTargetArmorReaction())
	}
	if got := attackReq.GetTargetArmorReaction().GetTimeslowing().GetRng().GetSeed(); got != 17 {
		t.Fatalf("armor reaction seed = %d, want 17", got)
	}
}

func TestRunAdversaryAttackStepRequiresTargetOrTargets(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	attackFlowCalls := 0
	dhClient.sessionAdversaryAttackFlow = func(_ context.Context, _ *daggerheartv1.SessionAdversaryAttackFlowRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionAdversaryAttackFlowResponse, error) {
		attackFlowCalls++
		return &daggerheartv1.SessionAdversaryAttackFlowResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Ranger"] = "adv-ranger"
	state.actors["Frodo"] = "char-frodo"

	err := runner.runAdversaryAttackStep(context.Background(), state, Step{
		Kind: "adversary_attack",
		Args: map[string]any{
			"actor":       "Ranger",
			"difficulty":  10,
			"damage_type": "physical",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing target data")
	}
	if attackFlowCalls != 0 {
		t.Fatalf("attack flow calls = %d, want 0", attackFlowCalls)
	}
}

func TestRunAdversaryFeatureStepCallsApplyAdversaryFeature(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var featureReq *daggerheartv1.DaggerheartApplyAdversaryFeatureRequest
	dhClient.applyAdversaryFeature = func(_ context.Context, req *daggerheartv1.DaggerheartApplyAdversaryFeatureRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryFeatureResponse, error) {
		featureReq = req
		return &daggerheartv1.DaggerheartApplyAdversaryFeatureResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Saruman"] = "adv-saruman"
	state.actors["Frodo"] = "char-frodo"

	err := runner.runAdversaryFeatureStep(context.Background(), state, Step{
		Kind: "adversary_feature",
		Args: map[string]any{
			"actor":   "Saruman",
			"target":  "Frodo",
			"feature": "warding_sphere",
		},
	})
	if err != nil {
		t.Fatalf("runAdversaryFeatureStep: %v", err)
	}
	if featureReq == nil {
		t.Fatal("expected apply adversary feature request")
	}
	if got := featureReq.GetAdversaryId(); got != "adv-saruman" {
		t.Fatalf("adversary_id = %q, want adv-saruman", got)
	}
	if got := featureReq.GetFeatureId(); got != "warding_sphere" {
		t.Fatalf("feature_id = %q, want warding_sphere", got)
	}
	if got := featureReq.GetTargetCharacterId(); got != "char-frodo" {
		t.Fatalf("target_character_id = %q, want char-frodo", got)
	}
}

func TestRunAdversaryReactionStepFeatureDelegatesToAdversaryFeature(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var featureReq *daggerheartv1.DaggerheartApplyAdversaryFeatureRequest
	dhClient.applyAdversaryFeature = func(_ context.Context, req *daggerheartv1.DaggerheartApplyAdversaryFeatureRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryFeatureResponse, error) {
		featureReq = req
		return &daggerheartv1.DaggerheartApplyAdversaryFeatureResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Saruman"] = "adv-saruman"
	state.actors["Frodo"] = "char-frodo"

	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "adversary_reaction",
		Args: map[string]any{
			"actor":   "Saruman",
			"target":  "Frodo",
			"feature": "warding_sphere",
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if featureReq == nil {
		t.Fatal("expected apply adversary feature request")
	}
	if got := featureReq.GetFeatureId(); got != "warding_sphere" {
		t.Fatalf("feature_id = %q, want warding_sphere", got)
	}
}

func TestRunStepAdversaryReactionStepAppliesDamageAndCooldown(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	var damageReq *daggerheartv1.DaggerheartApplyDamageRequest
	var updateReq *daggerheartv1.DaggerheartUpdateAdversaryRequest
	dhClient.applyDamage = func(_ context.Context, req *daggerheartv1.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error) {
		damageReq = req
		return &daggerheartv1.DaggerheartApplyDamageResponse{}, nil
	}
	dhClient.updateAdversary = func(_ context.Context, req *daggerheartv1.DaggerheartUpdateAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateAdversaryResponse, error) {
		updateReq = req
		return &daggerheartv1.DaggerheartUpdateAdversaryResponse{
			Adversary: &daggerheartv1.DaggerheartAdversary{Id: req.GetAdversaryId()},
		}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.adversaries["Saruman"] = "adv-saruman"
	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "adversary_reaction",
		Args: map[string]any{
			"actor":         "Saruman",
			"target":        "Frodo",
			"damage":        7,
			"damage_type":   "magic",
			"cooldown_note": "warding_sphere:cooldown",
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if damageReq == nil {
		t.Fatal("expected apply damage request")
	}
	if got := damageReq.GetCharacterId(); got != "char-frodo" {
		t.Fatalf("damage character_id = %q, want char-frodo", got)
	}
	if got := damageReq.GetDamage().GetAmount(); got != 7 {
		t.Fatalf("damage amount = %d, want 7", got)
	}
	if updateReq == nil {
		t.Fatal("expected update adversary request")
	}
	if got := updateReq.GetAdversaryId(); got != "adv-saruman" {
		t.Fatalf("update adversary_id = %q, want adv-saruman", got)
	}
	if got := updateReq.GetNotes().GetValue(); got != "warding_sphere:cooldown" {
		t.Fatalf("update notes = %q, want warding_sphere:cooldown", got)
	}
}

func TestRunStepAdversaryUpdateSetsSceneAndNotes(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	var updateReq *daggerheartv1.DaggerheartUpdateAdversaryRequest
	dhClient.updateAdversary = func(_ context.Context, req *daggerheartv1.DaggerheartUpdateAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateAdversaryResponse, error) {
		updateReq = req
		return &daggerheartv1.DaggerheartUpdateAdversaryResponse{
			Adversary: &daggerheartv1.DaggerheartAdversary{
				Id:      req.GetAdversaryId(),
				SceneId: req.GetSceneId(),
				Notes:   req.GetNotes().GetValue(),
			},
		}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Mirkwood Warden"] = "adv-warden"

	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "adversary_update",
		Args: map[string]any{
			"target":   "Mirkwood Warden",
			"scene_id": "scene-2",
			"notes":    "ferocious_defense",
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if updateReq == nil {
		t.Fatal("expected update adversary request")
	}
	if got := updateReq.GetSceneId(); got != "scene-2" {
		t.Fatalf("scene_id = %q, want scene-2", got)
	}
	if got := updateReq.GetNotes().GetValue(); got != "ferocious_defense" {
		t.Fatalf("notes = %q, want ferocious_defense", got)
	}
}

func TestRunStepAdversaryUpdateMapsLegacyStatMutationToNotes(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var updateReq *daggerheartv1.DaggerheartUpdateAdversaryRequest
	dhClient.updateAdversary = func(_ context.Context, req *daggerheartv1.DaggerheartUpdateAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartUpdateAdversaryResponse, error) {
		updateReq = req
		return &daggerheartv1.DaggerheartUpdateAdversaryResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.adversaries["Mirkwood Warden"] = "adv-warden"

	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "adversary_update",
		Args: map[string]any{
			"target":       "Mirkwood Warden",
			"stress_delta": 1,
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if updateReq == nil {
		t.Fatal("expected update adversary request")
	}
	if got := updateReq.GetNotes().GetValue(); got != "stress_delta=1" {
		t.Fatalf("notes = %q, want stress_delta=1", got)
	}
}

func TestRunStepGroupReactionAppliesFailureConditionsOnly(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	env.characterClient.(*fakeCharacterClient).getSheet = func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
		return readyDaggerheartSheetResponse(), nil
	}

	reactionRollCalls := 0
	applyReactionRollSeqs := make([]uint64, 0, 2)
	conditionTargets := make([]string, 0, 2)

	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		reactionRollCalls++
		if req.GetRollKind() != daggerheartv1.RollKind_ROLL_KIND_REACTION {
			t.Fatalf("roll kind = %v, want REACTION", req.GetRollKind())
		}
		if req.GetCharacterId() == "char-frodo" {
			return &daggerheartv1.SessionActionRollResponse{
				RollSeq:    201,
				HopeDie:    1,
				FearDie:    10,
				Total:      8,
				Difficulty: 15,
				Success:    false,
				Crit:       false,
			}, nil
		}
		if req.GetCharacterId() == "char-sam" {
			return &daggerheartv1.SessionActionRollResponse{
				RollSeq:    202,
				HopeDie:    9,
				FearDie:    2,
				Total:      16,
				Difficulty: 15,
				Success:    true,
				Crit:       false,
			}, nil
		}
		t.Fatalf("unexpected character_id %q", req.GetCharacterId())
		return nil, nil
	}
	dhClient.applyReactionOutcome = func(_ context.Context, req *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
		applyReactionRollSeqs = append(applyReactionRollSeqs, req.GetRollSeq())
		success := req.GetRollSeq() == 202
		outcome := daggerheartv1.Outcome_FAILURE_WITH_FEAR
		if success {
			outcome = daggerheartv1.Outcome_SUCCESS_WITH_HOPE
		}
		return &daggerheartv1.DaggerheartApplyReactionOutcomeResponse{
			Result: &daggerheartv1.DaggerheartReactionOutcomeResult{
				Success: success,
				Outcome: outcome,
			},
		}, nil
	}
	dhClient.applyConditions = func(_ context.Context, req *daggerheartv1.DaggerheartApplyConditionsRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyConditionsResponse, error) {
		conditionTargets = append(conditionTargets, req.GetCharacterId())
		return &daggerheartv1.DaggerheartApplyConditionsResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.actors["Sam"] = "char-sam"

	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "group_reaction",
		Args: map[string]any{
			"targets":    []any{"Frodo", "Sam"},
			"trait":      "agility",
			"difficulty": 15,
			"failure_conditions": []any{
				"VULNERABLE",
			},
			"source": "snowblind_trap",
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if reactionRollCalls != 2 {
		t.Fatalf("reaction roll calls = %d, want 2", reactionRollCalls)
	}
	if len(applyReactionRollSeqs) != 2 || applyReactionRollSeqs[0] != 201 || applyReactionRollSeqs[1] != 202 {
		t.Fatalf("apply reaction roll seqs = %v, want [201 202]", applyReactionRollSeqs)
	}
	if len(conditionTargets) != 1 || conditionTargets[0] != "char-frodo" {
		t.Fatalf("condition targets = %v, want [char-frodo]", conditionTargets)
	}
}

func TestRunStepGroupReactionAppliesHalfDamageOnSuccess(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	env.characterClient.(*fakeCharacterClient).getSheet = func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
		return readyDaggerheartSheetResponse(), nil
	}

	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		if req.GetCharacterId() == "char-frodo" {
			return &daggerheartv1.SessionActionRollResponse{
				RollSeq:    301,
				Total:      8,
				Difficulty: 15,
				Success:    false,
			}, nil
		}
		if req.GetCharacterId() == "char-sam" {
			return &daggerheartv1.SessionActionRollResponse{
				RollSeq:    302,
				Total:      16,
				Difficulty: 15,
				Success:    true,
			}, nil
		}
		t.Fatalf("unexpected character_id %q", req.GetCharacterId())
		return nil, nil
	}
	dhClient.applyReactionOutcome = func(_ context.Context, req *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
		success := req.GetRollSeq() == 302
		return &daggerheartv1.DaggerheartApplyReactionOutcomeResponse{
			Result: &daggerheartv1.DaggerheartReactionOutcomeResult{
				Success: success,
			},
		}, nil
	}

	applied := map[string]int32{}
	dhClient.applyDamage = func(_ context.Context, req *daggerheartv1.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error) {
		applied[req.GetCharacterId()] = req.GetDamage().GetAmount()
		return &daggerheartv1.DaggerheartApplyDamageResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.actors["Sam"] = "char-sam"

	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "group_reaction",
		Args: map[string]any{
			"targets":                []any{"Frodo", "Sam"},
			"trait":                  "agility",
			"difficulty":             15,
			"damage":                 9,
			"damage_type":            "magic",
			"half_damage_on_success": true,
			"source":                 "arcane_artillery",
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if got := len(applied); got != 2 {
		t.Fatalf("damage applications = %d, want 2", got)
	}
	if got := applied["char-frodo"]; got != 9 {
		t.Fatalf("frodo damage = %d, want 9", got)
	}
	if got := applied["char-sam"]; got != 4 {
		t.Fatalf("sam damage = %d, want 4", got)
	}
}

func TestRunAttackStepAdversaryTargetForwardsDisadvantage(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var actionRollReq *daggerheartv1.SessionActionRollRequest
	dhClient.sessionActionRoll = func(_ context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		actionRollReq = req
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 601}, nil
	}
	dhClient.applyRollOutcome = func(_ context.Context, _ *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	dhClient.applyAttackOutcome = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyAttackOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAttackOutcomeResponse, error) {
		return &daggerheartv1.DaggerheartApplyAttackOutcomeResponse{
			Result: &daggerheartv1.DaggerheartAttackOutcomeResult{Success: false},
		}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.adversaries["Ranger"] = "adv-ranger"
	err := runner.runAttackStep(context.Background(), state, Step{
		Kind: "attack",
		Args: map[string]any{
			"actor":        "Frodo",
			"target":       "Ranger",
			"trait":        "instinct",
			"difficulty":   10,
			"disadvantage": 1,
		},
	})
	if err != nil {
		t.Fatalf("runAttackStep: %v", err)
	}
	if actionRollReq == nil {
		t.Fatal("expected SessionActionRoll request")
	}
	if actionRollReq.GetDisadvantage() != 1 {
		t.Fatalf("disadvantage = %d, want 1", actionRollReq.GetDisadvantage())
	}
}

func TestRunAttackStepForwardsAttackRangeAndArmorBackedHopeSpend(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var attackReq *daggerheartv1.SessionAttackFlowRequest

	dhClient.sessionAttackFlow = func(_ context.Context, req *daggerheartv1.SessionAttackFlowRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionAttackFlowResponse, error) {
		attackReq = req
		return &daggerheartv1.SessionAttackFlowResponse{
			ActionRoll:    &daggerheartv1.SessionActionRollResponse{RollSeq: 11},
			AttackOutcome: &daggerheartv1.DaggerheartApplyAttackOutcomeResponse{Result: &daggerheartv1.DaggerheartAttackOutcomeResult{Success: false}},
		}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.actors["Sam"] = "char-sam"

	err := runner.runAttackStep(context.Background(), state, Step{
		Kind: "attack",
		Args: map[string]any{
			"actor":                   "Frodo",
			"target":                  "Sam",
			"trait":                   "instinct",
			"difficulty":              10,
			"attack_range":            "ranged",
			"replace_hope_with_armor": true,
			"modifiers": []any{
				map[string]any{"source": "experience"},
			},
		},
	})
	if err != nil {
		t.Fatalf("runAttackStep: %v", err)
	}
	if attackReq == nil {
		t.Fatal("expected SessionAttackFlow request")
	}
	if attackReq.GetStandardAttack() == nil {
		t.Fatal("expected standard attack profile")
	}
	if attackReq.GetStandardAttack().GetAttackRange() != daggerheartv1.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED {
		t.Fatalf("attack range = %v, want ranged", attackReq.GetStandardAttack().GetAttackRange())
	}
	if !attackReq.GetReplaceHopeWithArmor() {
		t.Fatal("replace_hope_with_armor = false, want true")
	}
}

// --- damage_roll step ---

func TestRunDamageRollStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunApplyRollOutcomeStepRunsSuccessBranch(t *testing.T) {
	fixture := testEnv()
	env, sessionClient, dhClient := fixture.env, fixture.sessionClient, fixture.daggerheartClient
	var cleared bool
	sessionClient.clearSpotlight = func(_ context.Context, _ *gamev1.ClearSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
		cleared = true
		return &gamev1.ClearSessionSpotlightResponse{}, nil
	}
	dhClient.applyRollOutcome = func(_ context.Context, req *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	state.rollOutcomes[42] = actionRollResult{
		rollSeq: 42,
		success: true,
	}
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{
			"on_success": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err != nil {
		t.Fatalf("runApplyRollOutcomeStep: %v", err)
	}
	if !cleared {
		t.Fatal("expected success branch step to run")
	}
}

func TestRunApplyRollOutcomeStepRunsFailureBranchForFailureResult(t *testing.T) {
	fixture := testEnv()
	env, sessionClient, dhClient := fixture.env, fixture.sessionClient, fixture.daggerheartClient
	var cleared bool
	sessionClient.clearSpotlight = func(_ context.Context, _ *gamev1.ClearSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
		cleared = true
		return &gamev1.ClearSessionSpotlightResponse{}, nil
	}
	dhClient.applyRollOutcome = func(_ context.Context, req *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	state.rollOutcomes[42] = actionRollResult{
		rollSeq: 42,
		success: false,
	}
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{
			"on_failure": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err != nil {
		t.Fatalf("runApplyRollOutcomeStep: %v", err)
	}
	if !cleared {
		t.Fatal("expected failure branch step to run")
	}
}

func TestRunApplyRollOutcomeStepInvalidBranchSkipsApply(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	called := false
	dhClient.applyRollOutcome = func(_ context.Context, req *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		called = true
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	state.rollOutcomes[42] = actionRollResult{
		rollSeq: 42,
		success: true,
	}
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{
			"on_magic": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown outcome branch") {
		t.Fatalf("expected unknown outcome branch error, got %v", err)
	}
	if called {
		t.Fatal("expected apply roll outcome to be skipped")
	}
}

func TestRunApplyRollOutcomeStepRunsFailureHopeBranchForFailureHopeResult(t *testing.T) {
	fixture := testEnv()
	env, sessionClient, dhClient := fixture.env, fixture.sessionClient, fixture.daggerheartClient
	var cleared bool
	sessionClient.clearSpotlight = func(_ context.Context, _ *gamev1.ClearSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
		cleared = true
		return &gamev1.ClearSessionSpotlightResponse{}, nil
	}
	dhClient.applyRollOutcome = func(_ context.Context, req *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	state.rollOutcomes[42] = actionRollResult{
		rollSeq: 42,
		success: false,
		hopeDie: 6,
		fearDie: 1,
	}
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{
			"on_failure_hope": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err != nil {
		t.Fatalf("runApplyRollOutcomeStep: %v", err)
	}
	if !cleared {
		t.Fatal("expected failure_hope branch step to run")
	}
}

func TestRunApplyRollOutcomeStepMissingOutcomeMetadata(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.applyRollOutcome = func(_ context.Context, req *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyRollOutcomeStep(context.Background(), state, Step{
		Kind: "apply_roll_outcome",
		Args: map[string]any{
			"on_success": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "missing action roll outcome") {
		t.Fatalf("expected missing action roll outcome error, got %v", err)
	}
}

// --- apply_attack_outcome step ---

func TestRunApplyAttackOutcomeStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunMultiAttackStep_DeduplicatesTargetsForDamageFanout(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	var attackOutcomeTargets []string
	applyDamageCalls := 0
	dhClient.sessionActionRoll = func(_ context.Context, _ *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
		return &daggerheartv1.SessionActionRollResponse{RollSeq: 101}, nil
	}
	dhClient.applyRollOutcome = func(_ context.Context, _ *daggerheartv1.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.ApplyRollOutcomeResponse, error) {
		return &daggerheartv1.ApplyRollOutcomeResponse{}, nil
	}
	dhClient.applyAttackOutcome = func(_ context.Context, req *daggerheartv1.DaggerheartApplyAttackOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAttackOutcomeResponse, error) {
		attackOutcomeTargets = append(attackOutcomeTargets, req.GetTargets()...)
		return &daggerheartv1.DaggerheartApplyAttackOutcomeResponse{
			Result: &daggerheartv1.DaggerheartAttackOutcomeResult{Success: true},
		}, nil
	}
	dhClient.sessionDamageRoll = func(_ context.Context, _ *daggerheartv1.SessionDamageRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionDamageRollResponse, error) {
		return &daggerheartv1.SessionDamageRollResponse{RollSeq: 202, Total: 0}, nil
	}
	dhClient.applyDamage = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error) {
		applyDamageCalls++
		return &daggerheartv1.DaggerheartApplyDamageResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Sam"] = "char-sam"
	state.actors["Frodo"] = "char-frodo"

	err := runner.runMultiAttackStep(context.Background(), state, Step{
		Kind: "multi_attack",
		Args: map[string]any{
			"actor":      "Sam",
			"targets":    []any{"Frodo", "Frodo"},
			"difficulty": 10,
			"seed":       42,
			"damage_dice": []any{
				map[string]any{"sides": 6, "count": 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("runMultiAttackStep: %v", err)
	}
	if len(attackOutcomeTargets) != 1 {
		t.Fatalf("apply_attack_outcome targets len = %d, want 1 unique target", len(attackOutcomeTargets))
	}
	if applyDamageCalls != 1 {
		t.Fatalf("apply damage calls = %d, want 1 unique target application", applyDamageCalls)
	}
}

func TestRunCombinedDamageStep_RejectsDuplicateContributors(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	applyDamageCalled := false
	dhClient.applyDamage = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error) {
		applyDamageCalled = true
		return &daggerheartv1.DaggerheartApplyDamageResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.actors["Rat A"] = "char-rat-a"

	err := runner.runCombinedDamageStep(context.Background(), state, Step{
		Kind: "combined_damage",
		Args: map[string]any{
			"target":          "Frodo",
			"immune_physical": true,
			"sources": []any{
				map[string]any{"character": "Rat A", "amount": 1},
				map[string]any{"character": "Rat A", "amount": 1},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate source character") {
		t.Fatalf("expected duplicate source character error, got %v", err)
	}
	if applyDamageCalled {
		t.Fatal("expected combined_damage to fail before ApplyDamage call")
	}
}

func TestRunCombinedDamageStep_DoesNotFakeOverflowDeletion(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	applyAdversaryDamageCalls := 0
	dhClient.applyAdversaryDamage = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyAdversaryDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryDamageResponse, error) {
		applyAdversaryDamageCalls++
		return &daggerheartv1.DaggerheartApplyAdversaryDamageResponse{
			AdversaryId: "adv-rat-a",
			Adversary:   &daggerheartv1.DaggerheartAdversary{Id: "adv-rat-a"},
		}, nil
	}
	dhClient.deleteAdversary = func(_ context.Context, req *daggerheartv1.DaggerheartDeleteAdversaryRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartDeleteAdversaryResponse, error) {
		t.Fatalf("unexpected delete adversary call: %s", req.GetAdversaryId())
		return nil, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.adversaries["Moria Rat A"] = "adv-rat-a"
	state.adversaries["Moria Rat B"] = "adv-rat-b"
	state.adversaries["Moria Rat C"] = "adv-rat-c"

	err := runner.runCombinedDamageStep(context.Background(), state, Step{
		Kind: "combined_damage",
		Args: map[string]any{
			"target":      "Moria Rat A",
			"damage_type": "physical",
			"sources": []any{
				map[string]any{"character": "Frodo", "amount": 6},
			},
		},
	})
	if err != nil {
		t.Fatalf("runCombinedDamageStep: %v", err)
	}
	if applyAdversaryDamageCalls != 1 {
		t.Fatalf("apply adversary damage calls = %d, want 1", applyAdversaryDamageCalls)
	}
}

func TestRunCombinedDamageStepForwardsImpenetrableReaction(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var applyDamageReq *daggerheartv1.DaggerheartApplyDamageRequest
	currentState := &daggerheartv1.DaggerheartCharacterState{Hp: 2, Armor: 1}
	env.characterClient = &fakeCharacterClient{
		getSheet: func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
			return &gamev1.GetCharacterSheetResponse{
				State: &gamev1.CharacterState{
					SystemState: &gamev1.CharacterState_Daggerheart{
						Daggerheart: currentState,
					},
				},
			}, nil
		},
	}

	dhClient.applyDamage = func(_ context.Context, req *daggerheartv1.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDamageResponse, error) {
		applyDamageReq = req
		currentState = &daggerheartv1.DaggerheartCharacterState{Hp: 1, Stress: 2, Armor: 0}
		return &daggerheartv1.DaggerheartApplyDamageResponse{
			CharacterId: req.GetCharacterId(),
			State:       currentState,
		}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	state.actors["Rat A"] = "char-rat-a"

	err := runner.runCombinedDamageStep(context.Background(), state, Step{
		Kind: "combined_damage",
		Args: map[string]any{
			"target":         "Frodo",
			"damage_type":    "physical",
			"armor_reaction": "impenetrable",
			"sources": []any{
				map[string]any{"character": "Rat A", "amount": 10},
			},
		},
	})
	if err != nil {
		t.Fatalf("runCombinedDamageStep: %v", err)
	}
	if applyDamageReq == nil {
		t.Fatal("expected ApplyDamage request")
	}
	if applyDamageReq.GetArmorReaction() == nil || applyDamageReq.GetArmorReaction().GetImpenetrable() == nil {
		t.Fatalf("armor reaction = %v, want impenetrable", applyDamageReq.GetArmorReaction())
	}
}

// --- apply_reaction_outcome step ---

func TestRunApplyReactionOutcomeStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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

func TestRunApplyReactionOutcomeStepRunsOutcomeBranch(t *testing.T) {
	fixture := testEnv()
	env, sessionClient, dhClient := fixture.env, fixture.sessionClient, fixture.daggerheartClient
	var spotlightCleared bool
	sessionClient.clearSpotlight = func(_ context.Context, _ *gamev1.ClearSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
		spotlightCleared = true
		return &gamev1.ClearSessionSpotlightResponse{}, nil
	}
	dhClient.applyReactionOutcome = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
		return &daggerheartv1.DaggerheartApplyReactionOutcomeResponse{
			Result: &daggerheartv1.DaggerheartReactionOutcomeResult{Success: true},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyReactionOutcomeStep(context.Background(), state, Step{
		Kind: "apply_reaction_outcome",
		Args: map[string]any{
			"on_success": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err != nil {
		t.Fatalf("runApplyReactionOutcomeStep: %v", err)
	}
	if !spotlightCleared {
		t.Fatal("expected success branch step to run")
	}
}

func TestRunApplyReactionOutcomeStepRunsFearSubbranchForFailureFearResult(t *testing.T) {
	fixture := testEnv()
	env, sessionClient, dhClient := fixture.env, fixture.sessionClient, fixture.daggerheartClient
	var cleared bool
	sessionClient.clearSpotlight = func(_ context.Context, _ *gamev1.ClearSessionSpotlightRequest, _ ...grpc.CallOption) (*gamev1.ClearSessionSpotlightResponse, error) {
		cleared = true
		return &gamev1.ClearSessionSpotlightResponse{}, nil
	}
	dhClient.applyReactionOutcome = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
		return &daggerheartv1.DaggerheartApplyReactionOutcomeResponse{
			Result: &daggerheartv1.DaggerheartReactionOutcomeResult{
				Success: false,
				Outcome: daggerheartv1.Outcome_FAILURE_WITH_FEAR,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyReactionOutcomeStep(context.Background(), state, Step{
		Kind: "apply_reaction_outcome",
		Args: map[string]any{
			"on_failure_fear": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err != nil {
		t.Fatalf("runApplyReactionOutcomeStep: %v", err)
	}
	if !cleared {
		t.Fatal("expected failure_fear branch step to run")
	}
}

func TestRunApplyReactionOutcomeStepInvalidBranchSkipsApply(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	called := false
	dhClient.applyReactionOutcome = func(_ context.Context, req *daggerheartv1.DaggerheartApplyReactionOutcomeRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyReactionOutcomeResponse, error) {
		called = true
		return &daggerheartv1.DaggerheartApplyReactionOutcomeResponse{
			Result: &daggerheartv1.DaggerheartReactionOutcomeResult{Success: true},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.lastRollSeq = 42
	err := runner.runApplyReactionOutcomeStep(context.Background(), state, Step{
		Kind: "apply_reaction_outcome",
		Args: map[string]any{
			"on_magic": []any{
				map[string]any{"kind": "clear_spotlight"},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown outcome branch") {
		t.Fatalf("expected unknown outcome branch error, got %v", err)
	}
	if called {
		t.Fatal("expected apply reaction outcome to be skipped")
	}
}

// --- gm_spend_fear step ---

func TestRunGMSpendFearStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		target := req.GetDirectMove()
		if target == nil {
			t.Fatal("expected direct move target")
		}
		if target.GetKind() != daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE {
			t.Fatalf("kind = %v, want additional_move", target.GetKind())
		}
		if target.GetShape() != daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY {
			t.Fatalf("shape = %v, want spotlight_adversary", target.GetShape())
		}
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

func TestRunGMSpendFearStepDescriptionDefaultsToCustomMove(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var gotRequest *daggerheartv1.DaggerheartApplyGmMoveRequest
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		gotRequest = req
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 1}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 2
	state.adversaries["Shire Elder"] = "adv-elder"
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{
			"amount":      1,
			"target":      "Shire Elder",
			"description": "there_will_be_peace_rebuke",
		},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
	if gotRequest == nil || gotRequest.GetDirectMove() == nil {
		t.Fatal("expected direct move request")
	}
	if got := gotRequest.GetDirectMove().GetShape(); got != daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM {
		t.Fatalf("shape = %v, want custom", got)
	}
	if got := gotRequest.GetDirectMove().GetAdversaryId(); got != "adv-elder" {
		t.Fatalf("adversary_id = %q, want adv-elder", got)
	}
}

func TestRunGMSpendFearStepUnknownSpotlightTargetFallsBackToCustom(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var gotRequest *daggerheartv1.DaggerheartApplyGmMoveRequest
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		gotRequest = req
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 0}, nil
	}
	dhClient.createEnvironmentEntity = func(_ context.Context, req *daggerheartv1.DaggerheartCreateEnvironmentEntityRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateEnvironmentEntityResponse, error) {
		return &daggerheartv1.DaggerheartCreateEnvironmentEntityResponse{
			EnvironmentEntity: &daggerheartv1.DaggerheartEnvironmentEntity{
				Id:            "env-entity-1",
				EnvironmentId: req.GetEnvironmentId(),
				SessionId:     req.GetSessionId(),
				SceneId:       req.GetSceneId(),
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 1
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{
			"amount": 1,
			"move":   "spotlight",
			"target": "Bruinen Ford",
		},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
	if gotRequest == nil || gotRequest.GetDirectMove() == nil {
		t.Fatal("expected direct move request")
	}
	if got := gotRequest.GetDirectMove().GetShape(); got != daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM {
		t.Fatalf("shape = %v, want custom", got)
	}
	if got := gotRequest.GetDirectMove().GetDescription(); got != "spotlight Bruinen Ford" {
		t.Fatalf("description = %q, want spotlight Bruinen Ford", got)
	}
}

func TestRunGMSpendFearStepAdversaryFeature(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var gotRequest *daggerheartv1.DaggerheartApplyGmMoveRequest
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		gotRequest = req
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 0}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 1
	state.adversaries["Shadow Hound"] = "adv-1"
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{
			"amount":       1,
			"spend_target": "adversary_feature",
			"target":       "Shadow Hound",
			"feature_id":   "feature.shadow-hound-pounce",
			"description":  "Leap from shadow",
		},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
	if gotRequest == nil || gotRequest.GetAdversaryFeature() == nil {
		t.Fatal("expected adversary feature request")
	}
	if gotRequest.GetAdversaryFeature().GetAdversaryId() != "adv-1" {
		t.Fatalf("adversary_id = %q", gotRequest.GetAdversaryFeature().GetAdversaryId())
	}
	if gotRequest.GetAdversaryFeature().GetFeatureId() != "feature.shadow-hound-pounce" {
		t.Fatalf("feature_id = %q", gotRequest.GetAdversaryFeature().GetFeatureId())
	}
}

func TestRunGMSpendFearStepEnvironmentFeature(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var gotRequest *daggerheartv1.DaggerheartApplyGmMoveRequest
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		gotRequest = req
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 0}, nil
	}
	dhClient.createEnvironmentEntity = func(_ context.Context, req *daggerheartv1.DaggerheartCreateEnvironmentEntityRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartCreateEnvironmentEntityResponse, error) {
		return &daggerheartv1.DaggerheartCreateEnvironmentEntityResponse{
			EnvironmentEntity: &daggerheartv1.DaggerheartEnvironmentEntity{
				Id:            "env-entity-1",
				EnvironmentId: req.GetEnvironmentId(),
				SessionId:     req.GetSessionId(),
				SceneId:       req.GetSceneId(),
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 2
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{
			"amount":         2,
			"spend_target":   "environment_feature",
			"environment_id": "environment.crumbling-bridge",
			"feature_id":     "feature.crumbling-bridge-falling-stones",
		},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
	if gotRequest == nil || gotRequest.GetEnvironmentFeature() == nil {
		t.Fatal("expected environment feature request")
	}
	if gotRequest.GetEnvironmentFeature().GetEnvironmentEntityId() == "" {
		t.Fatal("environment_entity_id should be populated")
	}
	if gotRequest.GetEnvironmentFeature().GetFeatureId() != "feature.crumbling-bridge-falling-stones" {
		t.Fatalf("feature_id = %q", gotRequest.GetEnvironmentFeature().GetFeatureId())
	}
}

func TestRunGMSpendFearStepAdversaryExperience(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var gotRequest *daggerheartv1.DaggerheartApplyGmMoveRequest
	dhClient.applyGmMove = func(_ context.Context, req *daggerheartv1.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyGmMoveResponse, error) {
		gotRequest = req
		return &daggerheartv1.DaggerheartApplyGmMoveResponse{GmFearAfter: 0}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 1
	state.adversaries["Shadow Hound"] = "adv-1"
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{
			"amount":          1,
			"spend_target":    "adversary_experience",
			"target":          "Shadow Hound",
			"experience_name": "Pack Hunter",
		},
	})
	if err != nil {
		t.Fatalf("runGMSpendFearStep: %v", err)
	}
	if gotRequest == nil || gotRequest.GetAdversaryExperience() == nil {
		t.Fatal("expected adversary experience request")
	}
	if gotRequest.GetAdversaryExperience().GetAdversaryId() != "adv-1" {
		t.Fatalf("adversary_id = %q", gotRequest.GetAdversaryExperience().GetAdversaryId())
	}
	if gotRequest.GetAdversaryExperience().GetExperienceName() != "Pack Hunter" {
		t.Fatalf("experience_name = %q", gotRequest.GetAdversaryExperience().GetExperienceName())
	}
}

func TestRunGMSpendFearStepZeroAmount(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	state := testState()
	state.gmFear = 3
	err := runner.runGMSpendFearStep(context.Background(), state, Step{
		Kind: "gm_spend_fear",
		Args: map[string]any{"amount": 0, "move": "spotlight"},
	})
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected greater-than-zero error, got %v", err)
	}
}

// --- set_spotlight step ---

func TestRunSetSpotlightStepGM(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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

func TestRunApplyConditionStepAdversaryTarget(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	var request *daggerheartv1.DaggerheartApplyAdversaryConditionsRequest
	dhClient.applyAdversaryConditions = func(_ context.Context, req *daggerheartv1.DaggerheartApplyAdversaryConditionsRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyAdversaryConditionsResponse, error) {
		request = req
		return &daggerheartv1.DaggerheartApplyAdversaryConditionsResponse{}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.adversaries["Orc Stalker"] = "adv-stalker"
	err := runner.runApplyConditionStep(context.Background(), state, Step{
		Kind: "apply_condition",
		Args: map[string]any{"target": "Orc Stalker", "add": []any{"HIDDEN"}, "source": "cloaked"},
	})
	if err != nil {
		t.Fatalf("runApplyConditionStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected adversary conditions request")
	}
	if request.GetAdversaryId() != "adv-stalker" {
		t.Fatalf("adversary_id = %q, want adv-stalker", request.GetAdversaryId())
	}
	if got := request.GetSource(); got != "cloaked" {
		t.Fatalf("source = %q, want cloaked", got)
	}
	if len(request.GetAddConditions()) != 1 || request.GetAddConditions()[0].GetCode() != "hidden" {
		t.Fatalf("add conditions = %v, want [hidden]", request.GetAddConditions())
	}
}

func TestRunApplyConditionStepMissingTarget(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunStepTemporaryArmor(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient

	var request *daggerheartv1.DaggerheartApplyTemporaryArmorRequest
	dhClient.applyTemporaryArmor = func(_ context.Context, req *daggerheartv1.DaggerheartApplyTemporaryArmorRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyTemporaryArmorResponse, error) {
		request = req
		return &daggerheartv1.DaggerheartApplyTemporaryArmorResponse{}, nil
	}

	runner := quietRunner(env)
	state := testState()
	state.actors["Gandalf"] = "char-gandalf"

	err := runner.runStep(context.Background(), state, Step{
		System: "DAGGERHEART",
		Kind:   "temporary_armor",
		Args: map[string]any{
			"target":    "Gandalf",
			"source":    "ritual",
			"duration":  "short_rest",
			"amount":    2,
			"source_id": "blessing:1",
		},
	})
	if err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if request == nil {
		t.Fatal("expected apply temporary armor request")
	}
	if got := request.GetCharacterId(); got != "char-gandalf" {
		t.Fatalf("character_id = %q, want char-gandalf", got)
	}
	if got := request.GetArmor().GetSource(); got != "ritual" {
		t.Fatalf("source = %q, want ritual", got)
	}
	if got := request.GetArmor().GetDuration(); got != "short_rest" {
		t.Fatalf("duration = %q, want short_rest", got)
	}
	if got := request.GetArmor().GetAmount(); got != 2 {
		t.Fatalf("amount = %d, want 2", got)
	}
	if got := request.GetArmor().GetSourceId(); got != "blessing:1" {
		t.Fatalf("source_id = %q, want blessing:1", got)
	}
}

func TestRunRestStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.applyRest = func(_ context.Context, in *daggerheartv1.DaggerheartApplyRestRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyRestResponse, error) {
		if len(in.GetRest().GetParticipants()) != 1 || in.GetRest().GetParticipants()[0].GetCharacterId() != "char-frodo" {
			t.Fatalf("participants = %+v, want char-frodo", in.GetRest().GetParticipants())
		}
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
	fixture := testEnv()
	env := fixture.env
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

// --- death_move step ---

func TestRunDeathMoveStep(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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

func TestRunDeathMoveStepAssertsExpectedDeathOutcome(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.applyDeathMove = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyDeathMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDeathMoveResponse, error) {
		hopeDie := int32(7)
		return &daggerheartv1.DaggerheartApplyDeathMoveResponse{
			Result: &daggerheartv1.DaggerheartDeathMoveResult{
				LifeState:  daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD,
				ScarGained: true,
				HopeDie:    &hopeDie,
			},
			State: &daggerheartv1.DaggerheartCharacterState{
				HopeMax:   0,
				LifeState: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runDeathMoveStep(context.Background(), state, Step{
		Kind: "death_move",
		Args: map[string]any{
			"target":             "Frodo",
			"move":               "avoid_death",
			"expect_life_state":  "dead",
			"expect_scar_gained": true,
			"expect_hope_die":    7,
			"expect_hope_max":    0,
		},
	})
	if err != nil {
		t.Fatalf("runDeathMoveStep: %v", err)
	}
}

func TestRunDeathMoveStepRejectsMismatchedDeathOutcome(t *testing.T) {
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
	dhClient.applyDeathMove = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyDeathMoveRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyDeathMoveResponse, error) {
		return &daggerheartv1.DaggerheartApplyDeathMoveResponse{
			Result: &daggerheartv1.DaggerheartDeathMoveResult{
				LifeState:  daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
				ScarGained: false,
			},
			State: &daggerheartv1.DaggerheartCharacterState{
				HopeMax:   1,
				LifeState: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
			},
		}, nil
	}
	runner := quietRunner(env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	err := runner.runDeathMoveStep(context.Background(), state, Step{
		Kind: "death_move",
		Args: map[string]any{
			"target":            "Frodo",
			"move":              "avoid_death",
			"expect_life_state": "dead",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "death_move life_state") {
		t.Fatalf("expected life_state assertion failure, got %v", err)
	}
}

func TestRunDeathMoveStepMissingTarget(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env, dhClient := fixture.env, fixture.daggerheartClient
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
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
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	err := runner.RunScenario(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "scenario is required") {
		t.Fatalf("expected scenario required, got %v", err)
	}
}

func TestRunScenarioStepError(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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

func TestRunScenarioSystemStepRequiresScopeWithoutCampaign(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
	runner := quietRunner(env)
	err := runner.RunScenario(context.Background(), &Scenario{
		Name: "test",
		Steps: []Step{
			{Kind: "attack", Args: map[string]any{"actor": "Frodo", "target": "Nazgul"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires explicit system scope") {
		t.Fatalf("expected explicit system scope error, got %v", err)
	}
}

func TestRunScenarioCampaignAndSession(t *testing.T) {
	fixture := testEnv()
	env := fixture.env
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
		{"parseGameSystem prefixed valid", func() error { _, err := parseGameSystem("GAME_SYSTEM_DAGGERHEART"); return err }},
		{"parseGameSystem invalid", func() error { _, err := parseGameSystem("BOGUS"); return err }},
		{"parseGameSystem unspecified invalid", func() error { _, err := parseGameSystem("GAME_SYSTEM_UNSPECIFIED"); return err }},
		{"parseGmMode valid", func() error { _, err := parseGmMode("HUMAN"); return err }},
		{"parseGmMode hybrid valid", func() error { _, err := parseGmMode("HYBRID"); return err }},
		{"parseGmMode invalid", func() error { _, err := parseGmMode("BOGUS"); return err }},
		{"parseCampaignIntent valid", func() error { _, err := parseCampaignIntent("SANDBOX"); return err }},
		{"parseCampaignIntent invalid", func() error { _, err := parseCampaignIntent("BOGUS"); return err }},
		{"parseCampaignAccessPolicy valid", func() error { _, err := parseCampaignAccessPolicy("PRIVATE"); return err }},
		{"parseCampaignAccessPolicy invalid", func() error { _, err := parseCampaignAccessPolicy("BOGUS"); return err }},
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
				map[string]any{"source": "experience"}, // missing value, skip
				map[string]any{"source": "x"},          // missing value, skip
				"not-a-map",                            // skip
			},
		}, "mods")
		if len(mods) != 1 {
			t.Fatalf("got %d modifiers, want 1", len(mods))
		}
	})
	t.Run("buildActionRollHopeSpends", func(t *testing.T) {
		spends := buildActionRollHopeSpends(map[string]any{
			"hope_spends": []any{
				map[string]any{"source": "experience", "amount": 1},
				map[string]any{"source": "hope_feature"},
				"not-a-map",
			},
		}, "hope_spends")
		if len(spends) != 2 {
			t.Fatalf("got %d spends, want 2", len(spends))
		}
		if spends[1].GetAmount() != 3 {
			t.Fatalf("second amount = %d, want 3", spends[1].GetAmount())
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
	fixture := testEnv()
	env := fixture.env
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

func TestRunLevelUpStepBuildsStageBasedRequestAndChecksSubclassExpectations(t *testing.T) {
	fixture := testEnv()
	var req *daggerheartv1.DaggerheartApplyLevelUpRequest
	fixture.daggerheartClient.applyLevelUp = func(_ context.Context, in *daggerheartv1.DaggerheartApplyLevelUpRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyLevelUpResponse, error) {
		req = in
		return &daggerheartv1.DaggerheartApplyLevelUpResponse{}, nil
	}
	listCalls := 0
	fixture.eventClient.listEvents = func(_ context.Context, _ *gamev1.ListEventsRequest, _ ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
		listCalls++
		if listCalls == 1 {
			return &gamev1.ListEventsResponse{
				Events: []*gamev1.Event{{Seq: 1, Type: "baseline"}},
			}, nil
		}
		return &gamev1.ListEventsResponse{
			Events: []*gamev1.Event{{Seq: 2, Type: "sys.daggerheart.level_up_applied"}},
		}, nil
	}
	fixture.env.characterClient.(*fakeCharacterClient).getSheet = func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
		return &gamev1.GetCharacterSheetResponse{
			Profile: &gamev1.CharacterProfile{
				SystemProfile: &gamev1.CharacterProfile_Daggerheart{
					Daggerheart: &daggerheartv1.DaggerheartProfile{
						Level: 2,
						SubclassTracks: []*daggerheartv1.DaggerheartSubclassTrack{{
							Origin:     daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY,
							ClassId:    "class.guardian",
							SubclassId: "subclass.stalwart",
							Rank:       daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_SPECIALIZATION,
						}},
						ActiveSubclassFeatures: []*daggerheartv1.DaggerheartActiveSubclassTrackFeatures{{
							FoundationFeatures:     []*daggerheartv1.DaggerheartActiveSubclassFeature{{Id: "feature.stalwart-unwavering"}},
							SpecializationFeatures: []*daggerheartv1.DaggerheartActiveSubclassFeature{{Id: "feature.stalwart-unrelenting"}},
						}},
					},
				},
			},
		}, nil
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	step := Step{
		Kind: "level_up",
		Args: map[string]any{
			"target":                       "Frodo",
			"level_after":                  2,
			"expect_level":                 2,
			"expect_subclass_track_count":  1,
			"expect_primary_subclass_rank": "specialization",
			"expect_active_feature_ids":    []any{"feature.stalwart-unwavering", "feature.stalwart-unrelenting"},
			"advancements": []any{
				map[string]any{"type": "upgraded_subclass"},
				map[string]any{"type": "add_hp_slots"},
			},
		},
	}

	if err := runner.runLevelUpStep(context.Background(), state, step); err != nil {
		t.Fatalf("runLevelUpStep returned error: %v", err)
	}
	if req == nil {
		t.Fatal("expected ApplyLevelUp request")
	}
	if req.GetCharacterId() != "char-frodo" || req.GetLevelAfter() != 2 {
		t.Fatalf("request = %+v", req)
	}
	if got := req.GetAdvancements(); len(got) != 2 || got[0].GetType() != "upgraded_subclass" || got[1].GetType() != "add_hp_slots" {
		t.Fatalf("advancements = %+v", got)
	}
}

func TestRunLevelUpStepBuildsMulticlassWithoutFoundationCard(t *testing.T) {
	fixture := testEnv()
	var req *daggerheartv1.DaggerheartApplyLevelUpRequest
	fixture.daggerheartClient.applyLevelUp = func(_ context.Context, in *daggerheartv1.DaggerheartApplyLevelUpRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyLevelUpResponse, error) {
		req = in
		return &daggerheartv1.DaggerheartApplyLevelUpResponse{}, nil
	}
	listCalls := 0
	fixture.eventClient.listEvents = func(_ context.Context, _ *gamev1.ListEventsRequest, _ ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
		listCalls++
		if listCalls == 1 {
			return &gamev1.ListEventsResponse{
				Events: []*gamev1.Event{{Seq: 1, Type: "baseline"}},
			}, nil
		}
		return &gamev1.ListEventsResponse{
			Events: []*gamev1.Event{{Seq: 2, Type: "sys.daggerheart.level_up_applied"}},
		}, nil
	}
	fixture.env.characterClient.(*fakeCharacterClient).getSheet = func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
		return &gamev1.GetCharacterSheetResponse{
			Profile: &gamev1.CharacterProfile{
				SystemProfile: &gamev1.CharacterProfile_Daggerheart{
					Daggerheart: &daggerheartv1.DaggerheartProfile{
						Level: 6,
						SubclassTracks: []*daggerheartv1.DaggerheartSubclassTrack{
							{Origin: daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY, ClassId: "class.guardian", SubclassId: "subclass.stalwart", Rank: daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_FOUNDATION},
							{Origin: daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS, ClassId: "class.bard", SubclassId: "subclass.wordsmith", Rank: daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_FOUNDATION, DomainId: "domain.codex"},
						},
						ActiveSubclassFeatures: []*daggerheartv1.DaggerheartActiveSubclassTrackFeatures{{
							FoundationFeatures: []*daggerheartv1.DaggerheartActiveSubclassFeature{{Id: "feature.wordsmith-foundation"}},
						}},
					},
				},
			},
		}, nil
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.actors["Frodo"] = "char-frodo"
	step := Step{
		Kind: "level_up",
		Args: map[string]any{
			"target":                        "Frodo",
			"level_after":                   6,
			"expect_level":                  6,
			"expect_subclass_track_count":   2,
			"expect_multiclass_subclass_id": "subclass.wordsmith",
			"expect_active_feature_ids":     []string{"feature.wordsmith-foundation"},
			"advancements": []any{
				map[string]any{
					"type": "multiclass",
					"multiclass": map[string]any{
						"secondary_class_id":    "class.bard",
						"secondary_subclass_id": "subclass.wordsmith",
						"spellcast_trait":       "presence",
						"domain_id":             "domain.codex",
					},
				},
			},
		},
	}

	if err := runner.runLevelUpStep(context.Background(), state, step); err != nil {
		t.Fatalf("runLevelUpStep returned error: %v", err)
	}
	if req == nil || len(req.GetAdvancements()) != 1 || req.GetAdvancements()[0].GetMulticlass() == nil {
		t.Fatalf("request = %+v", req)
	}
	if req.GetAdvancements()[0].GetMulticlass().GetSecondaryClassId() != "class.bard" || req.GetAdvancements()[0].GetMulticlass().GetDomainId() != "domain.codex" {
		t.Fatalf("multiclass payload = %+v", req.GetAdvancements()[0].GetMulticlass())
	}
}

func TestRunApplyStatModifierStepWaitsForProjectedModifiers(t *testing.T) {
	fixture := testEnv()
	modifier := &daggerheartv1.DaggerheartStatModifier{
		Id:            "mod-evasion-wall",
		Target:        "evasion",
		Delta:         100,
		Label:         "Wall of Iron",
		Source:        "domain_card",
		ClearTriggers: []daggerheartv1.DaggerheartConditionClearTrigger{daggerheartv1.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST},
	}
	fixture.daggerheartClient.applyStatModifiers = func(_ context.Context, _ *daggerheartv1.DaggerheartApplyStatModifiersRequest, _ ...grpc.CallOption) (*daggerheartv1.DaggerheartApplyStatModifiersResponse, error) {
		return &daggerheartv1.DaggerheartApplyStatModifiersResponse{
			ActiveModifiers: []*daggerheartv1.DaggerheartStatModifier{modifier},
			Added:           []*daggerheartv1.DaggerheartStatModifier{modifier},
		}, nil
	}

	listCalls := 0
	fixture.eventClient.listEvents = func(_ context.Context, _ *gamev1.ListEventsRequest, _ ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
		listCalls++
		if listCalls == 1 {
			return &gamev1.ListEventsResponse{
				Events: []*gamev1.Event{{Seq: 1, Type: "baseline"}},
			}, nil
		}
		return &gamev1.ListEventsResponse{
			Events: []*gamev1.Event{{Seq: 2, Type: "sys.daggerheart.stat_modifier_changed"}},
		}, nil
	}

	getSheetCalls := 0
	fixture.env.characterClient.(*fakeCharacterClient).getSheet = func(_ context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
		getSheetCalls++
		state := &daggerheartv1.DaggerheartCharacterState{}
		if getSheetCalls >= 2 {
			state.StatModifiers = []*daggerheartv1.DaggerheartStatModifier{modifier}
		}
		return &gamev1.GetCharacterSheetResponse{
			State: &gamev1.CharacterState{
				SystemState: &gamev1.CharacterState_Daggerheart{
					Daggerheart: state,
				},
			},
		}, nil
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.actors["Rogue"] = "char-rogue"

	err := runner.runApplyStatModifierStep(context.Background(), state, Step{
		Kind: "apply_stat_modifier",
		Args: map[string]any{
			"target": "Rogue",
			"source": "domain_card.wall_of_iron",
			"add": []any{
				map[string]any{
					"id":             "mod-evasion-wall",
					"target":         "evasion",
					"delta":          100,
					"label":          "Wall of Iron",
					"source":         "domain_card",
					"clear_triggers": []any{"SHORT_REST"},
				},
			},
			"expect_active_count": 1,
			"expect_added_count":  1,
		},
	})
	if err != nil {
		t.Fatalf("runApplyStatModifierStep returned error: %v", err)
	}
	if getSheetCalls < 2 {
		t.Fatalf("getSheet calls = %d, want projection retry", getSheetCalls)
	}
}

func TestSubclassTrackHelpers(t *testing.T) {
	tracks := []*daggerheartv1.DaggerheartSubclassTrack{
		{
			Origin:     daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY,
			ClassId:    "class.guardian",
			SubclassId: "subclass.stalwart",
			Rank:       daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_SPECIALIZATION,
		},
		{
			Origin:     daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS,
			ClassId:    "class.bard",
			SubclassId: "subclass.wordsmith",
			Rank:       daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_FOUNDATION,
		},
	}

	if track, ok := findSubclassTrack(tracks, daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS); !ok || track.GetSubclassId() != "subclass.wordsmith" {
		t.Fatalf("multiclass track = %+v ok=%v", track, ok)
	}
	if _, ok := findSubclassTrack(tracks, daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_UNSPECIFIED); ok {
		t.Fatal("expected unspecified origin lookup to miss")
	}
	if rank, ok := findSubclassTrackRank(tracks, daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY); !ok || normalizeSubclassTrackRank(rank) != "specialization" {
		t.Fatalf("primary rank = %v ok=%v", rank, ok)
	}
	if got := normalizeSubclassTrackRank(daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_UNSPECIFIED); got != "" {
		t.Fatalf("normalized rank = %q, want empty", got)
	}
	if got := normalizeScenarioKey("  FEATURE.Stalwart-Unwavering  "); got != "feature.stalwart-unwavering" {
		t.Fatalf("normalized key = %q", got)
	}
}

func TestProfileHasActiveSubclassFeatureAcrossStages(t *testing.T) {
	profile := &daggerheartv1.DaggerheartProfile{
		ActiveSubclassFeatures: []*daggerheartv1.DaggerheartActiveSubclassTrackFeatures{
			{
				FoundationFeatures:     []*daggerheartv1.DaggerheartActiveSubclassFeature{{Id: "feature.stalwart-unwavering"}},
				SpecializationFeatures: []*daggerheartv1.DaggerheartActiveSubclassFeature{{Id: "feature.stalwart-unrelenting"}},
				MasteryFeatures:        []*daggerheartv1.DaggerheartActiveSubclassFeature{{Id: "feature.stalwart-undaunted"}},
			},
		},
	}

	if !profileHasActiveSubclassFeature(profile, "FEATURE.STALWART-UNRELENTING") {
		t.Fatal("expected specialization feature to be found")
	}
	if !profileHasActiveSubclassFeature(profile, "feature.stalwart-undaunted") {
		t.Fatal("expected mastery feature to be found")
	}
	if profileHasActiveSubclassFeature(profile, "feature.unknown") {
		t.Fatal("unexpected unknown feature match")
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
