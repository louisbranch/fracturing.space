package admin

import (
	"context"
	"sort"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

type dashboardActivityRecord struct {
	event        *statev1.Event
	campaignName string
}

type dashboardActivityService struct {
	campaignClient statev1.CampaignServiceClient
	eventClient    statev1.EventServiceClient
}

func newDashboardActivityService(
	campaignClient statev1.CampaignServiceClient,
	eventClient statev1.EventServiceClient,
) dashboardActivityService {
	return dashboardActivityService{
		campaignClient: campaignClient,
		eventClient:    eventClient,
	}
}

func (s dashboardActivityService) listRecent(ctx context.Context) []dashboardActivityRecord {
	if s.campaignClient == nil || s.eventClient == nil {
		return nil
	}
	campaignsResp, err := s.campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil || campaignsResp == nil {
		return nil
	}

	records := make([]dashboardActivityRecord, 0)
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
			records = append(records, dashboardActivityRecord{
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
