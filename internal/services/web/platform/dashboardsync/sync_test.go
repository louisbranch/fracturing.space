package dashboardsync

import (
	"context"
	"io"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestProfileSavedInvalidatesUserDashboard(t *testing.T) {
	t.Parallel()

	userhub := &userhubControlStub{}
	syncer := New(userhub, nil, nil)

	syncer.ProfileSaved(context.Background(), "user-1")

	if userhub.req == nil {
		t.Fatalf("expected invalidation request")
	}
	if got := userhub.req.GetUserIds(); len(got) != 1 || got[0] != "user-1" {
		t.Fatalf("UserIds = %v, want [user-1]", got)
	}
	if got := userhub.req.GetReason(); got != "web.profile_saved" {
		t.Fatalf("Reason = %q, want %q", got, "web.profile_saved")
	}
}

func TestCampaignCreatedWaitsForProjectionAndInvalidates(t *testing.T) {
	t.Parallel()

	userhub := &userhubControlStub{}
	game := &gameEventStub{
		listResp: &gamev1.ListEventsResponse{Events: []*gamev1.Event{{Seq: 8}}},
		stream: &campaignStreamStub{
			updates: []*gamev1.CampaignUpdate{{CampaignId: "camp-1", Seq: 8, Update: &gamev1.CampaignUpdate_ProjectionApplied{
				ProjectionApplied: &gamev1.ProjectionApplied{Scopes: []string{projectionScopeCampaigns}},
			}}},
		},
	}
	syncer := New(userhub, game, nil)

	syncer.CampaignCreated(context.Background(), "user-1", "camp-1")

	if game.subscribeReq == nil {
		t.Fatalf("expected SubscribeCampaignUpdates call")
	}
	if game.subscribeReq.GetAfterSeq() != 7 {
		t.Fatalf("AfterSeq = %d, want 7", game.subscribeReq.GetAfterSeq())
	}
	if got := game.subscribeReq.GetProjectionScopes(); len(got) != 1 || got[0] != projectionScopeCampaigns {
		t.Fatalf("ProjectionScopes = %v, want [%s]", got, projectionScopeCampaigns)
	}
	if userhub.req == nil {
		t.Fatalf("expected invalidation request")
	}
	if got := userhub.req.GetCampaignIds(); len(got) != 1 || got[0] != "camp-1" {
		t.Fatalf("CampaignIds = %v, want [camp-1]", got)
	}
}

func TestSessionStartedInvalidatesWhenProjectionWaitTimesOut(t *testing.T) {
	t.Parallel()

	userhub := &userhubControlStub{}
	game := &gameEventStub{
		listResp: &gamev1.ListEventsResponse{Events: []*gamev1.Event{{Seq: 3}}},
		stream:   &blockingCampaignStreamStub{},
	}
	syncer := New(userhub, game, nil)
	syncer.waitTimeout = 10 * time.Millisecond

	syncer.SessionStarted(context.Background(), "user-1", "camp-2")

	if userhub.req == nil {
		t.Fatalf("expected invalidation request after timeout")
	}
	if got := userhub.req.GetCampaignIds(); len(got) != 1 || got[0] != "camp-2" {
		t.Fatalf("CampaignIds = %v, want [camp-2]", got)
	}
}

func TestInviteChangedInvalidatesWithoutProjectionWait(t *testing.T) {
	t.Parallel()

	userhub := &userhubControlStub{}
	game := &gameEventStub{}
	syncer := New(userhub, game, nil)

	syncer.InviteChanged(context.Background(), []string{"user-9"}, "camp-3")

	// Invite service uses direct SQL — no game projection wait should occur.
	if game.subscribeReq != nil {
		t.Fatalf("InviteChanged should not subscribe to campaign updates")
	}
	if userhub.req == nil {
		t.Fatalf("expected invalidation request")
	}
	if got := userhub.req.GetUserIds(); len(got) != 1 || got[0] != "user-9" {
		t.Fatalf("UserIds = %v, want [user-9]", got)
	}
	if got := userhub.req.GetCampaignIds(); len(got) != 1 || got[0] != "camp-3" {
		t.Fatalf("CampaignIds = %v, want [camp-3]", got)
	}
	if got := userhub.req.GetReason(); got != "web.invite_changed" {
		t.Fatalf("Reason = %q, want %q", got, "web.invite_changed")
	}
}

type userhubControlStub struct {
	req *userhubv1.InvalidateDashboardsRequest
}

func (s *userhubControlStub) InvalidateDashboards(_ context.Context, req *userhubv1.InvalidateDashboardsRequest, _ ...grpc.CallOption) (*userhubv1.InvalidateDashboardsResponse, error) {
	s.req = req
	return &userhubv1.InvalidateDashboardsResponse{}, nil
}

type gameEventStub struct {
	listResp     *gamev1.ListEventsResponse
	stream       grpc.ServerStreamingClient[gamev1.CampaignUpdate]
	subscribeReq *gamev1.SubscribeCampaignUpdatesRequest
}

func (s *gameEventStub) ListEvents(context.Context, *gamev1.ListEventsRequest, ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
	return s.listResp, nil
}

func (s *gameEventStub) SubscribeCampaignUpdates(_ context.Context, req *gamev1.SubscribeCampaignUpdatesRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	s.subscribeReq = req
	return s.stream, nil
}

type campaignStreamStub struct {
	updates []*gamev1.CampaignUpdate
	index   int
}

func (s *campaignStreamStub) Header() (metadata.MD, error) { return nil, nil }
func (s *campaignStreamStub) Trailer() metadata.MD         { return nil }
func (s *campaignStreamStub) CloseSend() error             { return nil }
func (s *campaignStreamStub) Context() context.Context     { return context.Background() }
func (s *campaignStreamStub) SendMsg(any) error            { return nil }
func (s *campaignStreamStub) RecvMsg(any) error            { return nil }

func (s *campaignStreamStub) Recv() (*gamev1.CampaignUpdate, error) {
	if s.index >= len(s.updates) {
		return nil, io.EOF
	}
	update := s.updates[s.index]
	s.index++
	return update, nil
}

type blockingCampaignStreamStub struct{}

func (s *blockingCampaignStreamStub) Header() (metadata.MD, error) { return nil, nil }
func (s *blockingCampaignStreamStub) Trailer() metadata.MD         { return nil }
func (s *blockingCampaignStreamStub) CloseSend() error             { return nil }
func (s *blockingCampaignStreamStub) Context() context.Context     { return context.Background() }
func (s *blockingCampaignStreamStub) SendMsg(any) error            { return nil }
func (s *blockingCampaignStreamStub) RecvMsg(any) error            { return nil }

func (s *blockingCampaignStreamStub) Recv() (*gamev1.CampaignUpdate, error) {
	time.Sleep(50 * time.Millisecond)
	return nil, context.DeadlineExceeded
}
