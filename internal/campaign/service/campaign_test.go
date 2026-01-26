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

type fakeCampaignStore struct {
	putCampaign domain.Campaign
	putErr      error
	listPage    storage.CampaignPage
	listErr     error
	listSize    int
	listToken   string
	getCampaign domain.Campaign
	getErr      error
	getFunc     func(ctx context.Context, id string) (domain.Campaign, error)
}

func (f *fakeCampaignStore) Put(ctx context.Context, campaign domain.Campaign) error {
	f.putCampaign = campaign
	return f.putErr
}

func (f *fakeCampaignStore) Get(ctx context.Context, id string) (domain.Campaign, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, id)
	}
	return f.getCampaign, f.getErr
}

func (f *fakeCampaignStore) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	f.listSize = pageSize
	f.listToken = pageToken
	return f.listPage, f.listErr
}

type fakeParticipantStore struct {
	putParticipant     domain.Participant
	putErr             error
	getParticipant     domain.Participant
	getErr             error
	listParticipants   []domain.Participant
	listErr            error
	listPage           storage.ParticipantPage
	listPageErr        error
	listPageSize       int
	listPageToken      string
	listPageCampaignID string
}

func (f *fakeParticipantStore) PutParticipant(ctx context.Context, participant domain.Participant) error {
	f.putParticipant = participant
	return f.putErr
}

func (f *fakeParticipantStore) GetParticipant(ctx context.Context, campaignID, participantID string) (domain.Participant, error) {
	return f.getParticipant, f.getErr
}

func (f *fakeParticipantStore) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	return f.listParticipants, f.listErr
}

func (f *fakeParticipantStore) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	f.listPageCampaignID = campaignID
	f.listPageSize = pageSize
	f.listPageToken = pageToken
	return f.listPage, f.listPageErr
}

func TestCreateCampaignSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	store := &fakeCampaignStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign: store,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "camp-123", nil
		},
	}

	response, err := service.CreateCampaign(context.Background(), &campaignv1.CreateCampaignRequest{
		Name:        "  First Steps ",
		GmMode:      campaignv1.GmMode_HYBRID,
		ThemePrompt: "gentle hills",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if response == nil || response.Campaign == nil {
		t.Fatal("expected campaign response")
	}
	if response.Campaign.Id != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", response.Campaign.Id)
	}
	if response.Campaign.Name != "First Steps" {
		t.Fatalf("expected trimmed name, got %q", response.Campaign.Name)
	}
	if response.Campaign.GmMode != campaignv1.GmMode_HYBRID {
		t.Fatalf("expected hybrid gm mode, got %v", response.Campaign.GmMode)
	}
	if response.Campaign.ParticipantCount != 0 {
		t.Fatalf("expected 0 participant count, got %d", response.Campaign.ParticipantCount)
	}
	if response.Campaign.CharacterCount != 0 {
		t.Fatalf("expected 0 character count, got %d", response.Campaign.CharacterCount)
	}
	if response.Campaign.ThemePrompt != "gentle hills" {
		t.Fatalf("expected theme prompt preserved, got %q", response.Campaign.ThemePrompt)
	}
	if response.Campaign.CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Campaign.CreatedAt.AsTime())
	}
	if response.Campaign.UpdatedAt.AsTime() != fixedTime {
		t.Fatalf("expected updated_at %v, got %v", fixedTime, response.Campaign.UpdatedAt.AsTime())
	}
	if store.putCampaign.ID != "camp-123" {
		t.Fatalf("expected stored id camp-123, got %q", store.putCampaign.ID)
	}
}

func TestCreateCampaignValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  *campaignv1.CreateCampaignRequest
	}{
		{
			name: "empty name",
			req: &campaignv1.CreateCampaignRequest{
				Name:   "  ",
				GmMode: campaignv1.GmMode_HUMAN,
			},
		},
		{
			name: "missing gm mode",
			req: &campaignv1.CreateCampaignRequest{
				Name:   "Campaign",
				GmMode: campaignv1.GmMode_GM_MODE_UNSPECIFIED,
			},
		},
	}

	service := &CampaignService{
		stores: Stores{
			Campaign: &fakeCampaignStore{},
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "camp-1", nil },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateCampaign(context.Background(), tt.req)
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

func TestCreateCampaignNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.CreateCampaign(context.Background(), nil)
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

func TestCreateCampaignIDGenerationFailure(t *testing.T) {
	service := &CampaignService{
		stores: Stores{
			Campaign: &fakeCampaignStore{},
		},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "", errors.New("boom")
		},
	}

	_, err := service.CreateCampaign(context.Background(), &campaignv1.CreateCampaignRequest{
		Name:   "Campaign",
		GmMode: campaignv1.GmMode_HUMAN,
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

func TestCreateCampaignStoreFailure(t *testing.T) {
	store := &fakeCampaignStore{putErr: errors.New("boom")}
	service := &CampaignService{
		stores: Stores{
			Campaign: store,
		},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "camp-123", nil
		},
	}

	_, err := service.CreateCampaign(context.Background(), &campaignv1.CreateCampaignRequest{
		Name:   "Campaign",
		GmMode: campaignv1.GmMode_HUMAN,
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

func TestCreateCampaignMissingStore(t *testing.T) {
	service := &CampaignService{
		stores: Stores{},
		clock:  time.Now,
		idGenerator: func() (string, error) {
			return "camp-123", nil
		},
	}

	_, err := service.CreateCampaign(context.Background(), &campaignv1.CreateCampaignRequest{
		Name:   "Campaign",
		GmMode: campaignv1.GmMode_AI,
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

func TestListCampaignsDefaults(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	store := &fakeCampaignStore{
		listPage: storage.CampaignPage{
			Campaigns: []domain.Campaign{
				{
					ID:               "camp-10",
					Name:             "Wayfarers",
					GmMode:           domain.GmModeAI,
					ParticipantCount: 3,
					CharacterCount:   2,
					ThemePrompt:      "windswept",
					CreatedAt:        fixedTime,
					UpdatedAt:        fixedTime,
				},
			},
			NextPageToken: "camp-11",
		},
	}
	service := NewCampaignService(Stores{
		Campaign:    store,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	response, err := service.ListCampaigns(context.Background(), &campaignv1.ListCampaignsRequest{})
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if store.listSize != defaultListCampaignsPageSize {
		t.Fatalf("expected default page size %d, got %d", defaultListCampaignsPageSize, store.listSize)
	}
	if response.NextPageToken != "camp-11" {
		t.Fatalf("expected next page token, got %q", response.NextPageToken)
	}
	if len(response.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(response.Campaigns))
	}
	if response.Campaigns[0].Id != "camp-10" {
		t.Fatalf("expected id camp-10, got %q", response.Campaigns[0].Id)
	}
	if response.Campaigns[0].GmMode != campaignv1.GmMode_AI {
		t.Fatalf("expected gm mode AI, got %v", response.Campaigns[0].GmMode)
	}
	if response.Campaigns[0].CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Campaigns[0].CreatedAt.AsTime())
	}
}

func TestListCampaignsClampPageSize(t *testing.T) {
	store := &fakeCampaignStore{listPage: storage.CampaignPage{}}
	service := NewCampaignService(Stores{
		Campaign:    store,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.ListCampaigns(context.Background(), &campaignv1.ListCampaignsRequest{
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if store.listSize != maxListCampaignsPageSize {
		t.Fatalf("expected max page size %d, got %d", maxListCampaignsPageSize, store.listSize)
	}
}

func TestListCampaignsPassesToken(t *testing.T) {
	store := &fakeCampaignStore{listPage: storage.CampaignPage{}}
	service := NewCampaignService(Stores{
		Campaign:    store,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.ListCampaigns(context.Background(), &campaignv1.ListCampaignsRequest{
		PageSize:  1,
		PageToken: "next",
	})
	if err != nil {
		t.Fatalf("list campaigns: %v", err)
	}
	if store.listToken != "next" {
		t.Fatalf("expected page token next, got %q", store.listToken)
	}
}

func TestListCampaignsNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.ListCampaigns(context.Background(), nil)
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

func TestListCampaignsStoreFailure(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{listErr: errors.New("boom")},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.ListCampaigns(context.Background(), &campaignv1.ListCampaignsRequest{
		PageSize: 1,
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

func TestListCampaignsMissingStore(t *testing.T) {
	service := &CampaignService{}

	_, err := service.ListCampaigns(context.Background(), &campaignv1.ListCampaignsRequest{
		PageSize: 1,
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

func TestGetCampaignSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	store := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:               "camp-123",
			Name:             "Test Campaign",
			GmMode:           domain.GmModeHybrid,
			ParticipantCount: 5,
			CharacterCount:   3,
			ThemePrompt:      "fantasy adventure",
			CreatedAt:        fixedTime,
			UpdatedAt:        fixedTime,
		},
	}
	service := NewCampaignService(Stores{
		Campaign:    store,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	response, err := service.GetCampaign(context.Background(), &campaignv1.GetCampaignRequest{
		CampaignId: "camp-123",
	})
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if response == nil || response.Campaign == nil {
		t.Fatal("expected campaign response")
	}
	if response.Campaign.Id != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", response.Campaign.Id)
	}
	if response.Campaign.Name != "Test Campaign" {
		t.Fatalf("expected name Test Campaign, got %q", response.Campaign.Name)
	}
	if response.Campaign.GmMode != campaignv1.GmMode_HYBRID {
		t.Fatalf("expected hybrid gm mode, got %v", response.Campaign.GmMode)
	}
	if response.Campaign.ParticipantCount != 5 {
		t.Fatalf("expected 5 participant count, got %d", response.Campaign.ParticipantCount)
	}
	if response.Campaign.CharacterCount != 3 {
		t.Fatalf("expected 3 character count, got %d", response.Campaign.CharacterCount)
	}
	if response.Campaign.ThemePrompt != "fantasy adventure" {
		t.Fatalf("expected theme prompt fantasy adventure, got %q", response.Campaign.ThemePrompt)
	}
	if response.Campaign.CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Campaign.CreatedAt.AsTime())
	}
	if response.Campaign.UpdatedAt.AsTime() != fixedTime {
		t.Fatalf("expected updated_at %v, got %v", fixedTime, response.Campaign.UpdatedAt.AsTime())
	}
}

func TestGetCampaignNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetCampaign(context.Background(), nil)
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

func TestGetCampaignMissingStore(t *testing.T) {
	service := &CampaignService{}

	_, err := service.GetCampaign(context.Background(), &campaignv1.GetCampaignRequest{
		CampaignId: "camp-123",
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

func TestGetCampaignEmptyID(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:    &fakeCampaignStore{},
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetCampaign(context.Background(), &campaignv1.GetCampaignRequest{
		CampaignId: "  ",
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

func TestGetCampaignNotFound(t *testing.T) {
	store := &fakeCampaignStore{
		getErr: storage.ErrNotFound,
	}
	service := NewCampaignService(Stores{
		Campaign:    store,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetCampaign(context.Background(), &campaignv1.GetCampaignRequest{
		CampaignId: "camp-999",
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

func TestGetCampaignStoreError(t *testing.T) {
	store := &fakeCampaignStore{
		getErr: errors.New("database error"),
	}
	service := NewCampaignService(Stores{
		Campaign:    store,
		Participant: &fakeParticipantStore{},
		Character:   &fakeCharacterStore{},
	})

	_, err := service.GetCampaign(context.Background(), &campaignv1.GetCampaignRequest{
		CampaignId: "camp-123",
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

func TestGMModeToProto(t *testing.T) {
	tests := []struct {
		name   string
		gmMode domain.GmMode
		proto  campaignv1.GmMode
	}{
		{
			name:   "human",
			gmMode: domain.GmModeHuman,
			proto:  campaignv1.GmMode_HUMAN,
		},
		{
			name:   "ai",
			gmMode: domain.GmModeAI,
			proto:  campaignv1.GmMode_AI,
		},
		{
			name:   "hybrid",
			gmMode: domain.GmModeHybrid,
			proto:  campaignv1.GmMode_HYBRID,
		},
		{
			name:   "unspecified",
			gmMode: domain.GmModeUnspecified,
			proto:  campaignv1.GmMode_GM_MODE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := gmModeToProto(tt.gmMode)
			if proto != tt.proto {
				t.Fatalf("expected %v, got %v", tt.proto, proto)
			}
		})
	}
}
