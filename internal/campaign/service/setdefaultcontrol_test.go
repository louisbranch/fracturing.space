package service

import (
	"context"
	"errors"
	"testing"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeControlDefaultStore struct {
	putController  domain.CharacterController
	putCampaignID  string
	putCharacterID string
	putErr         error
	getController  domain.CharacterController
	getErr         error
}

func (f *fakeControlDefaultStore) PutControlDefault(ctx context.Context, campaignID, characterID string, controller domain.CharacterController) error {
	f.putCampaignID = campaignID
	f.putCharacterID = characterID
	f.putController = controller
	return f.putErr
}

func (f *fakeControlDefaultStore) GetControlDefault(ctx context.Context, campaignID, characterID string) (domain.CharacterController, error) {
	return f.getController, f.getErr
}

func TestSetDefaultControlSuccessGM(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}

	characterStore := &fakeCharacterStore{}
	characterStore.getErr = nil
	characterStore.getCharacter = domain.Character{
		ID:         "character-456",
		CampaignID: "camp-123",
		Name:       "Test Character",
	}

	controlStore := &fakeControlDefaultStore{}

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			ControlDefault: controlStore,
		},
	}

	response, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
	})
	if err != nil {
		t.Fatalf("set default control: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if response.CampaignId != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", response.CampaignId)
	}
	if response.CharacterId != "character-456" {
		t.Fatalf("expected character id character-456, got %q", response.CharacterId)
	}
	if controlStore.putCampaignID != "camp-123" {
		t.Fatalf("expected stored campaign id camp-123, got %q", controlStore.putCampaignID)
	}
	if controlStore.putCharacterID != "character-456" {
		t.Fatalf("expected stored character id character-456, got %q", controlStore.putCharacterID)
	}
	if !controlStore.putController.IsGM {
		t.Fatal("expected stored controller to be GM")
	}
	if controlStore.putController.ParticipantID != "" {
		t.Fatalf("expected empty participant ID, got %q", controlStore.putController.ParticipantID)
	}
}

func TestSetDefaultControlSuccessParticipant(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}

	characterStore := &fakeCharacterStore{}
	characterStore.getErr = nil
	characterStore.getCharacter = domain.Character{
		ID:         "character-456",
		CampaignID: "camp-123",
		Name:       "Test Character",
	}

	participantStore := &fakeParticipantStore{}
	participantStore.getErr = nil
	participantStore.getParticipant = domain.Participant{
		ID:         "participant-789",
		CampaignID: "camp-123",
	}

	controlStore := &fakeControlDefaultStore{}

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Participant:    participantStore,
			ControlDefault: controlStore,
		},
	}

	response, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Participant{
				Participant: &campaignv1.ParticipantController{
					ParticipantId: "participant-789",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("set default control: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if response.CampaignId != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", response.CampaignId)
	}
	if response.CharacterId != "character-456" {
		t.Fatalf("expected character id character-456, got %q", response.CharacterId)
	}
	if controlStore.putCampaignID != "camp-123" {
		t.Fatalf("expected stored campaign id camp-123, got %q", controlStore.putCampaignID)
	}
	if controlStore.putCharacterID != "character-456" {
		t.Fatalf("expected stored character id character-456, got %q", controlStore.putCharacterID)
	}
	if controlStore.putController.IsGM {
		t.Fatal("expected stored controller to be participant")
	}
	if controlStore.putController.ParticipantID != "participant-789" {
		t.Fatalf("expected participant ID participant-789, got %q", controlStore.putController.ParticipantID)
	}
}

func TestSetDefaultControlNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:       &fakeCampaignStore{},
		Character:      &fakeCharacterStore{},
		ControlDefault: &fakeControlDefaultStore{},
	})

	_, err := service.SetDefaultControl(context.Background(), nil)
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

func TestSetDefaultControlMissingCampaign(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getErr = storage.ErrNotFound

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      &fakeCharacterStore{},
			ControlDefault: &fakeControlDefaultStore{},
		},
	}

	_, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
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
		t.Fatalf("expected 'campaign not found', got %q", st.Message())
	}
}

