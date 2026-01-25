package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeActorStore struct {
	putActor domain.Actor
	putErr   error
	getActor domain.Actor
	getErr   error
}

func (f *fakeActorStore) PutActor(ctx context.Context, actor domain.Actor) error {
	f.putActor = actor
	return f.putErr
}

func (f *fakeActorStore) GetActor(ctx context.Context, campaignID, actorID string) (domain.Actor, error) {
	return f.getActor, f.getErr
}

func TestCreateActorSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	actorStore := &fakeActorStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
			Actor:    actorStore,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "actor-456", nil
		},
	}

	response, err := service.CreateActor(context.Background(), &campaignv1.CreateActorRequest{
		CampaignId: "camp-123",
		Name:       "  Alice  ",
		Kind:       campaignv1.ActorKind_PC,
		Notes:      "A brave warrior",
	})
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if response == nil || response.Actor == nil {
		t.Fatal("expected actor response")
	}
	if response.Actor.Id != "actor-456" {
		t.Fatalf("expected id actor-456, got %q", response.Actor.Id)
	}
	if response.Actor.CampaignId != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", response.Actor.CampaignId)
	}
	if response.Actor.Name != "Alice" {
		t.Fatalf("expected trimmed name, got %q", response.Actor.Name)
	}
	if response.Actor.Kind != campaignv1.ActorKind_PC {
		t.Fatalf("expected kind PC, got %v", response.Actor.Kind)
	}
	if response.Actor.Notes != "A brave warrior" {
		t.Fatalf("expected notes, got %q", response.Actor.Notes)
	}
	if response.Actor.CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Actor.CreatedAt.AsTime())
	}
	if actorStore.putActor.ID != "actor-456" {
		t.Fatalf("expected stored id actor-456, got %q", actorStore.putActor.ID)
	}
}

func TestCreateActorValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  *campaignv1.CreateActorRequest
	}{
		{
			name: "empty name",
			req: &campaignv1.CreateActorRequest{
				CampaignId: "camp-123",
				Name:       "  ",
				Kind:       campaignv1.ActorKind_PC,
			},
		},
		{
			name: "missing kind",
			req: &campaignv1.CreateActorRequest{
				CampaignId: "camp-123",
				Name:       "Alice",
				Kind:       campaignv1.ActorKind_ACTOR_KIND_UNSPECIFIED,
			},
		},
		{
			name: "empty campaign id",
			req: &campaignv1.CreateActorRequest{
				CampaignId: "  ",
				Name:       "Alice",
				Kind:       campaignv1.ActorKind_PC,
			},
		},
	}

	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: id}, nil
	}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
			Actor:    &fakeActorStore{},
		},
		clock:      time.Now,
		idGenerator: func() (string, error) { return "actor-1", nil },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateActor(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected grpc status error, got %v", err)
			}
			if st.Code() != codes.InvalidArgument {
				t.Fatalf("expected invalid argument, got %v", st.Code())
			}
		})
	}
}

func TestCreateActorCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{}, storage.ErrNotFound
	}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
			Actor:    &fakeActorStore{},
		},
		clock:      time.Now,
		idGenerator: func() (string, error) { return "actor-1", nil },
	}

	_, err := service.CreateActor(context.Background(), &campaignv1.CreateActorRequest{
		CampaignId: "missing",
		Name:       "Alice",
		Kind:       campaignv1.ActorKind_PC,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected not found, got %v", st.Code())
	}
}

func TestCreateActorNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Actor:       &fakeActorStore{},
	})

	_, err := service.CreateActor(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
}

func TestCreateActorStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	actorStore := &fakeActorStore{putErr: errors.New("boom")}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
			Actor:    actorStore,
		},
		clock:      time.Now,
		idGenerator: func() (string, error) { return "actor-123", nil },
	}

	_, err := service.CreateActor(context.Background(), &campaignv1.CreateActorRequest{
		CampaignId: "camp-123",
		Name:       "Alice",
		Kind:       campaignv1.ActorKind_PC,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal error, got %v", st.Code())
	}
	if !strings.Contains(st.Message(), "persist actor") {
		t.Fatalf("expected error message to mention 'persist actor', got %q", st.Message())
	}
}

func TestCreateActorNPCKind(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	actorStore := &fakeActorStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
			Actor:    actorStore,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "actor-789", nil
		},
	}

	response, err := service.CreateActor(context.Background(), &campaignv1.CreateActorRequest{
		CampaignId: "camp-123",
		Name:       "Goblin",
		Kind:       campaignv1.ActorKind_NPC,
		Notes:      "A small creature",
	})
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if response.Actor.Kind != campaignv1.ActorKind_NPC {
		t.Fatalf("expected kind NPC, got %v", response.Actor.Kind)
	}
	if actorStore.putActor.Kind != domain.ActorKindNPC {
		t.Fatalf("expected stored kind NPC, got %v", actorStore.putActor.Kind)
	}
}

func TestCreateActorMissingStore(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
		},
	}

	_, err := service.CreateActor(context.Background(), &campaignv1.CreateActorRequest{
		CampaignId: "camp-123",
		Name:       "Alice",
		Kind:       campaignv1.ActorKind_PC,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal error, got %v", st.Code())
	}
}
