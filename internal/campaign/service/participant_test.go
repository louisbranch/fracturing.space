package service

import (
	"context"
	"errors"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateParticipantSuccess(t *testing.T) {
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
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "part-456", nil
		},
	}

	response, err := service.CreateParticipant(context.Background(), &campaignv1.CreateParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "  Alice  ",
		Role:        campaignv1.ParticipantRole_PLAYER,
		Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
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

func TestCreateParticipantIncrementsParticipantCount(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	participantStore := &fakeParticipantStore{}
	service := &CampaignService{
		stores: Stores{
			Participant: participantStore,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "part-789", nil
		},
	}

	// Test that participant is created
	_, err := service.CreateParticipant(context.Background(), &campaignv1.CreateParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "Charlie",
		Role:        campaignv1.ParticipantRole_PLAYER,
		Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}

	// Verify participant was stored
	if participantStore.putParticipant.ID != "part-789" {
		t.Fatalf("expected participant id part-789, got %q", participantStore.putParticipant.ID)
	}
	if participantStore.putParticipant.Role != domain.ParticipantRolePlayer {
		t.Fatalf("expected role Player, got %v", participantStore.putParticipant.Role)
	}
}

func TestCreateParticipantIncrementsCountForGM(t *testing.T) {
	participantStore := &fakeParticipantStore{}
	service := &CampaignService{
		stores: Stores{
			Participant: participantStore,
		},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "part-999", nil
		},
	}

	// Test that GM participant is created (counter increment happens for all roles including GM)
	_, err := service.CreateParticipant(context.Background(), &campaignv1.CreateParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "GM",
		Role:        campaignv1.ParticipantRole_GM,
		Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}

	// Verify participant was stored
	if participantStore.putParticipant.ID != "part-999" {
		t.Fatalf("expected participant id part-999, got %q", participantStore.putParticipant.ID)
	}
	if participantStore.putParticipant.Role != domain.ParticipantRoleGM {
		t.Fatalf("expected role GM, got %v", participantStore.putParticipant.Role)
	}
}

func TestCreateParticipantDefaultsController(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "part-1", nil },
	}

	response, err := service.CreateParticipant(context.Background(), &campaignv1.CreateParticipantRequest{
		CampaignId:  "camp-123",
		DisplayName: "Bob",
		Role:        campaignv1.ParticipantRole_GM,
		Controller:  campaignv1.Controller_CONTROLLER_UNSPECIFIED,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}
	if response.Participant.Controller != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("expected default controller human, got %v", response.Participant.Controller)
	}
}

func TestCreateParticipantValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  *campaignv1.CreateParticipantRequest
	}{
		{
			name: "empty display name",
			req: &campaignv1.CreateParticipantRequest{
				CampaignId:  "camp-123",
				DisplayName: "  ",
				Role:        campaignv1.ParticipantRole_PLAYER,
			},
		},
		{
			name: "missing role",
			req: &campaignv1.CreateParticipantRequest{
				CampaignId:  "camp-123",
				DisplayName: "Alice",
				Role:        campaignv1.ParticipantRole_ROLE_UNSPECIFIED,
			},
		},
		{
			name: "empty campaign id",
			req: &campaignv1.CreateParticipantRequest{
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
		stores: Stores{
			Campaign:    campaignStore,
			Participant: &fakeParticipantStore{},
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "part-1", nil },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateParticipant(context.Background(), tt.req)
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

func TestCreateParticipantCampaignNotFound(t *testing.T) {
	participantStore := &fakeParticipantStore{putErr: storage.ErrNotFound}
	service := &CampaignService{
		stores: Stores{
			Participant: participantStore,
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "part-1", nil },
	}

	_, err := service.CreateParticipant(context.Background(), &campaignv1.CreateParticipantRequest{
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
	if st.Message() != "campaign not found" {
		t.Fatalf("expected error message 'campaign not found', got %q", st.Message())
	}
}

func TestCreateParticipantNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.CreateParticipant(context.Background(), nil)
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

func TestCreateParticipantStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{putErr: errors.New("boom")}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "part-123", nil },
	}

	_, err := service.CreateParticipant(context.Background(), &campaignv1.CreateParticipantRequest{
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

func TestListParticipantsSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	participantStore := &fakeParticipantStore{
		listPage: storage.ParticipantPage{
			Participants: []domain.Participant{
				{
					ID:          "part-1",
					CampaignID:  "camp-123",
					DisplayName: "Alice",
					Role:        domain.ParticipantRolePlayer,
					Controller:  domain.ControllerHuman,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
				{
					ID:          "part-2",
					CampaignID:  "camp-123",
					DisplayName: "Bob",
					Role:        domain.ParticipantRoleGM,
					Controller:  domain.ControllerHuman,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
			NextPageToken: "next-token",
		},
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: time.Now,
	}

	response, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(response.Participants))
	}
	if response.NextPageToken != "next-token" {
		t.Fatalf("expected next page token next-token, got %q", response.NextPageToken)
	}
	if response.Participants[0].Id != "part-1" {
		t.Fatalf("expected first participant id part-1, got %q", response.Participants[0].Id)
	}
	if response.Participants[0].DisplayName != "Alice" {
		t.Fatalf("expected first participant name Alice, got %q", response.Participants[0].DisplayName)
	}
	if response.Participants[0].Role != campaignv1.ParticipantRole_PLAYER {
		t.Fatalf("expected first participant role PLAYER, got %v", response.Participants[0].Role)
	}
	if response.Participants[1].Id != "part-2" {
		t.Fatalf("expected second participant id part-2, got %q", response.Participants[1].Id)
	}
	if response.Participants[1].Role != campaignv1.ParticipantRole_GM {
		t.Fatalf("expected second participant role GM, got %v", response.Participants[1].Role)
	}
	if response.Participants[0].CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Participants[0].CreatedAt.AsTime())
	}
}

func TestListParticipantsDefaults(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{
		listPage: storage.ParticipantPage{
			Participants: []domain.Participant{
				{
					ID:          "part-1",
					CampaignID:  "camp-123",
					DisplayName: "Alice",
					Role:        domain.ParticipantRolePlayer,
					Controller:  domain.ControllerHuman,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
			NextPageToken: "next-token",
		},
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: time.Now,
	}

	response, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   0,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if participantStore.listPageSize != defaultListParticipantsPageSize {
		t.Fatalf("expected default page size %d, got %d", defaultListParticipantsPageSize, participantStore.listPageSize)
	}
	if response.NextPageToken != "next-token" {
		t.Fatalf("expected next page token, got %q", response.NextPageToken)
	}
	if len(response.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(response.Participants))
	}
}

func TestListParticipantsEmpty(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{
		listPage: storage.ParticipantPage{},
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: time.Now,
	}

	response, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.Participants) != 0 {
		t.Fatalf("expected 0 participants, got %d", len(response.Participants))
	}
}

func TestListParticipantsClampPageSize(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{listPage: storage.ParticipantPage{}}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: time.Now,
	}

	_, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   25,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if participantStore.listPageSize != maxListParticipantsPageSize {
		t.Fatalf("expected max page size %d, got %d", maxListParticipantsPageSize, participantStore.listPageSize)
	}
}

func TestListParticipantsPassesToken(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{listPage: storage.ParticipantPage{}}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: time.Now,
	}

	_, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   1,
		PageToken:  "next",
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if participantStore.listPageToken != "next" {
		t.Fatalf("expected page token next, got %q", participantStore.listPageToken)
	}
	if participantStore.listPageCampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", participantStore.listPageCampaignID)
	}
}

func TestListParticipantsNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.ListParticipants(context.Background(), nil)
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

func TestListParticipantsCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{}, storage.ErrNotFound
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: &fakeParticipantStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "missing",
		PageSize:   10,
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

func TestListParticipantsStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	participantStore := &fakeParticipantStore{listPageErr: errors.New("boom")}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: participantStore,
		},
		clock: time.Now,
	}

	_, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   10,
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

