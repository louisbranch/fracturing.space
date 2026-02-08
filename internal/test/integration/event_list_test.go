//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func runEventListTests(t *testing.T, grpcAddr string) {
	t.Helper()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := statev1.NewCampaignServiceClient(conn)
	eventClient := statev1.NewEventServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create a campaign
	createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:   "Event List Test",
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: statev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := createResp.Campaign.Id
	lastSeq := requireEventTypesAfterSeq(t, ctx, eventClient, campaignID, 0, "campaign.created")

	// Append events to the campaign journal.
	for i := 0; i < 3; i++ {
		if _, err := eventClient.AppendEvent(ctx, &statev1.AppendEventRequest{
			CampaignId:  campaignID,
			Type:        "action.note_added",
			ActorType:   "system",
			EntityType:  "campaign",
			EntityId:    campaignID,
			PayloadJson: []byte("{}"),
		}); err != nil {
			t.Fatalf("append note event %d: %v", i, err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignID, lastSeq, "action.note_added")
		if _, err := eventClient.AppendEvent(ctx, &statev1.AppendEventRequest{
			CampaignId:  campaignID,
			Type:        "action.roll_resolved",
			ActorType:   "system",
			EntityType:  "campaign",
			EntityId:    campaignID,
			PayloadJson: []byte("{}"),
		}); err != nil {
			t.Fatalf("append roll event %d: %v", i, err)
		}
		lastSeq = requireEventTypesAfterSeq(t, ctx, eventClient, campaignID, lastSeq, "action.roll_resolved")
	}

	t.Run("list events basic", func(t *testing.T) {
		resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			t.Fatalf("list events: %v", err)
		}
		// 6 appended events (plus campaign.created).
		if len(resp.Events) < 6 {
			t.Errorf("expected at least 6 events, got %d", len(resp.Events))
		}
		// Verify ASC order (default)
		for i := 1; i < len(resp.Events); i++ {
			if resp.Events[i].Seq < resp.Events[i-1].Seq {
				t.Errorf("events not in ASC order: seq %d before seq %d",
					resp.Events[i-1].Seq, resp.Events[i].Seq)
			}
		}
	})

	t.Run("list events DESC order", func(t *testing.T) {
		resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			OrderBy:    "seq desc",
		})
		if err != nil {
			t.Fatalf("list events: %v", err)
		}
		// Verify DESC order
		for i := 1; i < len(resp.Events); i++ {
			if resp.Events[i].Seq > resp.Events[i-1].Seq {
				t.Errorf("events not in DESC order: seq %d before seq %d",
					resp.Events[i-1].Seq, resp.Events[i].Seq)
			}
		}
	})

	t.Run("pagination ASC", func(t *testing.T) {
		// Get first page
		page1, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
		})
		if err != nil {
			t.Fatalf("page 1: %v", err)
		}
		if len(page1.Events) != 3 {
			t.Fatalf("page 1: expected 3 events, got %d", len(page1.Events))
		}
		if page1.NextPageToken == "" {
			t.Fatal("page 1: expected next page token")
		}
		if page1.PreviousPageToken != "" {
			t.Error("page 1: expected no previous page token")
		}

		// Get second page
		page2, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
			PageToken:  page1.NextPageToken,
		})
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if len(page2.Events) != 3 {
			t.Fatalf("page 2: expected 3 events, got %d", len(page2.Events))
		}
		// Page 2 first event should be after page 1 last event
		if page2.Events[0].Seq <= page1.Events[2].Seq {
			t.Errorf("page 2 first seq %d should be > page 1 last seq %d",
				page2.Events[0].Seq, page1.Events[2].Seq)
		}
		if page2.PreviousPageToken == "" {
			t.Error("page 2: expected previous page token")
		}

		// Go back to page 1
		backToPage1, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
			PageToken:  page2.PreviousPageToken,
		})
		if err != nil {
			t.Fatalf("back to page 1: %v", err)
		}
		if len(backToPage1.Events) != 3 {
			t.Fatalf("back to page 1: expected 3 events, got %d", len(backToPage1.Events))
		}
		// Should match original page 1
		for i := 0; i < 3; i++ {
			if backToPage1.Events[i].Seq != page1.Events[i].Seq {
				t.Errorf("back to page 1: event %d seq mismatch: got %d, want %d",
					i, backToPage1.Events[i].Seq, page1.Events[i].Seq)
			}
		}
	})

	t.Run("pagination DESC", func(t *testing.T) {
		// Get first page (DESC = newest first)
		page1, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
			OrderBy:    "seq desc",
		})
		if err != nil {
			t.Fatalf("page 1: %v", err)
		}
		if len(page1.Events) != 3 {
			t.Fatalf("page 1: expected 3 events, got %d", len(page1.Events))
		}
		if page1.NextPageToken == "" {
			t.Fatal("page 1: expected next page token")
		}

		// Get second page
		page2, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
			OrderBy:    "seq desc",
			PageToken:  page1.NextPageToken,
		})
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if len(page2.Events) != 3 {
			t.Fatalf("page 2: expected 3 events, got %d", len(page2.Events))
		}
		// In DESC, page 2 first event should have LOWER seq than page 1 last event
		if page2.Events[0].Seq >= page1.Events[2].Seq {
			t.Errorf("DESC page 2 first seq %d should be < page 1 last seq %d",
				page2.Events[0].Seq, page1.Events[2].Seq)
		}
		if page2.PreviousPageToken == "" {
			t.Error("page 2: expected previous page token")
		}

		// Go back to page 1
		backToPage1, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
			OrderBy:    "seq desc",
			PageToken:  page2.PreviousPageToken,
		})
		if err != nil {
			t.Fatalf("back to page 1: %v", err)
		}
		if len(backToPage1.Events) != 3 {
			t.Fatalf("back to page 1: expected 3 events, got %d", len(backToPage1.Events))
		}
		// Should match original page 1
		for i := 0; i < 3; i++ {
			if backToPage1.Events[i].Seq != page1.Events[i].Seq {
				t.Errorf("back to page 1: event %d seq mismatch: got %d, want %d",
					i, backToPage1.Events[i].Seq, page1.Events[i].Seq)
			}
		}
	})

	t.Run("filter by event type", func(t *testing.T) {
		resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			Filter:     `type = "action.note_added"`,
		})
		if err != nil {
			t.Fatalf("list events with filter: %v", err)
		}
		if len(resp.Events) != 3 {
			t.Errorf("expected 3 note events, got %d", len(resp.Events))
		}
		for _, evt := range resp.Events {
			if evt.Type != "action.note_added" {
				t.Errorf("expected type action.note_added, got %s", evt.Type)
			}
		}
	})

	t.Run("invalid order_by", func(t *testing.T) {
		_, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			OrderBy:    "invalid",
		})
		if err == nil {
			t.Fatal("expected error for invalid order_by")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("token with changed order_by rejected", func(t *testing.T) {
		// Get a token with ASC order
		resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
		})
		if err != nil {
			t.Fatalf("get token: %v", err)
		}

		// Try to use with DESC order
		_, err = eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   3,
			OrderBy:    "seq desc",
			PageToken:  resp.NextPageToken,
		})
		if err == nil {
			t.Fatal("expected error when order_by changes")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("token with changed filter rejected", func(t *testing.T) {
		// Get a token with one filter
		resp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   2,
			Filter:     `type = "action.note_added"`,
		})
		if err != nil {
			t.Fatalf("get token: %v", err)
		}

		// Try to use with different filter
		_, err = eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   2,
			Filter:     `type = "action.roll_resolved"`,
			PageToken:  resp.NextPageToken,
		})
		if err == nil {
			t.Fatal("expected error when filter changes")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})
}
