package interactiontransport

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInteractionServiceRejectsNilRequests(t *testing.T) {
	t.Parallel()

	svc := &InteractionService{}
	ctx := context.Background()

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "get interaction state", run: func() error { _, err := svc.GetInteractionState(ctx, nil); return err }},
		{name: "set session gm authority", run: func() error { _, err := svc.SetSessionGMAuthority(ctx, nil); return err }},
		{name: "retry ai gm turn", run: func() error { _, err := svc.RetryAIGMTurn(ctx, nil); return err }},
		{name: "set active scene", run: func() error { _, err := svc.ActivateScene(ctx, nil); return err }},
		{name: "start scene player phase", run: func() error { _, err := svc.OpenScenePlayerPhase(ctx, nil); return err }},
		{name: "submit scene player post", run: func() error { _, err := svc.SubmitScenePlayerAction(ctx, nil); return err }},
		{name: "yield scene player phase", run: func() error { _, err := svc.YieldScenePlayerPhase(ctx, nil); return err }},
		{name: "unyield scene player phase", run: func() error { _, err := svc.WithdrawScenePlayerYield(ctx, nil); return err }},
		{name: "end scene player phase", run: func() error { _, err := svc.InterruptScenePlayerPhase(ctx, nil); return err }},
		{name: "commit scene gm interaction", run: func() error { _, err := svc.RecordSceneGMInteraction(ctx, nil); return err }},
		{name: "resolve scene player phase review", run: func() error { _, err := svc.ResolveScenePlayerReview(ctx, nil); return err }},
		{name: "pause session for ooc", run: func() error { _, err := svc.OpenSessionOOC(ctx, nil); return err }},
		{name: "post session ooc", run: func() error { _, err := svc.PostSessionOOC(ctx, nil); return err }},
		{name: "mark ooc ready", run: func() error { _, err := svc.MarkOOCReadyToResume(ctx, nil); return err }},
		{name: "clear ooc ready", run: func() error { _, err := svc.ClearOOCReadyToResume(ctx, nil); return err }},
		{name: "resolve session ooc", run: func() error { _, err := svc.ResolveSessionOOC(ctx, nil); return err }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
			}
		})
	}
}

func TestInteractionServiceRejectsMissingCampaignID(t *testing.T) {
	t.Parallel()

	svc := &InteractionService{}
	ctx := context.Background()

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "get interaction state", run: func() error { _, err := svc.GetInteractionState(ctx, &gamev1.GetInteractionStateRequest{}); return err }},
		{name: "set session gm authority", run: func() error {
			_, err := svc.SetSessionGMAuthority(ctx, &gamev1.SetSessionGMAuthorityRequest{ParticipantId: "gm-ai"})
			return err
		}},
		{name: "retry ai gm turn", run: func() error {
			_, err := svc.RetryAIGMTurn(ctx, &gamev1.RetryAIGMTurnRequest{})
			return err
		}},
		{name: "set active scene", run: func() error {
			_, err := svc.ActivateScene(ctx, &gamev1.ActivateSceneRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "start scene player phase", run: func() error {
			_, err := svc.OpenScenePlayerPhase(ctx, &gamev1.OpenScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "submit scene player post", run: func() error {
			_, err := svc.SubmitScenePlayerAction(ctx, &gamev1.SubmitScenePlayerActionRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "yield scene player phase", run: func() error {
			_, err := svc.YieldScenePlayerPhase(ctx, &gamev1.YieldScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "unyield scene player phase", run: func() error {
			_, err := svc.WithdrawScenePlayerYield(ctx, &gamev1.WithdrawScenePlayerYieldRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "end scene player phase", run: func() error {
			_, err := svc.InterruptScenePlayerPhase(ctx, &gamev1.InterruptScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "commit scene gm interaction", run: func() error {
			_, err := svc.RecordSceneGMInteraction(ctx, &gamev1.RecordSceneGMInteractionRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "resolve scene player phase review", run: func() error {
			_, err := svc.ResolveScenePlayerReview(ctx, &gamev1.ResolveScenePlayerReviewRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "pause session for ooc", run: func() error {
			_, err := svc.OpenSessionOOC(ctx, &gamev1.OpenSessionOOCRequest{})
			return err
		}},
		{name: "post session ooc", run: func() error {
			_, err := svc.PostSessionOOC(ctx, &gamev1.PostSessionOOCRequest{})
			return err
		}},
		{name: "mark ooc ready", run: func() error {
			_, err := svc.MarkOOCReadyToResume(ctx, &gamev1.MarkOOCReadyToResumeRequest{})
			return err
		}},
		{name: "clear ooc ready", run: func() error {
			_, err := svc.ClearOOCReadyToResume(ctx, &gamev1.ClearOOCReadyToResumeRequest{})
			return err
		}},
		{name: "resolve session ooc", run: func() error {
			_, err := svc.ResolveSessionOOC(ctx, &gamev1.ResolveSessionOOCRequest{})
			return err
		}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
			}
		})
	}
}
