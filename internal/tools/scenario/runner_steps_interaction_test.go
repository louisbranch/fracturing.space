package scenario

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRunStepInteractionPostUsesExplicitActor(t *testing.T) {
	fixture := testEnv()
	var gotParticipantID string
	var gotRequest *gamev1.SubmitScenePlayerPostRequest
	fixture.env.interactionClient = &fakeInteractionClient{
		submitPlayerPost: func(ctx context.Context, req *gamev1.SubmitScenePlayerPostRequest, _ ...grpc.CallOption) (*gamev1.SubmitScenePlayerPostResponse, error) {
			gotParticipantID = outgoingParticipantID(t, ctx)
			gotRequest = req
			return &gamev1.SubmitScenePlayerPostResponse{}, nil
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.scenes = map[string]string{"The Bridge": "scene-1"}
	state.activeSceneID = "scene-1"
	state.participants = map[string]string{"Guide": "participant-guide", "Rhea": "participant-rhea"}
	state.actors = map[string]string{"Aria": "character-aria"}

	step := Step{
		Kind: "interaction_post",
		Args: map[string]any{
			"as":         "Rhea",
			"summary":    "Aria braces the rope bridge for the others.",
			"characters": []any{"Aria"},
			"yield":      true,
		},
	}
	if err := runner.runStep(context.Background(), state, step); err != nil {
		t.Fatalf("runStep(interaction_post): %v", err)
	}
	if gotParticipantID != "participant-rhea" {
		t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "participant-rhea")
	}
	if gotRequest == nil {
		t.Fatal("expected SubmitScenePlayerPost request")
	}
	if gotRequest.GetSceneId() != "scene-1" {
		t.Fatalf("scene_id = %q, want %q", gotRequest.GetSceneId(), "scene-1")
	}
	if gotRequest.GetSummaryText() != "Aria braces the rope bridge for the others." {
		t.Fatalf("summary_text = %q", gotRequest.GetSummaryText())
	}
	if len(gotRequest.GetCharacterIds()) != 1 || gotRequest.GetCharacterIds()[0] != "character-aria" {
		t.Fatalf("character_ids = %v, want [character-aria]", gotRequest.GetCharacterIds())
	}
	if !gotRequest.GetYieldAfterPost() {
		t.Fatal("yield_after_post = false, want true")
	}
}

func TestRunStepInteractionSetGMAuthorityUsesExplicitActor(t *testing.T) {
	fixture := testEnv()
	var gotParticipantID string
	var gotRequest *gamev1.SetSessionGMAuthorityRequest
	fixture.env.interactionClient = &fakeInteractionClient{
		setGMAuthority: func(ctx context.Context, req *gamev1.SetSessionGMAuthorityRequest, _ ...grpc.CallOption) (*gamev1.SetSessionGMAuthorityResponse, error) {
			gotParticipantID = outgoingParticipantID(t, ctx)
			gotRequest = req
			return &gamev1.SetSessionGMAuthorityResponse{}, nil
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.participants = map[string]string{
		"Guide": "participant-guide",
		"Owner": "owner-1",
	}

	step := Step{
		Kind: "interaction_set_gm_authority",
		Args: map[string]any{
			"as":          "Owner",
			"participant": "Guide",
		},
	}
	if err := runner.runStep(context.Background(), state, step); err != nil {
		t.Fatalf("runStep(interaction_set_gm_authority): %v", err)
	}
	if gotParticipantID != "owner-1" {
		t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "owner-1")
	}
	if gotRequest == nil {
		t.Fatal("expected SetSessionGMAuthority request")
	}
	if gotRequest.GetCampaignId() != "campaign-1" {
		t.Fatalf("campaign_id = %q, want %q", gotRequest.GetCampaignId(), "campaign-1")
	}
	if gotRequest.GetParticipantId() != "participant-guide" {
		t.Fatalf("participant_id = %q, want %q", gotRequest.GetParticipantId(), "participant-guide")
	}
}

func TestRunStepInteractionSetActiveSceneUsesResolvedScene(t *testing.T) {
	fixture := testEnv()
	var gotParticipantID string
	var gotRequest *gamev1.SetActiveSceneRequest
	fixture.env.interactionClient = &fakeInteractionClient{
		setActiveScene: func(ctx context.Context, req *gamev1.SetActiveSceneRequest, _ ...grpc.CallOption) (*gamev1.SetActiveSceneResponse, error) {
			gotParticipantID = outgoingParticipantID(t, ctx)
			gotRequest = req
			return &gamev1.SetActiveSceneResponse{}, nil
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.participants = map[string]string{"Guide": "participant-guide"}
	state.scenes = map[string]string{"The Bridge": "scene-1"}

	step := Step{
		Kind: "interaction_set_active_scene",
		Args: map[string]any{
			"as":    "Guide",
			"scene": "The Bridge",
		},
	}
	if err := runner.runStep(context.Background(), state, step); err != nil {
		t.Fatalf("runStep(interaction_set_active_scene): %v", err)
	}
	if gotParticipantID != "participant-guide" {
		t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "participant-guide")
	}
	if gotRequest == nil {
		t.Fatal("expected SetActiveScene request")
	}
	if gotRequest.GetSceneId() != "scene-1" {
		t.Fatalf("scene_id = %q, want %q", gotRequest.GetSceneId(), "scene-1")
	}
	if state.activeSceneID != "scene-1" {
		t.Fatalf("activeSceneID = %q, want %q", state.activeSceneID, "scene-1")
	}
}

func TestRunStepInteractionStartPlayerPhaseUsesExplicitActor(t *testing.T) {
	fixture := testEnv()
	var gotParticipantID string
	var gotRequest *gamev1.StartScenePlayerPhaseRequest
	fixture.env.interactionClient = &fakeInteractionClient{
		startPlayerPhase: func(ctx context.Context, req *gamev1.StartScenePlayerPhaseRequest, _ ...grpc.CallOption) (*gamev1.StartScenePlayerPhaseResponse, error) {
			gotParticipantID = outgoingParticipantID(t, ctx)
			gotRequest = req
			return &gamev1.StartScenePlayerPhaseResponse{}, nil
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.activeSceneID = "scene-1"
	state.participants = map[string]string{"Guide": "participant-guide"}
	state.actors = map[string]string{
		"Aria":  "character-aria",
		"Corin": "character-corin",
	}

	step := Step{
		Kind: "interaction_start_player_phase",
		Args: map[string]any{
			"as":         "Guide",
			"frame_text": "What do you do?",
			"characters": []any{"Aria", "Corin"},
		},
	}
	if err := runner.runStep(context.Background(), state, step); err != nil {
		t.Fatalf("runStep(interaction_start_player_phase): %v", err)
	}
	if gotParticipantID != "participant-guide" {
		t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "participant-guide")
	}
	if gotRequest == nil {
		t.Fatal("expected StartScenePlayerPhase request")
	}
	if gotRequest.GetSceneId() != "scene-1" {
		t.Fatalf("scene_id = %q, want %q", gotRequest.GetSceneId(), "scene-1")
	}
	if gotRequest.GetFrameText() != "What do you do?" {
		t.Fatalf("frame_text = %q, want %q", gotRequest.GetFrameText(), "What do you do?")
	}
	if len(gotRequest.GetCharacterIds()) != 2 || gotRequest.GetCharacterIds()[0] != "character-aria" || gotRequest.GetCharacterIds()[1] != "character-corin" {
		t.Fatalf("character_ids = %v, want [character-aria character-corin]", gotRequest.GetCharacterIds())
	}
}

func TestRunStepInteractionReviewFlowUsesExplicitActor(t *testing.T) {
	fixture := testEnv()
	runner := quietRunner(fixture.env)

	t.Run("accept player phase", func(t *testing.T) {
		state := testState()
		state.activeSceneID = "scene-1"
		state.participants = map[string]string{"Guide": "participant-guide"}

		var gotParticipantID string
		var gotRequest *gamev1.AcceptScenePlayerPhaseRequest
		runner.env.interactionClient = &fakeInteractionClient{
			acceptPlayerPhase: func(ctx context.Context, req *gamev1.AcceptScenePlayerPhaseRequest, _ ...grpc.CallOption) (*gamev1.AcceptScenePlayerPhaseResponse, error) {
				gotParticipantID = outgoingParticipantID(t, ctx)
				gotRequest = req
				return &gamev1.AcceptScenePlayerPhaseResponse{}, nil
			},
		}

		if err := runner.runStep(context.Background(), state, Step{
			Kind: "interaction_accept_player_phase",
			Args: map[string]any{"as": "Guide"},
		}); err != nil {
			t.Fatalf("runStep(interaction_accept_player_phase): %v", err)
		}
		if gotParticipantID != "participant-guide" {
			t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "participant-guide")
		}
		if gotRequest == nil || gotRequest.GetCampaignId() != "campaign-1" || gotRequest.GetSceneId() != "scene-1" {
			t.Fatalf("accept request = %#v", gotRequest)
		}
	})

	t.Run("request revisions", func(t *testing.T) {
		state := testState()
		state.activeSceneID = "scene-1"
		state.participants = map[string]string{"Guide": "participant-guide", "Rhea": "participant-rhea"}
		state.actors = map[string]string{"Aria": "character-aria"}

		var gotParticipantID string
		var gotRequest *gamev1.RequestScenePlayerRevisionsRequest
		runner.env.interactionClient = &fakeInteractionClient{
			requestRevisions: func(ctx context.Context, req *gamev1.RequestScenePlayerRevisionsRequest, _ ...grpc.CallOption) (*gamev1.RequestScenePlayerRevisionsResponse, error) {
				gotParticipantID = outgoingParticipantID(t, ctx)
				gotRequest = req
				return &gamev1.RequestScenePlayerRevisionsResponse{}, nil
			},
		}

		if err := runner.runStep(context.Background(), state, Step{
			Kind: "interaction_request_revisions",
			Args: map[string]any{
				"as": "Guide",
				"revisions": []any{
					map[string]any{
						"participant": "Rhea",
						"reason":      "Clarify Aria's route through the water.",
						"characters":  []any{"Aria"},
					},
				},
			},
		}); err != nil {
			t.Fatalf("runStep(interaction_request_revisions): %v", err)
		}
		if gotParticipantID != "participant-guide" {
			t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "participant-guide")
		}
		if gotRequest == nil || gotRequest.GetCampaignId() != "campaign-1" || gotRequest.GetSceneId() != "scene-1" {
			t.Fatalf("revision request = %#v", gotRequest)
		}
		if len(gotRequest.GetRevisions()) != 1 {
			t.Fatalf("revisions = %#v, want 1 entry", gotRequest.GetRevisions())
		}
		revision := gotRequest.GetRevisions()[0]
		if revision.GetParticipantId() != "participant-rhea" || revision.GetReason() != "Clarify Aria's route through the water." {
			t.Fatalf("revision = %#v", revision)
		}
		if got := revision.GetCharacterIds(); len(got) != 1 || got[0] != "character-aria" {
			t.Fatalf("revision character_ids = %v, want [character-aria]", got)
		}
	})
}

func TestRunStepSystemActionRollUsesExplicitActor(t *testing.T) {
	fixture := testEnv()
	var gotParticipantID string
	fixture.env.daggerheartClient = &fakeDaggerheartClient{
		sessionActionRoll: func(ctx context.Context, req *daggerheartv1.SessionActionRollRequest, _ ...grpc.CallOption) (*daggerheartv1.SessionActionRollResponse, error) {
			gotParticipantID = outgoingParticipantID(t, ctx)
			return &daggerheartv1.SessionActionRollResponse{
				RollSeq:    41,
				HopeDie:    6,
				FearDie:    2,
				Total:      10,
				Difficulty: req.GetDifficulty(),
				Success:    true,
			}, nil
		},
	}
	fixture.env.eventClient = &fakeEventClient{}

	runner := quietRunner(fixture.env)
	state := testState()
	state.campaignSystem = commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	state.scenes = map[string]string{"The Bridge": "scene-1"}
	state.activeSceneID = "scene-1"
	state.participants = map[string]string{"Rhea": "participant-rhea"}
	state.actors = map[string]string{"Aria": "character-aria"}

	step := Step{
		System: "DAGGERHEART",
		Kind:   "action_roll",
		Args: map[string]any{
			"as":         "Rhea",
			"actor":      "Aria",
			"trait":      "agility",
			"difficulty": 12,
			"seed":       77,
		},
	}
	if err := runner.runStep(context.Background(), state, step); err != nil {
		t.Fatalf("runStep(action_roll): %v", err)
	}
	if gotParticipantID != "participant-rhea" {
		t.Fatalf("participant metadata = %q, want %q", gotParticipantID, "participant-rhea")
	}
	if state.lastRollSeq != 41 {
		t.Fatalf("lastRollSeq = %d, want %d", state.lastRollSeq, 41)
	}
	if outcome := state.rollOutcomes[41]; outcome.total != 10 || !outcome.success {
		t.Fatalf("roll outcome = %+v, want success total 10", outcome)
	}
}

func TestRunStepRejectsUnknownExplicitActor(t *testing.T) {
	runner := quietRunner(testEnv().env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{
		Kind: "interaction_ready_ooc",
		Args: map[string]any{"as": "Nobody"},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown participant") {
		t.Fatalf("expected unknown participant error, got %v", err)
	}
}

func TestRunStepExpectedErrorMatchesGRPCCodeAndMessage(t *testing.T) {
	fixture := testEnv()
	fixture.env.interactionClient = &fakeInteractionClient{
		resumeOOC: func(context.Context, *gamev1.ResumeFromOOCRequest, ...grpc.CallOption) (*gamev1.ResumeFromOOCResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "session is not paused for out-of-character discussion")
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	if err := runner.runStep(context.Background(), state, Step{
		Kind: "interaction_resume_ooc",
		Args: map[string]any{
			"expect_error": map[string]any{
				"code":     "FAILED_PRECONDITION",
				"contains": "not paused for out-of-character discussion",
			},
		},
	}); err != nil {
		t.Fatalf("runStep(interaction_resume_ooc with expect_error): %v", err)
	}
}

func TestRunStepExpectedErrorFailsWhenStepSucceeds(t *testing.T) {
	fixture := testEnv()
	fixture.env.interactionClient = &fakeInteractionClient{
		resumeOOC: func(context.Context, *gamev1.ResumeFromOOCRequest, ...grpc.CallOption) (*gamev1.ResumeFromOOCResponse, error) {
			return &gamev1.ResumeFromOOCResponse{}, nil
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{
		Kind: "interaction_resume_ooc",
		Args: map[string]any{
			"expect_error": map[string]any{"code": "FAILED_PRECONDITION"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "expected interaction_resume_ooc to fail with gRPC code FailedPrecondition") {
		t.Fatalf("expected success mismatch error, got %v", err)
	}
}

func TestRunStepExpectedErrorFailsWhenCodeMismatches(t *testing.T) {
	fixture := testEnv()
	fixture.env.interactionClient = &fakeInteractionClient{
		resumeOOC: func(context.Context, *gamev1.ResumeFromOOCRequest, ...grpc.CallOption) (*gamev1.ResumeFromOOCResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "not allowed")
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{
		Kind: "interaction_resume_ooc",
		Args: map[string]any{
			"expect_error": map[string]any{"code": "FAILED_PRECONDITION"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "expected gRPC code FailedPrecondition, got PermissionDenied") {
		t.Fatalf("expected code mismatch error, got %v", err)
	}
}

func TestRunStepExpectedErrorFailsForNonGRPCError(t *testing.T) {
	fixture := testEnv()
	fixture.env.interactionClient = &fakeInteractionClient{
		resumeOOC: func(context.Context, *gamev1.ResumeFromOOCRequest, ...grpc.CallOption) (*gamev1.ResumeFromOOCResponse, error) {
			return nil, errors.New("boom")
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{
		Kind: "interaction_resume_ooc",
		Args: map[string]any{
			"expect_error": map[string]any{"code": "FAILED_PRECONDITION"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "expected gRPC code FailedPrecondition but step returned non-gRPC error") {
		t.Fatalf("expected non-gRPC mismatch error, got %v", err)
	}
}

func TestRunStepExpectedErrorRejectsMissingCode(t *testing.T) {
	runner := quietRunner(testEnv().env)
	state := testState()
	err := runner.runStep(context.Background(), state, Step{
		Kind: "interaction_resume_ooc",
		Args: map[string]any{
			"expect_error": map[string]any{},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "expect_error code is required") {
		t.Fatalf("expected missing code error, got %v", err)
	}
}

func TestRunStepInteractionParticipantCallsUseExplicitActor(t *testing.T) {
	type interactionCapture struct {
		participantID string
		campaignID    string
		sceneID       string
		reason        string
		body          string
	}

	fixture := testEnv()
	runner := quietRunner(fixture.env)

	cases := []struct {
		name   string
		step   Step
		setup  func(*scenarioState)
		client func(*interactionCapture) *fakeInteractionClient
		assert func(*testing.T, *interactionCapture, *scenarioState)
	}{
		{
			name: "yield",
			step: Step{Kind: "interaction_yield", Args: map[string]any{"as": "Rhea"}},
			setup: func(state *scenarioState) {
				state.activeSceneID = "scene-1"
				state.participants = map[string]string{"Rhea": "participant-rhea"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					yieldPlayerPhase: func(ctx context.Context, req *gamev1.YieldScenePlayerPhaseRequest, _ ...grpc.CallOption) (*gamev1.YieldScenePlayerPhaseResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						capture.sceneID = req.GetSceneId()
						return &gamev1.YieldScenePlayerPhaseResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-rhea" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-rhea")
				}
				if capture.campaignID != "campaign-1" || capture.sceneID != "scene-1" {
					t.Fatalf("yield request = campaign %q scene %q", capture.campaignID, capture.sceneID)
				}
			},
		},
		{
			name: "unyield",
			step: Step{Kind: "interaction_unyield", Args: map[string]any{"as": "Rhea"}},
			setup: func(state *scenarioState) {
				state.activeSceneID = "scene-1"
				state.participants = map[string]string{"Rhea": "participant-rhea"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					unyieldPlayerPhase: func(ctx context.Context, req *gamev1.UnyieldScenePlayerPhaseRequest, _ ...grpc.CallOption) (*gamev1.UnyieldScenePlayerPhaseResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						capture.sceneID = req.GetSceneId()
						return &gamev1.UnyieldScenePlayerPhaseResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-rhea" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-rhea")
				}
				if capture.campaignID != "campaign-1" || capture.sceneID != "scene-1" {
					t.Fatalf("unyield request = campaign %q scene %q", capture.campaignID, capture.sceneID)
				}
			},
		},
		{
			name: "end_player_phase",
			step: Step{Kind: "interaction_end_player_phase", Args: map[string]any{"as": "Guide", "reason": "gm_interrupted"}},
			setup: func(state *scenarioState) {
				state.activeSceneID = "scene-1"
				state.participants = map[string]string{"Guide": "participant-guide"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					endPlayerPhase: func(ctx context.Context, req *gamev1.EndScenePlayerPhaseRequest, _ ...grpc.CallOption) (*gamev1.EndScenePlayerPhaseResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						capture.sceneID = req.GetSceneId()
						capture.reason = req.GetReason()
						return &gamev1.EndScenePlayerPhaseResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-guide" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-guide")
				}
				if capture.reason != "gm_interrupted" {
					t.Fatalf("reason = %q, want %q", capture.reason, "gm_interrupted")
				}
			},
		},
		{
			name: "accept_player_phase",
			step: Step{Kind: "interaction_accept_player_phase", Args: map[string]any{"as": "Guide"}},
			setup: func(state *scenarioState) {
				state.activeSceneID = "scene-1"
				state.participants = map[string]string{"Guide": "participant-guide"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					acceptPlayerPhase: func(ctx context.Context, req *gamev1.AcceptScenePlayerPhaseRequest, _ ...grpc.CallOption) (*gamev1.AcceptScenePlayerPhaseResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						capture.sceneID = req.GetSceneId()
						return &gamev1.AcceptScenePlayerPhaseResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-guide" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-guide")
				}
				if capture.campaignID != "campaign-1" || capture.sceneID != "scene-1" {
					t.Fatalf("accept request = campaign %q scene %q", capture.campaignID, capture.sceneID)
				}
			},
		},
		{
			name: "pause_ooc",
			step: Step{Kind: "interaction_pause_ooc", Args: map[string]any{"as": "Guide", "reason": "clarify the ruling"}},
			setup: func(state *scenarioState) {
				state.participants = map[string]string{"Guide": "participant-guide"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					pauseOOC: func(ctx context.Context, req *gamev1.PauseSessionForOOCRequest, _ ...grpc.CallOption) (*gamev1.PauseSessionForOOCResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						capture.reason = req.GetReason()
						return &gamev1.PauseSessionForOOCResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-guide" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-guide")
				}
				if capture.reason != "clarify the ruling" {
					t.Fatalf("reason = %q, want %q", capture.reason, "clarify the ruling")
				}
			},
		},
		{
			name: "post_ooc",
			step: Step{Kind: "interaction_post_ooc", Args: map[string]any{"as": "Rhea", "body": "Question?"}},
			setup: func(state *scenarioState) {
				state.participants = map[string]string{"Rhea": "participant-rhea"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					postOOC: func(ctx context.Context, req *gamev1.PostSessionOOCRequest, _ ...grpc.CallOption) (*gamev1.PostSessionOOCResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						capture.body = req.GetBody()
						return &gamev1.PostSessionOOCResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-rhea" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-rhea")
				}
				if capture.body != "Question?" {
					t.Fatalf("body = %q, want %q", capture.body, "Question?")
				}
			},
		},
		{
			name: "ready_ooc",
			step: Step{Kind: "interaction_ready_ooc", Args: map[string]any{"as": "Rhea"}},
			setup: func(state *scenarioState) {
				state.participants = map[string]string{"Rhea": "participant-rhea"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					markOOCReady: func(ctx context.Context, req *gamev1.MarkOOCReadyToResumeRequest, _ ...grpc.CallOption) (*gamev1.MarkOOCReadyToResumeResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						return &gamev1.MarkOOCReadyToResumeResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-rhea" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-rhea")
				}
				if capture.campaignID != "campaign-1" {
					t.Fatalf("campaign_id = %q, want %q", capture.campaignID, "campaign-1")
				}
			},
		},
		{
			name: "clear_ready_ooc",
			step: Step{Kind: "interaction_clear_ready_ooc", Args: map[string]any{"as": "Rhea"}},
			setup: func(state *scenarioState) {
				state.participants = map[string]string{"Rhea": "participant-rhea"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					clearOOCReady: func(ctx context.Context, req *gamev1.ClearOOCReadyToResumeRequest, _ ...grpc.CallOption) (*gamev1.ClearOOCReadyToResumeResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						return &gamev1.ClearOOCReadyToResumeResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-rhea" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-rhea")
				}
				if capture.campaignID != "campaign-1" {
					t.Fatalf("campaign_id = %q, want %q", capture.campaignID, "campaign-1")
				}
			},
		},
		{
			name: "resume_ooc",
			step: Step{Kind: "interaction_resume_ooc", Args: map[string]any{"as": "Guide"}},
			setup: func(state *scenarioState) {
				state.participants = map[string]string{"Guide": "participant-guide"}
			},
			client: func(capture *interactionCapture) *fakeInteractionClient {
				return &fakeInteractionClient{
					resumeOOC: func(ctx context.Context, req *gamev1.ResumeFromOOCRequest, _ ...grpc.CallOption) (*gamev1.ResumeFromOOCResponse, error) {
						capture.participantID = outgoingParticipantID(t, ctx)
						capture.campaignID = req.GetCampaignId()
						return &gamev1.ResumeFromOOCResponse{}, nil
					},
				}
			},
			assert: func(t *testing.T, capture *interactionCapture, _ *scenarioState) {
				t.Helper()
				if capture.participantID != "participant-guide" {
					t.Fatalf("participant metadata = %q, want %q", capture.participantID, "participant-guide")
				}
				if capture.campaignID != "campaign-1" {
					t.Fatalf("campaign_id = %q, want %q", capture.campaignID, "campaign-1")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := testState()
			tc.setup(state)
			capture := &interactionCapture{}
			runner.env.interactionClient = tc.client(capture)

			if err := runner.runStep(context.Background(), state, tc.step); err != nil {
				t.Fatalf("runStep(%s): %v", tc.step.Kind, err)
			}
			tc.assert(t, capture, state)
		})
	}
}

func TestResolveInteractionSceneID(t *testing.T) {
	state := testState()
	state.activeSceneID = "scene-active"
	state.scenes = map[string]string{
		"The Bridge": "scene-1",
		"The Keep":   "scene-2",
	}

	tests := []struct {
		name            string
		args            map[string]any
		requireExplicit bool
		state           *scenarioState
		want            string
		wantErr         string
	}{
		{
			name:            "uses active scene when explicit scene is optional",
			args:            map[string]any{},
			requireExplicit: false,
			state:           state,
			want:            "scene-active",
		},
		{
			name:            "resolves explicit scene case-insensitively",
			args:            map[string]any{"scene": "the bridge"},
			requireExplicit: true,
			state:           state,
			want:            "scene-1",
		},
		{
			name:            "requires explicit scene when none active",
			args:            map[string]any{},
			requireExplicit: true,
			state:           testState(),
			wantErr:         "interaction scene is required",
		},
		{
			name:            "requires active scene when scene omitted",
			args:            map[string]any{},
			requireExplicit: false,
			state: func() *scenarioState {
				s := testState()
				s.scenes = map[string]string{"The Bridge": "scene-1"}
				return s
			}(),
			wantErr: "interaction step requires an active scene",
		},
		{
			name:            "rejects unknown scene",
			args:            map[string]any{"scene": "Unknown"},
			requireExplicit: true,
			state:           state,
			wantErr:         `unknown scene "Unknown"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveInteractionSceneID(tc.state, tc.args, tc.requireExplicit)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveInteractionSceneID: %v", err)
			}
			if got != tc.want {
				t.Fatalf("scene_id = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseExpectedInteractionSlots(t *testing.T) {
	state := testState()
	state.participants = map[string]string{"Rhea": "participant-rhea"}
	state.actors = map[string]string{"Aria": "character-aria"}

	tests := []struct {
		name    string
		args    map[string]any
		want    []expectedInteractionSlot
		wantErr string
	}{
		{
			name: "parses expected slots",
			args: map[string]any{
				"slots": []any{
					map[string]any{
						"participant":       "Rhea",
						"summary":           "Aria rushes forward.",
						"characters":        []any{"Aria"},
						"yielded":           true,
						"review_status":     "changes_requested",
						"review_reason":     "Clarify the route.",
						"review_characters": []any{"Aria"},
					},
				},
			},
			want: []expectedInteractionSlot{
				{
					participantID: "participant-rhea",
					summaryText:   "Aria rushes forward.",
					characterIDs:  []string{"character-aria"},
					yielded:       true,
					reviewStatus:  "CHANGES_REQUESTED",
					reviewReason:  "Clarify the route.",
					reviewChars:   []string{"character-aria"},
				},
			},
		},
		{
			name: "accepts empty lua table",
			args: map[string]any{"slots": map[string]any{}},
			want: []expectedInteractionSlot{},
		},
		{
			name:    "rejects non list",
			args:    map[string]any{"slots": "bad"},
			wantErr: "slots must be a list",
		},
		{
			name:    "rejects non table entry",
			args:    map[string]any{"slots": []any{"bad"}},
			wantErr: "slots entries must be tables",
		},
		{
			name:    "requires participant",
			args:    map[string]any{"slots": []any{map[string]any{"summary": "Aria rushes forward."}}},
			wantErr: "slots participant is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseExpectedInteractionSlots(state, tc.args, "slots")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseExpectedInteractionSlots: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("slots len = %d, want %d", len(got), len(tc.want))
			}
			for index := range got {
				if !equalExpectedInteractionSlot(got[index], tc.want[index]) {
					t.Fatalf("slot[%d] = %+v, want %+v", index, got[index], tc.want[index])
				}
			}
		})
	}
}

func TestParseScenePlayerRevisionRequests(t *testing.T) {
	state := testState()
	state.participants = map[string]string{"Rhea": "participant-rhea"}
	state.actors = map[string]string{"Aria": "character-aria"}

	tests := []struct {
		name    string
		args    map[string]any
		want    []*gamev1.ScenePlayerRevisionRequest
		wantErr string
	}{
		{
			name: "parses revisions",
			args: map[string]any{
				"revisions": []any{
					map[string]any{"participant": "Rhea", "reason": "Clarify the route.", "characters": []any{"Aria"}},
				},
			},
			want: []*gamev1.ScenePlayerRevisionRequest{{
				ParticipantId: "participant-rhea",
				Reason:        "Clarify the route.",
				CharacterIds:  []string{"character-aria"},
			}},
		},
		{
			name: "accepts empty table",
			args: map[string]any{"revisions": map[string]any{}},
			want: []*gamev1.ScenePlayerRevisionRequest{},
		},
		{
			name:    "rejects non list",
			args:    map[string]any{"revisions": "bad"},
			wantErr: "revisions must be a list",
		},
		{
			name:    "requires participant",
			args:    map[string]any{"revisions": []any{map[string]any{"reason": "Clarify"}}},
			wantErr: "revisions participant is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseScenePlayerRevisionRequests(state, tc.args, "revisions")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseScenePlayerRevisionRequests: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("revisions len = %d, want %d", len(got), len(tc.want))
			}
			for index := range got {
				if got[index].GetParticipantId() != tc.want[index].GetParticipantId() || got[index].GetReason() != tc.want[index].GetReason() || !equalStrings(got[index].GetCharacterIds(), tc.want[index].GetCharacterIds()) {
					t.Fatalf("revision[%d] = %#v, want %#v", index, got[index], tc.want[index])
				}
			}
		})
	}
}

func TestParseExpectedOOCPosts(t *testing.T) {
	state := testState()
	state.participants = map[string]string{"Guide": "participant-guide"}

	tests := []struct {
		name    string
		args    map[string]any
		want    []expectedOOCPost
		wantErr string
	}{
		{
			name: "parses expected ooc posts",
			args: map[string]any{
				"ooc_posts": []any{
					map[string]any{"participant": "Guide", "body": "The ward reacts to touch."},
				},
			},
			want: []expectedOOCPost{
				{participantID: "participant-guide", body: "The ward reacts to touch."},
			},
		},
		{
			name: "accepts empty lua table",
			args: map[string]any{"ooc_posts": map[string]any{}},
			want: []expectedOOCPost{},
		},
		{
			name:    "rejects non list",
			args:    map[string]any{"ooc_posts": "bad"},
			wantErr: "ooc_posts must be a list",
		},
		{
			name:    "rejects non table entry",
			args:    map[string]any{"ooc_posts": []any{"bad"}},
			wantErr: "ooc_posts entries must be tables",
		},
		{
			name:    "requires participant",
			args:    map[string]any{"ooc_posts": []any{map[string]any{"body": "Question?"}}},
			wantErr: "ooc_posts participant is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseExpectedOOCPosts(state, tc.args, "ooc_posts")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseExpectedOOCPosts: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("posts len = %d, want %d", len(got), len(tc.want))
			}
			for index := range got {
				if !equalExpectedOOCPost(got[index], tc.want[index]) {
					t.Fatalf("post[%d] = %+v, want %+v", index, got[index], tc.want[index])
				}
			}
		})
	}
}

func TestRunInteractionExpectStepMatchesAuthoritativeState(t *testing.T) {
	fixture := testEnv()
	fixture.env.interactionClient = &fakeInteractionClient{
		getState: func(context.Context, *gamev1.GetInteractionStateRequest, ...grpc.CallOption) (*gamev1.GetInteractionStateResponse, error) {
			return &gamev1.GetInteractionStateResponse{
				State: &gamev1.InteractionState{
					ActiveSession: &gamev1.InteractionSession{Name: "Crossing"},
					ActiveScene:   &gamev1.InteractionScene{Name: "The Bridge"},
					PlayerPhase: &gamev1.ScenePlayerPhase{
						Status:               gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW,
						FrameText:            "Rain lashes the ropes. What do you do next?",
						ActingCharacterIds:   []string{"character-aria", "character-corin"},
						ActingParticipantIds: []string{"participant-bryn", "participant-rhea"},
						Slots: []*gamev1.ScenePlayerSlot{
							{ParticipantId: "participant-bryn", SummaryText: "Corin steadies the lantern.", CharacterIds: []string{"character-corin"}, ReviewStatus: gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED},
							{ParticipantId: "participant-rhea", SummaryText: "Aria tests the bridge cables.", CharacterIds: []string{"character-aria"}, Yielded: true, ReviewStatus: gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED, ReviewReason: "Clarify the route.", ReviewCharacterIds: []string{"character-aria"}},
						},
					},
					Ooc: &gamev1.OOCState{
						Open:                        true,
						ReadyToResumeParticipantIds: []string{"participant-rhea"},
						Posts: []*gamev1.OOCPost{
							{ParticipantId: "participant-guide", Body: "The ward reacts to touch, not sight."},
						},
					},
					GmAuthorityParticipantId: "participant-guide",
				},
			}, nil
		},
	}

	runner := quietRunner(fixture.env)
	state := testState()
	state.scenes = map[string]string{"The Bridge": "scene-1"}
	state.participants = map[string]string{"Guide": "participant-guide", "Rhea": "participant-rhea", "Bryn": "participant-bryn"}
	state.actors = map[string]string{"Aria": "character-aria", "Corin": "character-corin"}

	step := Step{
		Kind: "interaction_expect",
		Args: map[string]any{
			"session":             "Crossing",
			"active_scene":        "The Bridge",
			"phase_status":        "GM_REVIEW",
			"frame_text":          "Rain lashes the ropes. What do you do next?",
			"acting_characters":   []any{"Aria", "Corin"},
			"acting_participants": []any{"Rhea", "Bryn"},
			"gm_authority":        "Guide",
			"ooc_open":            true,
			"ooc_ready":           []any{"Rhea"},
			"slots": []any{
				map[string]any{"participant": "Rhea", "summary": "Aria tests the bridge cables.", "characters": []any{"Aria"}, "yielded": true, "review_status": "changes_requested", "review_reason": "Clarify the route.", "review_characters": []any{"Aria"}},
				map[string]any{"participant": "Bryn", "summary": "Corin steadies the lantern.", "characters": []any{"Corin"}, "review_status": "accepted"},
			},
			"ooc_posts": []any{
				map[string]any{"participant": "Guide", "body": "The ward reacts to touch, not sight."},
			},
		},
	}
	if err := runner.runInteractionExpectStep(context.Background(), state, step); err != nil {
		t.Fatalf("runInteractionExpectStep: %v", err)
	}
}

func TestRunInteractionExpectStepValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		response     *gamev1.GetInteractionStateResponse
		args         map[string]any
		participants map[string]string
		actors       map[string]string
		wantErr      string
	}{
		{
			name:         "empty state",
			response:     &gamev1.GetInteractionStateResponse{},
			args:         map[string]any{"phase_status": "GM"},
			wantErr:      "interaction_expect returned empty state",
			participants: map[string]string{},
			actors:       map[string]string{},
		},
		{
			name: "slots must be a list",
			response: &gamev1.GetInteractionStateResponse{
				State: &gamev1.InteractionState{
					PlayerPhase: &gamev1.ScenePlayerPhase{Status: gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM},
					Ooc:         &gamev1.OOCState{},
				},
			},
			args:         map[string]any{"slots": "bad"},
			wantErr:      "slots must be a list",
			participants: map[string]string{},
			actors:       map[string]string{},
		},
		{
			name: "ooc posts must be a list",
			response: &gamev1.GetInteractionStateResponse{
				State: &gamev1.InteractionState{
					PlayerPhase: &gamev1.ScenePlayerPhase{Status: gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM},
					Ooc:         &gamev1.OOCState{},
				},
			},
			args:         map[string]any{"ooc_posts": "bad"},
			wantErr:      "ooc_posts must be a list",
			participants: map[string]string{},
			actors:       map[string]string{},
		},
		{
			name: "gm authority mismatch",
			response: &gamev1.GetInteractionStateResponse{
				State: &gamev1.InteractionState{
					PlayerPhase:              &gamev1.ScenePlayerPhase{Status: gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM},
					Ooc:                      &gamev1.OOCState{},
					GmAuthorityParticipantId: "participant-guide",
				},
			},
			args:         map[string]any{"gm_authority": "Rhea"},
			wantErr:      `interaction gm_authority = "participant-guide", want "participant-rhea"`,
			participants: map[string]string{"Rhea": "participant-rhea"},
			actors:       map[string]string{},
		},
		{
			name: "ooc posts mismatch",
			response: &gamev1.GetInteractionStateResponse{
				State: &gamev1.InteractionState{
					PlayerPhase: &gamev1.ScenePlayerPhase{Status: gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM},
					Ooc: &gamev1.OOCState{
						Posts: []*gamev1.OOCPost{
							{ParticipantId: "participant-guide", Body: "Actual"},
						},
					},
				},
			},
			args:         map[string]any{"ooc_posts": []any{map[string]any{"participant": "Guide", "body": "Wanted"}}},
			wantErr:      "interaction ooc_posts =",
			participants: map[string]string{"Guide": "participant-guide"},
			actors:       map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := testEnv()
			fixture.env.interactionClient = &fakeInteractionClient{
				getState: func(context.Context, *gamev1.GetInteractionStateRequest, ...grpc.CallOption) (*gamev1.GetInteractionStateResponse, error) {
					return tc.response, nil
				},
			}
			runner := quietRunner(fixture.env)
			state := testState()
			state.participants = tc.participants
			state.actors = tc.actors

			err := runner.runInteractionExpectStep(context.Background(), state, Step{Kind: "interaction_expect", Args: tc.args})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestNormalizeScenePhaseStatusDoesNotCollapseUnknownValuesToGM(t *testing.T) {
	t.Parallel()

	if got := normalizeScenePhaseStatus(gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_UNSPECIFIED); got != "UNSPECIFIED" {
		t.Fatalf("normalizeScenePhaseStatus(UNSPECIFIED) = %q, want UNSPECIFIED", got)
	}
	if got := normalizeScenePhaseStatus(gamev1.ScenePhaseStatus(99)); got != "99" {
		t.Fatalf("normalizeScenePhaseStatus(99) = %q, want 99", got)
	}
	if got := normalizeScenePhaseStatusString("SCENE_PHASE_STATUS_UNSPECIFIED"); got != "UNSPECIFIED" {
		t.Fatalf("normalizeScenePhaseStatusString(UNSPECIFIED) = %q, want UNSPECIFIED", got)
	}
	if got := normalizeScenePhaseStatusString("future_status"); got != "FUTURE_STATUS" {
		t.Fatalf("normalizeScenePhaseStatusString(future_status) = %q, want FUTURE_STATUS", got)
	}
}

func outgoingParticipantID(t *testing.T, ctx context.Context) string {
	t.Helper()
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	values := md.Get(grpcmeta.ParticipantIDHeader)
	if len(values) == 0 {
		t.Fatal("expected participant metadata")
	}
	return values[len(values)-1]
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
