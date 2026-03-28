package gametools

import (
	"context"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerhearttools"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type interactionClientStub struct {
	response *statev1.GetInteractionStateResponse
	err      error
}

func (stub interactionClientStub) GetInteractionState(context.Context, *statev1.GetInteractionStateRequest, ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
	return stub.response, stub.err
}

func (interactionClientStub) ActivateScene(context.Context, *statev1.ActivateSceneRequest, ...grpc.CallOption) (*statev1.ActivateSceneResponse, error) {
	return nil, nil
}

func (interactionClientStub) OpenScenePlayerPhase(context.Context, *statev1.OpenScenePlayerPhaseRequest, ...grpc.CallOption) (*statev1.OpenScenePlayerPhaseResponse, error) {
	return nil, nil
}

func (interactionClientStub) SubmitScenePlayerAction(context.Context, *statev1.SubmitScenePlayerActionRequest, ...grpc.CallOption) (*statev1.SubmitScenePlayerActionResponse, error) {
	return nil, nil
}

func (interactionClientStub) YieldScenePlayerPhase(context.Context, *statev1.YieldScenePlayerPhaseRequest, ...grpc.CallOption) (*statev1.YieldScenePlayerPhaseResponse, error) {
	return nil, nil
}

func (interactionClientStub) WithdrawScenePlayerYield(context.Context, *statev1.WithdrawScenePlayerYieldRequest, ...grpc.CallOption) (*statev1.WithdrawScenePlayerYieldResponse, error) {
	return nil, nil
}

func (interactionClientStub) InterruptScenePlayerPhase(context.Context, *statev1.InterruptScenePlayerPhaseRequest, ...grpc.CallOption) (*statev1.InterruptScenePlayerPhaseResponse, error) {
	return nil, nil
}

func (interactionClientStub) RecordSceneGMInteraction(context.Context, *statev1.RecordSceneGMInteractionRequest, ...grpc.CallOption) (*statev1.RecordSceneGMInteractionResponse, error) {
	return nil, nil
}

func (interactionClientStub) ResolveScenePlayerReview(context.Context, *statev1.ResolveScenePlayerReviewRequest, ...grpc.CallOption) (*statev1.ResolveScenePlayerReviewResponse, error) {
	return nil, nil
}

func (interactionClientStub) OpenSessionOOC(context.Context, *statev1.OpenSessionOOCRequest, ...grpc.CallOption) (*statev1.OpenSessionOOCResponse, error) {
	return nil, nil
}

func (interactionClientStub) PostSessionOOC(context.Context, *statev1.PostSessionOOCRequest, ...grpc.CallOption) (*statev1.PostSessionOOCResponse, error) {
	return nil, nil
}

func (interactionClientStub) MarkOOCReadyToResume(context.Context, *statev1.MarkOOCReadyToResumeRequest, ...grpc.CallOption) (*statev1.MarkOOCReadyToResumeResponse, error) {
	return nil, nil
}

func (interactionClientStub) ClearOOCReadyToResume(context.Context, *statev1.ClearOOCReadyToResumeRequest, ...grpc.CallOption) (*statev1.ClearOOCReadyToResumeResponse, error) {
	return nil, nil
}

func (interactionClientStub) ResolveSessionOOC(context.Context, *statev1.ResolveSessionOOCRequest, ...grpc.CallOption) (*statev1.ResolveSessionOOCResponse, error) {
	return nil, nil
}

func (interactionClientStub) SetSessionGMAuthority(context.Context, *statev1.SetSessionGMAuthorityRequest, ...grpc.CallOption) (*statev1.SetSessionGMAuthorityResponse, error) {
	return nil, nil
}

func (interactionClientStub) SetSessionCharacterController(context.Context, *statev1.SetSessionCharacterControllerRequest, ...grpc.CallOption) (*statev1.SetSessionCharacterControllerResponse, error) {
	return nil, nil
}

func (interactionClientStub) RetryAIGMTurn(context.Context, *statev1.RetryAIGMTurnRequest, ...grpc.CallOption) (*statev1.RetryAIGMTurnResponse, error) {
	return nil, nil
}

func TestOutgoingContextAddsAuthorityMetadataAndDeadline(t *testing.T) {
	ctx, cancel := outgoingContext(context.Background(), SessionContext{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "part-1",
	})
	defer cancel()

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("outgoing metadata missing")
	}
	if got := md.Get(grpcmeta.CampaignIDHeader); len(got) != 1 || got[0] != "camp-1" {
		t.Fatalf("campaign metadata = %#v", got)
	}
	if got := md.Get(grpcmeta.SessionIDHeader); len(got) != 1 || got[0] != "sess-1" {
		t.Fatalf("session metadata = %#v", got)
	}
	if got := md.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
		t.Fatalf("participant metadata = %#v", got)
	}
	if got := md.Get(grpcmeta.RequestIDHeader); len(got) != 1 || strings.TrimSpace(got[0]) == "" {
		t.Fatalf("request metadata = %#v", got)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("deadline missing")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > callTimeout {
		t.Fatalf("remaining deadline = %v", remaining)
	}
}

func TestOutgoingContextPreservesExistingDeadline(t *testing.T) {
	base, stop := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer stop()

	ctx, cancel := outgoingContext(base, SessionContext{})
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("deadline missing")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 5*time.Second {
		t.Fatalf("remaining deadline = %v", remaining)
	}
}

