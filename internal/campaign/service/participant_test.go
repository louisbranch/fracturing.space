package service

import (
	"context"
	"errors"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRegisterParticipantSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{
		putCampaign: domain.Campaign{ID: "camp-123"},
	}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	participantStore := &fakeParticipantStore{}
	service := &CampaignService{
		store:            campaignStore,
		participantStore: participantStore,
		clock: func() time.Time {
			return fixedTime
		},
		participantIDGen: func() (string, error) {
			return "part-456", nil
		},
	}

	response, err := service.RegisterParticipant(context.Background(), &campaignv1.RegisterParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "  Alice  ",
		Role:        campaignv1.ParticipantRole_PLAYER,
		Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("register participant: %v", err)
	}
	if response == nil || response.Participant == nil {
		t.Fatal("expected participant response")
	}
	if response.Participant.Id != "part-456" {
		t.Fatalf("expected id part-456, got %q", response.Participant.Id)
	}
	if response.Participant.CampaignId != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", response.Participant.CampaignId)
	}
	if response.Participant.DisplayName != "Alice" {
		t.Fatalf("expected trimmed display name, got %q", response.Participant.DisplayName)
	}
	if response.Participant.Role != campaignv1.ParticipantRole_PLAYER {
		t.Fatalf("expected role player, got %v", response.Participant.Role)
	}
	if response.Participant.Controller != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("expected controller human, got %v", response.Participant.Controller)
	}
	if response.Participant.CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Participant.CreatedAt.AsTime())
	}
	if participantStore.putParticipant.ID != "part-456" {
		t.Fatalf("expected stored id part-456, got %q", participantStore.putParticipant.ID)
	}
}

func TestRegisterParticipantDefaultsController(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{}
	service := &CampaignService{
		store:            campaignStore,
		participantStore: participantStore,
		clock:            time.Now,
		participantIDGen: func() (string, error) { return "part-1", nil },
	}

	response, err := service.RegisterParticipant(context.Background(), &campaignv1.RegisterParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "Bob",
		Role:        campaignv1.ParticipantRole_GM,
		Controller:  campaignv1.Controller_CONTROLLER_UNSPECIFIED,
	})
	if err != nil {
		t.Fatalf("register participant: %v", err)
	}
	if response.Participant.Controller != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("expected default controller human, got %v", response.Participant.Controller)
	}
}

func TestRegisterParticipantValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  *campaignv1.RegisterParticipantRequest
	}{
		{
			name: "empty display name",
			req: &campaignv1.RegisterParticipantRequest{
				CampaignId:  "camp-123",
				DisplayName: "  ",
				Role:        campaignv1.ParticipantRole_PLAYER,
			},
		},
		{
			name: "missing role",
			req: &campaignv1.RegisterParticipantRequest{
				CampaignId:  "camp-123",
				DisplayName: "Alice",
				Role:        campaignv1.ParticipantRole_ROLE_UNSPECIFIED,
			},
		},
		{
			name: "empty campaign id",
			req: &campaignv1.RegisterParticipantRequest{
				CampaignId:  "  ",
				DisplayName: "Alice",
				Role:        campaignv1.ParticipantRole_PLAYER,
			},
		},
	}

	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: id}, nil
	}
	service := &CampaignService{
		store:            campaignStore,
		participantStore: &fakeParticipantStore{},
		clock:            time.Now,
		participantIDGen: func() (string, error) { return "part-1", nil },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.RegisterParticipant(context.Background(), tt.req)
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

func TestRegisterParticipantCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{}, storage.ErrNotFound
	}
	service := &CampaignService{
		store:            campaignStore,
		participantStore: &fakeParticipantStore{},
		clock:            time.Now,
		participantIDGen: func() (string, error) { return "part-1", nil },
	}

	_, err := service.RegisterParticipant(context.Background(), &campaignv1.RegisterParticipantRequest{
		CampaignId:  "missing",
		DisplayName: "Alice",
		Role:        campaignv1.ParticipantRole_PLAYER,
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

func TestRegisterParticipantNilRequest(t *testing.T) {
	service := NewCampaignService(&fakeCampaignStore{}, &fakeParticipantStore{})

	_, err := service.RegisterParticipant(context.Background(), nil)
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

func TestRegisterParticipantStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{putErr: errors.New("boom")}
	service := &CampaignService{
		store:            campaignStore,
		participantStore: participantStore,
		clock:            time.Now,
		participantIDGen: func() (string, error) { return "part-123", nil },
	}

	_, err := service.RegisterParticipant(context.Background(), &campaignv1.RegisterParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "Alice",
		Role:        campaignv1.ParticipantRole_PLAYER,
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
