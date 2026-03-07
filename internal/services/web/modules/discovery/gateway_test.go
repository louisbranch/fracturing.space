package discovery

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubGateway implements Gateway for tests that only need a healthy module.
type stubGateway struct{}

func (stubGateway) ListStarterListings(context.Context) ([]StarterListing, error) {
	return nil, nil
}

// listingClientStub records calls and returns canned responses.
type listingClientStub struct {
	resp *listingv1.ListCampaignListingsResponse
	err  error
}

func (s listingClientStub) ListCampaignListings(_ context.Context, _ *listingv1.ListCampaignListingsRequest, _ ...grpc.CallOption) (*listingv1.ListCampaignListingsResponse, error) {
	return s.resp, s.err
}

func TestNewGRPCGatewayReturnsUnavailableWhenNil(t *testing.T) {
	t.Parallel()

	gw := NewGRPCGateway(nil)
	if IsGatewayHealthy(gw) {
		t.Fatal("expected unhealthy gateway for nil client")
	}
	_, err := gw.ListStarterListings(context.Background())
	if err == nil {
		t.Fatal("expected error from unavailable gateway")
	}
	if apperrors.HTTPStatus(err) != 503 {
		t.Fatalf("status = %d, want 503", apperrors.HTTPStatus(err))
	}
}

func TestGRPCGatewayFiltersToStarterIntent(t *testing.T) {
	t.Parallel()

	client := listingClientStub{
		resp: &listingv1.ListCampaignListingsResponse{
			Listings: []*listingv1.CampaignListing{
				{
					CampaignId:                 "starter-1",
					Title:                      "Starter Adventure",
					Description:                "A beginner adventure",
					Intent:                     listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STARTER,
					DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
					GmMode:                     listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_AI,
					System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
					ExpectedDurationLabel:      "2-3 sessions",
					Tags:                       []string{"solo", "beginner"},
					Level:                      1,
					RecommendedParticipantsMin: 2,
					RecommendedParticipantsMax: 4,
				},
				{
					CampaignId: "standard-1",
					Title:      "Standard Campaign",
					Intent:     listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STANDARD,
				},
			},
		},
	}

	gw := NewGRPCGateway(client)
	results, err := gw.ListStarterListings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (only starter)", len(results))
	}

	got := results[0]
	if got.CampaignID != "starter-1" {
		t.Errorf("CampaignID = %q, want %q", got.CampaignID, "starter-1")
	}
	if got.Title != "Starter Adventure" {
		t.Errorf("Title = %q, want %q", got.Title, "Starter Adventure")
	}
	if got.Difficulty != "Beginner" {
		t.Errorf("Difficulty = %q, want %q", got.Difficulty, "Beginner")
	}
	if got.GmMode != "AI" {
		t.Errorf("GmMode = %q, want %q", got.GmMode, "AI")
	}
	if got.System != "Daggerheart" {
		t.Errorf("System = %q, want %q", got.System, "Daggerheart")
	}
	if got.Duration != "2-3 sessions" {
		t.Errorf("Duration = %q, want %q", got.Duration, "2-3 sessions")
	}
	if got.Players != "2-4" {
		t.Errorf("Players = %q, want %q", got.Players, "2-4")
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 tags", got.Tags)
	}
}

func TestGRPCGatewayMapsUnavailableError(t *testing.T) {
	t.Parallel()

	client := listingClientStub{
		err: status.Error(codes.Unavailable, "service down"),
	}
	gw := NewGRPCGateway(client)
	_, err := gw.ListStarterListings(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if apperrors.HTTPStatus(err) != 503 {
		t.Fatalf("status = %d, want 503", apperrors.HTTPStatus(err))
	}
}

func TestGRPCGatewayReturnsNilForEmptyResponse(t *testing.T) {
	t.Parallel()

	client := listingClientStub{resp: &listingv1.ListCampaignListingsResponse{}}
	gw := NewGRPCGateway(client)
	results, err := gw.ListStarterListings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("got %v, want nil", results)
	}
}

func TestIsGatewayHealthyNilGateway(t *testing.T) {
	t.Parallel()

	if IsGatewayHealthy(nil) {
		t.Fatal("expected nil gateway to be unhealthy")
	}
}

func TestPlayersLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		min, max int32
		want     string
	}{
		{0, 0, ""},
		{2, 4, "2-4"},
		{3, 3, "3"},
		{2, 0, "2+"},
		{0, 6, "up to 6"},
	}
	for _, tt := range tests {
		if got := playersLabel(tt.min, tt.max); got != tt.want {
			t.Errorf("playersLabel(%d, %d) = %q, want %q", tt.min, tt.max, got, tt.want)
		}
	}
}