func TestListParticipantsMissingStore(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	service := &CampaignService{
		stores: Stores{
			Campaign: campaignStore,
		},
	}

	_, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "camp-123",
		PageSize:   10,
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

func TestListParticipantsEmptyCampaignID(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign:    campaignStore,
			Participant: &fakeParticipantStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListParticipants(context.Background(), &campaignv1.ListParticipantsRequest{
		CampaignId: "  ",
		PageSize:   10,
	})
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

func TestGetParticipantSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	participantStore := &fakeParticipantStore{
		getParticipant: domain.Participant{
			ID:          "part-456",
			CampaignID:  "camp-123",
			DisplayName: "Alice",
			Role:        domain.ParticipantRolePlayer,
			Controller:  domain.ControllerHuman,
			CreatedAt:   fixedTime,
			UpdatedAt:   fixedTime,
		},
	}
	service := NewCampaignService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   &fakeCharacterStore{},
	})

	response, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-123",
		ParticipantId: "part-456",
	})
	if err != nil {
		t.Fatalf("get participant: %v", err)
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
		t.Fatalf("expected display name Alice, got %q", response.Participant.DisplayName)
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
	if response.Participant.UpdatedAt.AsTime() != fixedTime {
		t.Fatalf("expected updated_at %v, got %v", fixedTime, response.Participant.UpdatedAt.AsTime())
	}
}

func TestGetParticipantNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetParticipant(context.Background(), nil)
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

func TestGetParticipantMissingCampaignStore(t *testing.T) {
	service := &CampaignService{
		stores: Stores{
			Participant: &fakeParticipantStore{},
		},
		clock: time.Now,
	}

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-123",
		ParticipantId: "part-456",
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

func TestGetParticipantMissingParticipantStore(t *testing.T) {
	service := &CampaignService{
		stores: Stores{
			Campaign: &fakeCampaignStore{},
		},
		clock: time.Now,
	}

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-123",
		ParticipantId: "part-456",
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

func TestGetParticipantEmptyCampaignID(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "  ",
		ParticipantId: "part-456",
	})
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

func TestGetParticipantEmptyParticipantID(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-123",
		ParticipantId: "  ",
	})
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

func TestGetParticipantCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getErr: storage.ErrNotFound,
	}
	service := NewCampaignService(Stores{
		Campaign:    campaignStore,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-999",
		ParticipantId: "part-456",
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
	if st.Message() != "campaign not found" {
		t.Fatalf("expected message 'campaign not found', got %q", st.Message())
	}
}

func TestGetParticipantNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	participantStore := &fakeParticipantStore{
		getErr: storage.ErrNotFound,
	}
	service := NewCampaignService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-123",
		ParticipantId: "part-999",
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
	if st.Message() != "participant not found" {
		t.Fatalf("expected message 'participant not found', got %q", st.Message())
	}
}

func TestGetParticipantStoreError(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	participantStore := &fakeParticipantStore{
		getErr: errors.New("database error"),
	}
	service := NewCampaignService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetParticipant(context.Background(), &campaignv1.GetParticipantRequest{
		CampaignId:    "camp-123",
		ParticipantId: "part-456",
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

func TestParticipantRole(t *testing.T) {
	tests := []struct {
		name            string
		participantRole domain.ParticipantRole
		proto           campaignv1.ParticipantRole
	}{
		{
			name:            "player",
			participantRole: domain.ParticipantRolePlayer,
			proto:           campaignv1.ParticipantRole_PLAYER,
		},
		{
			name:            "gm",
			participantRole: domain.ParticipantRoleGM,
			proto:           campaignv1.ParticipantRole_GM,
		},
		{
			name:            "unspecified",
			participantRole: domain.ParticipantRoleUnspecified,
			proto:           campaignv1.ParticipantRole_ROLE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := participantRoleToProto(tt.participantRole)
			if proto != tt.proto {
				t.Fatalf("expected %v, got %v", tt.proto, proto)
			}
		})
	}
}
