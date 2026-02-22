package server

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer_CreateGetAndListCampaignListingsRoundTrip(t *testing.T) {
	dbPath := t.TempDir() + "/listing.db"
	t.Setenv("FRACTURING_SPACE_LISTING_DB_PATH", dbPath)

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(runCtx)
	}()
	t.Cleanup(func() {
		runCancel()
		select {
		case serveErr := <-serveDone:
			if serveErr != nil {
				t.Fatalf("serve: %v", serveErr)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for server shutdown")
		}
	})

	conn, err := grpc.NewClient(srv.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial listing server: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close gRPC connection: %v", closeErr)
		}
	})

	client := listingv1.NewCampaignListingServiceClient(conn)

	createResp, err := client.CreateCampaignListing(context.Background(), &listingv1.CreateCampaignListingRequest{
		CampaignId:                 "camp-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	if err != nil {
		t.Fatalf("create campaign listing: %v", err)
	}
	if got := createResp.GetListing().GetCampaignId(); got != "camp-1" {
		t.Fatalf("campaign_id = %q, want camp-1", got)
	}

	getResp, err := client.GetCampaignListing(context.Background(), &listingv1.GetCampaignListingRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("get campaign listing: %v", err)
	}
	if got := getResp.GetListing().GetTitle(); got != "Sunfall" {
		t.Fatalf("title = %q, want Sunfall", got)
	}

	listResp, err := client.ListCampaignListings(context.Background(), &listingv1.ListCampaignListingsRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list campaign listings: %v", err)
	}
	if len(listResp.GetListings()) != 1 {
		t.Fatalf("listings len = %d, want 1", len(listResp.GetListings()))
	}
}
