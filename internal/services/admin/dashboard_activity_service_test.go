package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type dashboardActivityCampaignClient struct {
	listResponse *statev1.ListCampaignsResponse
	listErr      error
}

func (c *dashboardActivityCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResponse != nil {
		return c.listResponse, nil
	}
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *dashboardActivityCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityCampaignClient) EndCampaign(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityCampaignClient) ArchiveCampaign(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityCampaignClient) RestoreCampaign(context.Context, *statev1.RestoreCampaignRequest, ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityCampaignClient) SetCampaignCover(context.Context, *statev1.SetCampaignCoverRequest, ...grpc.CallOption) (*statev1.SetCampaignCoverResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type dashboardActivityEventClient struct {
	responses map[string]*statev1.ListEventsResponse
	errs      map[string]error
}

func (c *dashboardActivityEventClient) AppendEvent(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityEventClient) ListEvents(_ context.Context, in *statev1.ListEventsRequest, _ ...grpc.CallOption) (*statev1.ListEventsResponse, error) {
	campaignID := in.GetCampaignId()
	if err := c.errs[campaignID]; err != nil {
		return nil, err
	}
	if resp, ok := c.responses[campaignID]; ok {
		return resp, nil
	}
	return &statev1.ListEventsResponse{}, nil
}

func (c *dashboardActivityEventClient) ListTimelineEntries(context.Context, *statev1.ListTimelineEntriesRequest, ...grpc.CallOption) (*statev1.ListTimelineEntriesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *dashboardActivityEventClient) SubscribeCampaignUpdates(context.Context, *statev1.SubscribeCampaignUpdatesRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[statev1.CampaignUpdate], error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestDashboardActivityServiceListRecentSortsAndLimits(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	events := make([]*statev1.Event, 0, 20)
	for i := 0; i < 20; i++ {
		events = append(events, &statev1.Event{
			CampaignId: "camp-1",
			Type:       "campaign.updated",
			Ts:         timestamppb.New(now.Add(-time.Duration(i) * time.Minute)),
		})
	}
	// Feed oldest-first to prove sort order is applied by the service.
	for left, right := 0, len(events)-1; left < right; left, right = left+1, right-1 {
		events[left], events[right] = events[right], events[left]
	}

	service := newDashboardActivityService(
		&dashboardActivityCampaignClient{
			listResponse: &statev1.ListCampaignsResponse{
				Campaigns: []*statev1.Campaign{{Id: "camp-1", Name: "Alpha"}},
			},
		},
		&dashboardActivityEventClient{
			responses: map[string]*statev1.ListEventsResponse{
				"camp-1": {Events: events},
			},
		},
	)

	records := service.listRecent(context.Background())
	if len(records) != 15 {
		t.Fatalf("record count = %d, want 15", len(records))
	}
	if records[0].campaignName != "Alpha" {
		t.Fatalf("campaign name = %q, want %q", records[0].campaignName, "Alpha")
	}
	for i := 1; i < len(records); i++ {
		previous := records[i-1].event.GetTs().AsTime()
		current := records[i].event.GetTs().AsTime()
		if current.After(previous) {
			t.Fatalf("records not sorted desc at index %d", i)
		}
	}
}

func TestDashboardActivityServiceListRecentSkipsFailedCampaignEventLoads(t *testing.T) {
	service := newDashboardActivityService(
		&dashboardActivityCampaignClient{
			listResponse: &statev1.ListCampaignsResponse{
				Campaigns: []*statev1.Campaign{
					{Id: "camp-error", Name: "Error Campaign"},
					{Id: "camp-ok", Name: "OK Campaign"},
				},
			},
		},
		&dashboardActivityEventClient{
			errs: map[string]error{"camp-error": errors.New("boom")},
			responses: map[string]*statev1.ListEventsResponse{
				"camp-ok": {
					Events: []*statev1.Event{
						{
							CampaignId: "camp-ok",
							Type:       "campaign.updated",
							Ts:         timestamppb.New(time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)),
						},
					},
				},
			},
		},
	)

	records := service.listRecent(context.Background())
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].campaignName != "OK Campaign" {
		t.Fatalf("campaign name = %q, want %q", records[0].campaignName, "OK Campaign")
	}
}

func TestDashboardActivityServiceListRecentWithMissingClients(t *testing.T) {
	if records := newDashboardActivityService(nil, nil).listRecent(context.Background()); len(records) != 0 {
		t.Fatalf("record count = %d, want 0", len(records))
	}
}

func TestDashboardActivityServiceListRecentWithCampaignListError(t *testing.T) {
	service := newDashboardActivityService(
		&dashboardActivityCampaignClient{listErr: errors.New("list failed")},
		&dashboardActivityEventClient{},
	)
	if records := service.listRecent(context.Background()); len(records) != 0 {
		t.Fatalf("record count = %d, want 0", len(records))
	}
}
