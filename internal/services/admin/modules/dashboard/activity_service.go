package dashboard

import (
	"context"
	"sort"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// activityRecord captures a single recent event with its campaign name.
type activityRecord struct {
	event        *statev1.Event
	campaignName string
}

// activityService loads and sorts recent dashboard activity across campaigns.
type activityService struct {
	campaignClient statev1.CampaignServiceClient
	eventClient    statev1.EventServiceClient
}

// newActivityService builds an activity loader from campaign/event clients.
func newActivityService(
	campaignClient statev1.CampaignServiceClient,
	eventClient statev1.EventServiceClient,
) activityService {
	return activityService{
		campaignClient: campaignClient,
		eventClient:    eventClient,
	}
}

// listRecent returns the newest activity records across campaigns.
func (s activityService) listRecent(ctx context.Context) []activityRecord {
	campaignsResp, err := s.campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil || campaignsResp == nil {
		return nil
	}

	records := make([]activityRecord, 0)
	for _, campaign := range campaignsResp.GetCampaigns() {
		if campaign == nil {
			continue
		}
		eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaign.GetId(),
			PageSize:   5,
			OrderBy:    "seq desc",
		})
		if err != nil || eventsResp == nil {
			continue
		}
		for _, evt := range eventsResp.GetEvents() {
			if evt == nil {
				continue
			}
			records = append(records, activityRecord{
				event:        evt,
				campaignName: campaign.GetName(),
			})
		}
	}

	sort.SliceStable(records, func(i, j int) bool {
		iTS := records[i].event.GetTs()
		jTS := records[j].event.GetTs()
		if iTS == nil && jTS == nil {
			return false
		}
		if iTS == nil {
			return false
		}
		if jTS == nil {
			return true
		}
		return iTS.AsTime().After(jTS.AsTime())
	})

	if len(records) > 15 {
		records = records[:15]
	}
	return records
}
