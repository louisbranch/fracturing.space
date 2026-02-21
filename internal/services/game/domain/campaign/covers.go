package campaign

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

const defaultCampaignCoverSetID = catalog.CampaignCoverSetV1

var campaignCoverManifest = catalog.CampaignCoverManifest()

var campaignCoverAssetCatalog = campaignCoverManifest.Sets[defaultCampaignCoverSetID].AssetIDs

func isCampaignCoverAssetID(raw string) bool {
	_, ok := normalizeCampaignCoverAssetID(raw)
	return ok
}

func normalizeCampaignCoverAssetID(raw string) (string, bool) {
	coverAssetID := campaignCoverManifest.NormalizeAssetID(raw)
	if coverAssetID == "" {
		return "", false
	}
	if !campaignCoverManifest.ValidateAssetInSet(defaultCampaignCoverSetID, coverAssetID) {
		return "", false
	}
	return coverAssetID, true
}

func defaultCampaignCoverAssetID(campaignID string) string {
	if len(campaignCoverAssetCatalog) == 0 {
		return ""
	}

	trimmedCampaignID := strings.TrimSpace(campaignID)
	if trimmedCampaignID == "" {
		return campaignCoverAssetCatalog[0]
	}

	coverAssetID, err := campaignCoverManifest.DeterministicAsset(catalog.PickerInput{
		EntityType: "campaign",
		EntityID:   trimmedCampaignID,
		SetID:      defaultCampaignCoverSetID,
	})
	if err != nil || strings.TrimSpace(coverAssetID) == "" {
		return campaignCoverAssetCatalog[0]
	}
	return coverAssetID
}

func normalizeCampaignCoverSetID(raw string) (string, bool) {
	return campaignCoverManifest.NormalizeSetID(raw)
}
