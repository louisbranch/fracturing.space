package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPublicProfileRouteUsesConnectionsLookup(t *testing.T) {
	t.Parallel()

	connectionsClient := &fakeConnectionsClient{
		lookupUserProfileResp: &connectionsv1.LookupUserProfileResponse{
			UserProfileRecord: &connectionsv1.UserProfileRecord{
				Username: "alice",
				Name:     "Alice Adventurer",
				Bio:      "GM and worldbuilder.",
			},
		},
	}
	handler, err := NewHandlerWithCampaignAccess(
		Config{AuthBaseURL: "http://auth.local"},
		nil,
		handlerDependencies{connectionsClient: connectionsClient},
	)
	if err != nil {
		t.Fatalf("NewHandlerWithCampaignAccess: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/u/alice", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "Alice Adventurer") {
		t.Fatalf("expected display name in response, got %q", body)
	}
	if !strings.Contains(body, "GM and worldbuilder.") {
		t.Fatalf("expected bio in response, got %q", body)
	}
	if connectionsClient.lookupUserProfileReq == nil {
		t.Fatal("expected LookupUserProfile request")
	}
	if got := connectionsClient.lookupUserProfileReq.GetUsername(); got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
}

func TestPublicProfileRouteRejectsMissingUsername(t *testing.T) {
	t.Parallel()

	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/u/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}

func TestDiscoverRouteRendersListings(t *testing.T) {
	t.Parallel()

	listingClient := &fakeCampaignListingClient{
		listResp: &listingv1.ListCampaignListingsResponse{
			Listings: []*listingv1.CampaignListing{
				{
					CampaignId:            "camp-1",
					Title:                 "Skyfall Heist",
					Description:           "A city intrigue campaign.",
					ExpectedDurationLabel: "2-3 sessions",
				},
			},
		},
	}
	handler, err := NewHandlerWithCampaignAccess(
		Config{AuthBaseURL: "http://auth.local"},
		nil,
		handlerDependencies{listingClient: listingClient},
	)
	if err != nil {
		t.Fatalf("NewHandlerWithCampaignAccess: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "Skyfall Heist") {
		t.Fatalf("expected listing title in response, got %q", body)
	}
	if !strings.Contains(body, "/discover/campaigns/camp-1") {
		t.Fatalf("expected listing detail link in response, got %q", body)
	}
	if listingClient.listReq == nil {
		t.Fatal("expected ListCampaignListings request")
	}
}

func TestDiscoverCampaignRouteLoadsCampaignListing(t *testing.T) {
	t.Parallel()

	listingClient := &fakeCampaignListingClient{
		getResp: &listingv1.GetCampaignListingResponse{
			Listing: &listingv1.CampaignListing{
				CampaignId:            "camp-1",
				Title:                 "Skyfall Heist",
				Description:           "A city intrigue campaign.",
				ExpectedDurationLabel: "2-3 sessions",
			},
		},
	}
	handler, err := NewHandlerWithCampaignAccess(
		Config{AuthBaseURL: "http://auth.local"},
		nil,
		handlerDependencies{listingClient: listingClient},
	)
	if err != nil {
		t.Fatalf("NewHandlerWithCampaignAccess: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns/camp-1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "Skyfall Heist") {
		t.Fatalf("expected listing title in response, got %q", body)
	}
	if !strings.Contains(body, "2-3 sessions") {
		t.Fatalf("expected duration in response, got %q", body)
	}
	if listingClient.getReq == nil {
		t.Fatal("expected GetCampaignListing request")
	}
	if got := listingClient.getReq.GetCampaignId(); got != "camp-1" {
		t.Fatalf("campaign_id = %q, want %q", got, "camp-1")
	}
}

func TestDiscoverCampaignRouteRejectsMissingCampaignID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}

type fakeCampaignListingClient struct {
	listResp *listingv1.ListCampaignListingsResponse
	listErr  error
	listReq  *listingv1.ListCampaignListingsRequest

	getResp *listingv1.GetCampaignListingResponse
	getErr  error
	getReq  *listingv1.GetCampaignListingRequest
}

func (f *fakeCampaignListingClient) CreateCampaignListing(context.Context, *listingv1.CreateCampaignListingRequest, ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "fakeCampaignListingClient.CreateCampaignListing not implemented")
}

func (f *fakeCampaignListingClient) GetCampaignListing(ctx context.Context, req *listingv1.GetCampaignListingRequest, _ ...grpc.CallOption) (*listingv1.GetCampaignListingResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.getReq = req
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return nil, status.Error(codes.NotFound, "get campaign listing not configured")
}

func (f *fakeCampaignListingClient) ListCampaignListings(ctx context.Context, req *listingv1.ListCampaignListingsRequest, _ ...grpc.CallOption) (*listingv1.ListCampaignListingsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.listReq = req
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &listingv1.ListCampaignListingsResponse{}, nil
}
