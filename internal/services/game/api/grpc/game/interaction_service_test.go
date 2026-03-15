package game

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
		{name: "set active scene", run: func() error { _, err := svc.SetActiveScene(ctx, nil); return err }},
		{name: "start scene player phase", run: func() error { _, err := svc.StartScenePlayerPhase(ctx, nil); return err }},
		{name: "submit scene player post", run: func() error { _, err := svc.SubmitScenePlayerPost(ctx, nil); return err }},
		{name: "yield scene player phase", run: func() error { _, err := svc.YieldScenePlayerPhase(ctx, nil); return err }},
		{name: "unyield scene player phase", run: func() error { _, err := svc.UnyieldScenePlayerPhase(ctx, nil); return err }},
		{name: "end scene player phase", run: func() error { _, err := svc.EndScenePlayerPhase(ctx, nil); return err }},
		{name: "accept scene player phase", run: func() error { _, err := svc.AcceptScenePlayerPhase(ctx, nil); return err }},
		{name: "request scene player revisions", run: func() error { _, err := svc.RequestScenePlayerRevisions(ctx, nil); return err }},
		{name: "pause session for ooc", run: func() error { _, err := svc.PauseSessionForOOC(ctx, nil); return err }},
		{name: "post session ooc", run: func() error { _, err := svc.PostSessionOOC(ctx, nil); return err }},
		{name: "mark ooc ready", run: func() error { _, err := svc.MarkOOCReadyToResume(ctx, nil); return err }},
		{name: "clear ooc ready", run: func() error { _, err := svc.ClearOOCReadyToResume(ctx, nil); return err }},
		{name: "resume from ooc", run: func() error { _, err := svc.ResumeFromOOC(ctx, nil); return err }},
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
			_, err := svc.SetActiveScene(ctx, &gamev1.SetActiveSceneRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "start scene player phase", run: func() error {
			_, err := svc.StartScenePlayerPhase(ctx, &gamev1.StartScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "submit scene player post", run: func() error {
			_, err := svc.SubmitScenePlayerPost(ctx, &gamev1.SubmitScenePlayerPostRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "yield scene player phase", run: func() error {
			_, err := svc.YieldScenePlayerPhase(ctx, &gamev1.YieldScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "unyield scene player phase", run: func() error {
			_, err := svc.UnyieldScenePlayerPhase(ctx, &gamev1.UnyieldScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "end scene player phase", run: func() error {
			_, err := svc.EndScenePlayerPhase(ctx, &gamev1.EndScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "accept scene player phase", run: func() error {
			_, err := svc.AcceptScenePlayerPhase(ctx, &gamev1.AcceptScenePlayerPhaseRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "request scene player revisions", run: func() error {
			_, err := svc.RequestScenePlayerRevisions(ctx, &gamev1.RequestScenePlayerRevisionsRequest{SceneId: "scene-1"})
			return err
		}},
		{name: "pause session for ooc", run: func() error {
			_, err := svc.PauseSessionForOOC(ctx, &gamev1.PauseSessionForOOCRequest{})
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
		{name: "resume from ooc", run: func() error {
			_, err := svc.ResumeFromOOC(ctx, &gamev1.ResumeFromOOCRequest{})
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
