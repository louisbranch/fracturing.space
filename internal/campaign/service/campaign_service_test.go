package service

import (
	"context"
	"errors"
	"testing"
	"time"

	campaignpb "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCampaignStore struct {
	putCampaign domain.Campaign
	putErr      error
}

func (f *fakeCampaignStore) Put(ctx context.Context, campaign domain.Campaign) error {
	f.putCampaign = campaign
	return f.putErr
}

func (f *fakeCampaignStore) Get(ctx context.Context, id string) (domain.Campaign, error) {
	return domain.Campaign{}, storage.ErrNotFound
}

func TestCreateCampaignSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	store := &fakeCampaignStore{}
	service := &CampaignService{
		store: store,
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "camp-123", nil
		},
	}

	response, err := service.CreateCampaign(context.Background(), &campaignpb.CreateCampaignRequest{
		Name:        "  First Steps ",
		GmMode:      campaignpb.GmMode_HYBRID,
		PlayerSlots: 5,
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
	if response.Campaign.GmMode != campaignpb.GmMode_HYBRID {
		t.Fatalf("expected hybrid gm mode, got %v", response.Campaign.GmMode)
	}
	if response.Campaign.PlayerSlots != 5 {
		t.Fatalf("expected 5 player slots, got %d", response.Campaign.PlayerSlots)
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
		req  *campaignpb.CreateCampaignRequest
	}{
		{
			name: "empty name",
			req: &campaignpb.CreateCampaignRequest{
				Name:        "  ",
				GmMode:      campaignpb.GmMode_HUMAN,
				PlayerSlots: 1,
			},
		},
		{
			name: "missing gm mode",
			req: &campaignpb.CreateCampaignRequest{
				Name:        "Campaign",
				GmMode:      campaignpb.GmMode_GM_MODE_UNSPECIFIED,
				PlayerSlots: 1,
			},
		},
		{
			name: "invalid player slots",
			req: &campaignpb.CreateCampaignRequest{
				Name:        "Campaign",
				GmMode:      campaignpb.GmMode_AI,
				PlayerSlots: 0,
			},
		},
	}

	service := &CampaignService{
		store:       &fakeCampaignStore{},
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
	service := NewCampaignService(&fakeCampaignStore{})

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
		store: &fakeCampaignStore{},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "", errors.New("boom")
		},
	}

	_, err := service.CreateCampaign(context.Background(), &campaignpb.CreateCampaignRequest{
		Name:        "Campaign",
		GmMode:      campaignpb.GmMode_HUMAN,
		PlayerSlots: 2,
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
		store: store,
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "camp-123", nil
		},
	}

	_, err := service.CreateCampaign(context.Background(), &campaignpb.CreateCampaignRequest{
		Name:        "Campaign",
		GmMode:      campaignpb.GmMode_HUMAN,
		PlayerSlots: 2,
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
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "camp-123", nil
		},
	}

	_, err := service.CreateCampaign(context.Background(), &campaignpb.CreateCampaignRequest{
		Name:        "Campaign",
		GmMode:      campaignpb.GmMode_AI,
		PlayerSlots: 2,
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
