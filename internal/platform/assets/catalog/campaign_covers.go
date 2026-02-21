package catalog

const (
	DomainCampaignCovers = "campaign-covers"
	CampaignCoverSetV1   = "campaign_cover_set_v1"
)

var campaignCoverManifest = campaignCoverManifestData

// CampaignCoverManifest returns the canonical campaign cover set definition.
func CampaignCoverManifest() Manifest {
	return copyManifest(campaignCoverManifest)
}

// CampaignCoverAssetIDs returns the stable ordered asset ids for v1 covers.
func CampaignCoverAssetIDs() []string {
	coverSet, ok := campaignCoverManifest.Sets[CampaignCoverSetV1]
	if !ok {
		return []string{}
	}
	return append([]string(nil), coverSet.AssetIDs...)
}