func TestSetDefaultControlMissingCharacter(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}

	characterStore := &fakeCharacterStore{}
	characterStore.getErr = storage.ErrNotFound

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			ControlDefault: &fakeControlDefaultStore{},
		},
	}

	_, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
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
	if st.Message() != "character not found" {
		t.Fatalf("expected 'character not found', got %q", st.Message())
	}
}

func TestSetDefaultControlMissingParticipant(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}

	characterStore := &fakeCharacterStore{}
	characterStore.getErr = nil
	characterStore.getCharacter = domain.Character{
		ID:         "character-456",
		CampaignID: "camp-123",
		Name:       "Test Character",
	}

	participantStore := &fakeParticipantStore{}
	participantStore.getErr = storage.ErrNotFound

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Participant:    participantStore,
			ControlDefault: &fakeControlDefaultStore{},
		},
	}

	_, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Participant{
				Participant: &campaignv1.ParticipantController{
					ParticipantId: "participant-789",
				},
			},
		},
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
		t.Fatalf("expected 'participant not found', got %q", st.Message())
	}
}

func TestSetDefaultControlInvalidController(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}

	characterStore := &fakeCharacterStore{}
	characterStore.getErr = nil
	characterStore.getCharacter = domain.Character{
		ID:         "character-456",
		CampaignID: "camp-123",
		Name:       "Test Character",
	}

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			ControlDefault: &fakeControlDefaultStore{},
		},
	}

	tests := []struct {
		name     string
		request  *campaignv1.SetDefaultControlRequest
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name: "nil controller",
			request: &campaignv1.SetDefaultControlRequest{
				CampaignId:  "camp-123",
				CharacterId: "character-456",
				Controller:  nil,
			},
			wantCode: codes.InvalidArgument,
			wantMsg:  "controller is required",
		},
		{
			name: "empty campaign id",
			request: &campaignv1.SetDefaultControlRequest{
				CampaignId:  "   ",
				CharacterId: "character-456",
				Controller: &campaignv1.CharacterController{
					Controller: &campaignv1.CharacterController_Gm{
						Gm: &campaignv1.GmController{},
					},
				},
			},
			wantCode: codes.InvalidArgument,
			wantMsg:  "campaign id is required",
		},
		{
			name: "empty character id",
			request: &campaignv1.SetDefaultControlRequest{
				CampaignId:  "camp-123",
				CharacterId: "   ",
				Controller: &campaignv1.CharacterController{
					Controller: &campaignv1.CharacterController_Gm{
						Gm: &campaignv1.GmController{},
					},
				},
			},
			wantCode: codes.InvalidArgument,
			wantMsg:  "character id is required",
		},
		{
			name: "empty participant id",
			request: &campaignv1.SetDefaultControlRequest{
				CampaignId:  "camp-123",
				CharacterId: "character-456",
				Controller: &campaignv1.CharacterController{
					Controller: &campaignv1.CharacterController_Participant{
						Participant: &campaignv1.ParticipantController{
							ParticipantId: "   ",
						},
					},
				},
			},
			wantCode: codes.InvalidArgument,
			wantMsg:  "participant id is required when participant controller is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.SetDefaultControl(context.Background(), tt.request)
			if err == nil {
				t.Fatal("expected error")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected grpc status error, got %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Fatalf("expected code %v, got %v", tt.wantCode, st.Code())
			}
			if st.Message() != tt.wantMsg {
				t.Fatalf("expected message %q, got %q", tt.wantMsg, st.Message())
			}
		})
	}
}

func TestSetDefaultControlStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}

	characterStore := &fakeCharacterStore{}
	characterStore.getErr = nil
	characterStore.getCharacter = domain.Character{
		ID:         "character-456",
		CampaignID: "camp-123",
		Name:       "Test Character",
	}

	controlStore := &fakeControlDefaultStore{}
	controlStore.putErr = errors.New("storage error")

	service := &CampaignService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			ControlDefault: controlStore,
		},
	}

	_, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
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

func TestSetDefaultControlMissingStore(t *testing.T) {
	service := &CampaignService{
		stores: Stores{},
	}

	_, err := service.SetDefaultControl(context.Background(), &campaignv1.SetDefaultControlRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
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