func TestDaggerheartRuntimeAdapterWrapsSessionHelpers(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	executor := wrapDaggerheartExecutor(func(runtime daggerhearttools.Runtime, ctx context.Context, _ []byte) (orchestration.ToolResult, error) {
		callCtx, cancel := runtime.CallContext(ctx)
		defer cancel()
		if runtime.ResolveCampaignID("") != "camp-1" || runtime.ResolveSessionID("") != "sess-1" {
			t.Fatalf("resolved IDs = (%q, %q)", runtime.ResolveCampaignID(""), runtime.ResolveSessionID(""))
		}
		if _, ok := callCtx.Deadline(); !ok {
			t.Fatal("wrapped call context missing deadline")
		}
		return orchestration.ToolResult{Output: "wrapped"}, nil
	})

	result, err := executor(session, context.Background(), nil)
	if err != nil {
		t.Fatalf("wrapped executor error = %v", err)
	}
	if result.Output != "wrapped" {
		t.Fatalf("output = %q, want wrapped", result.Output)
	}
}

func TestResolveSceneIDHelpers(t *testing.T) {
	session := NewDirectSession(Clients{
		Interaction: interactionClientStub{
			response: &statev1.GetInteractionStateResponse{
				State: &statev1.InteractionState{
					ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
				},
			},
		},
	}, SessionContext{})

	sceneID, err := session.resolveSceneID(context.Background(), "camp-1", "")
	if err != nil {
		t.Fatalf("resolveSceneID() error = %v", err)
	}
	if sceneID != "scene-1" {
		t.Fatalf("scene_id = %q, want scene-1", sceneID)
	}

	if explicit, err := session.resolveSceneIDFromState(nil, " scene-2 "); err != nil || explicit != "scene-2" {
		t.Fatalf("resolveSceneIDFromState(explicit) = (%q, %v)", explicit, err)
	}

	if _, err := session.resolveSceneIDFromState(nil, ""); err == nil || err.Error() != "interaction state is required" {
		t.Fatalf("resolveSceneIDFromState(nil) error = %v", err)
	}
	if _, err := session.resolveSceneIDFromState(&statev1.InteractionState{}, ""); err == nil || err.Error() != "scene_id is required when no active scene is set" {
		t.Fatalf("resolveSceneIDFromState(no active scene) error = %v", err)
	}
}

func TestGetInteractionStateWrapsFailuresAndMissingResponse(t *testing.T) {
	session := NewDirectSession(Clients{
		Interaction: interactionClientStub{err: context.DeadlineExceeded},
	}, SessionContext{})

	if _, err := session.getInteractionState(context.Background(), "camp-1"); err == nil || !strings.Contains(err.Error(), "get interaction state failed") {
		t.Fatalf("getInteractionState() error = %v", err)
	}

	session = NewDirectSession(Clients{
		Interaction: interactionClientStub{response: &statev1.GetInteractionStateResponse{}},
	}, SessionContext{})
	if _, err := session.getInteractionState(context.Background(), "camp-1"); err == nil || err.Error() != "get interaction state response is missing" {
		t.Fatalf("getInteractionState() missing response error = %v", err)
	}
}

func TestInteractionGMInteractionHelpers(t *testing.T) {
	input := singleBeatGMInteractionInput("Arrival", statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION, "  The caravan arrives.  ", "char-1")
	if input.Title != "Arrival" || len(input.Beats) != 1 || input.Beats[0].Text != "The caravan arrives." {
		t.Fatalf("singleBeatGMInteractionInput() = %#v", input)
	}

	if got, err := parseGMInteractionBeatType("prompt"); err != nil || got != statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT {
		t.Fatalf("parseGMInteractionBeatType(prompt) = (%v, %v)", got, err)
	}
	if _, err := parseGMInteractionBeatType("unknown"); err == nil || !strings.Contains(err.Error(), "unsupported beat type") {
		t.Fatalf("parseGMInteractionBeatType(unknown) error = %v", err)
	}
	if got := gmInteractionBeatTypeToString(statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE); got != "consequence" {
		t.Fatalf("gmInteractionBeatTypeToString() = %q", got)
	}
	if got := gmInteractionBeatTypeToString(statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_UNSPECIFIED); got != "" {
		t.Fatalf("gmInteractionBeatTypeToString(unspecified) = %q", got)
	}

	result := interactionGMInteractionFromProto(&statev1.GMInteraction{
		InteractionId: "int-1",
		SceneId:       "scene-1",
		PhaseId:       "phase-1",
		ParticipantId: "gm-1",
		Title:         "Arrival",
		CharacterIds:  []string{"char-1"},
		CreatedAt:     timestamppb.New(time.Date(2026, time.March, 27, 5, 0, 0, 0, time.UTC)),
		Beats: []*statev1.GMInteractionBeat{{
			BeatId: "beat-1",
			Type:   statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION,
			Text:   "The caravan arrives.",
		}},
	})
	if result == nil || result.CreatedAt != "2026-03-27T05:00:00Z" || len(result.Beats) != 1 {
		t.Fatalf("interactionGMInteractionFromProto() = %#v", result)
	}
	if result.Beats[0].Type != "fiction" {
		t.Fatalf("beat type = %q, want fiction", result.Beats[0].Type)
	}
}
