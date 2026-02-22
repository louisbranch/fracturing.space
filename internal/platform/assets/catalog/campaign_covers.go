package catalog

const (
	DomainCampaignCovers = "campaign-covers"
	CampaignCoverSetV1   = "campaign_cover_set_v1"
)

// CampaignCoverManifest returns the canonical campaign cover set definition.
func CampaignCoverManifest() Manifest {
	coverManifest, _, _, err := EmbeddedCatalogManifests()
	if err != nil {
		return Manifest{}
	}
	return copyManifest(coverManifest)
}

// CampaignCoverAssetIDs returns the stable ordered asset ids for v1 covers.
func CampaignCoverAssetIDs() []string {
	manifest, _, _, err := EmbeddedCatalogManifests()
	if err != nil {
		return []string{}
	}
	coverSet, ok := manifest.Sets[CampaignCoverSetV1]
	if !ok {
		return []string{}
	}
	return append([]string(nil), coverSet.AssetIDs...)
}
