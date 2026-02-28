package campaigns

import statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"

// campaignCreatedAtUnixNano normalizes protobuf timestamps for deterministic sort order.
func campaignCreatedAtUnixNano(campaign *statev1.Campaign) int64 {
	if campaign == nil || campaign.GetCreatedAt() == nil {
		return 0
	}
	return campaign.GetCreatedAt().AsTime().UTC().UnixNano()
}

func campaignUpdatedAtUnixNano(campaign *statev1.Campaign) int64 {
	if campaign == nil {
		return 0
	}
	if campaign.GetUpdatedAt() != nil {
		return campaign.GetUpdatedAt().AsTime().UTC().UnixNano()
	}
	if campaign.GetCreatedAt() == nil {
		return 0
	}
	return campaign.GetCreatedAt().AsTime().UTC().UnixNano()
}
