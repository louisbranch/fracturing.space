package eventtransport

import (
	"context"
	"sync"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestSubscribeCampaignUpdates_MissingCampaignID(t *testing.T) {
	svc := NewService(Deps{Event: gametest.NewFakeEventStore()})
	stream := &fakeCampaignUpdateStream{ctx: context.Background()}

	err := svc.SubscribeCampaignUpdates(&campaignv1.SubscribeCampaignUpdatesRequest{}, stream)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSubscribeCampaignUpdates_RequiresCampaignReadPolicy(t *testing.T) {
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore},
		Event:       gametest.NewFakeEventStore(),
		Participant: participantStore,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := &fakeCampaignUpdateStream{ctx: ctx}

	err := svc.SubscribeCampaignUpdates(&campaignv1.SubscribeCampaignUpdatesRequest{CampaignId: "camp-1"}, stream)
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestSubscribeCampaignUpdates_StreamsCommittedAndProjectionUpdates(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	now := time.Now().UTC()
	eventStore.Events["camp-1"] = []event.Event{
		{
			CampaignID: "camp-1",
			Seq:        2,
			Type:       event.Type("character.updated"),
			Timestamp:  now,
			EntityType: "character",
			EntityID:   "char-1",
		},
	}

	svc := NewService(Deps{Event: eventStore})
	ctx, cancel := context.WithCancel(gametest.ContextWithAdminOverride("events-test"))
	stream := &fakeCampaignUpdateStream{ctx: ctx}
	stream.onSend = func() {
		if len(stream.updates) >= 2 {
			cancel()
		}
	}
	defer cancel()

	err := svc.SubscribeCampaignUpdates(&campaignv1.SubscribeCampaignUpdatesRequest{
		CampaignId: "camp-1",
		AfterSeq:   1,
	}, stream)
	if err != nil {
		t.Fatalf("subscribe campaign updates: %v", err)
	}

	if len(stream.updates) != 2 {
		t.Fatalf("updates = %d, want %d", len(stream.updates), 2)
	}

	committed := stream.updates[0]
	if committed.GetCampaignId() != "camp-1" {
		t.Fatalf("committed campaign id = %q, want %q", committed.GetCampaignId(), "camp-1")
	}
	if committed.GetSeq() != 2 {
		t.Fatalf("committed seq = %d, want %d", committed.GetSeq(), 2)
	}
	if committed.GetEventCommitted() == nil {
		t.Fatalf("expected committed update kind")
	}

	applied := stream.updates[1]
	if applied.GetProjectionApplied() == nil {
		t.Fatalf("expected projection_applied update kind")
	}
	if applied.GetProjectionApplied().GetSourceSeq() != 2 {
		t.Fatalf("projection source seq = %d, want %d", applied.GetProjectionApplied().GetSourceSeq(), 2)
	}
	if len(applied.GetProjectionApplied().GetScopes()) == 0 {
		t.Fatalf("projection scopes = empty, want non-empty")
	}
}

type fakeCampaignUpdateStream struct {
	ctx     context.Context
	mu      sync.Mutex
	updates []*campaignv1.CampaignUpdate
	onSend  func()
}

func (f *fakeCampaignUpdateStream) Send(update *campaignv1.CampaignUpdate) error {
	f.mu.Lock()
	f.updates = append(f.updates, update)
	hook := f.onSend
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	return nil
}

func (f *fakeCampaignUpdateStream) SetHeader(metadata.MD) error { return nil }

func (f *fakeCampaignUpdateStream) SendHeader(metadata.MD) error { return nil }

func (f *fakeCampaignUpdateStream) SetTrailer(metadata.MD) {}

func (f *fakeCampaignUpdateStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}

func (f *fakeCampaignUpdateStream) SendMsg(any) error { return nil }

func (f *fakeCampaignUpdateStream) RecvMsg(any) error { return nil }
