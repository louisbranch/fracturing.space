package campaigns

import (
	"net/url"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
)

// campaignThemePromptLimit keeps campaign cards compact and scan-friendly.
const campaignThemePromptLimit = 80

var campaignCoverManifest = catalog.CampaignCoverManifest()

// truncateCampaignTheme keeps card previews concise while preserving context.
func truncateCampaignTheme(themePrompt string) string {
	runes := []rune(strings.TrimSpace(themePrompt))
	if campaignThemePromptLimit <= 0 || len(runes) == 0 {
		return ""
	}
	if len(runes) <= campaignThemePromptLimit {
		return string(runes)
	}
	return string(runes[:campaignThemePromptLimit]) + "..."
}

// campaignCreatedAtUnixNano normalizes protobuf timestamps for deterministic sort order.
func campaignCreatedAtUnixNano(campaign *statev1.Campaign) int64 {
	if campaign == nil || campaign.GetCreatedAt() == nil {
		return 0
	}
	return campaign.GetCreatedAt().AsTime().UTC().UnixNano()
}

// campaignCoverImageURL resolves the final card image URL with a deterministic fallback.
func campaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID string) string {
	_, resolvedCoverAssetID := resolveCampaignCoverSelection(campaignID, coverSetID, coverAssetID)
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   resolvedCoverAssetID,
		Extension: ".png",
	})
	if err == nil {
		return resolvedAssetURL
	}
	// TODO(web-assets): this fallback assumes static campaign-cover files are served; verify assets exist or return a deterministic CDN-safe placeholder.
	return "/static/campaign-covers/" + url.PathEscape(resolvedCoverAssetID) + ".png"
}

// resolveCampaignCoverSelection ensures campaign cards always have a valid catalog asset.
func resolveCampaignCoverSelection(campaignID, coverSetID, coverAssetID string) (string, string) {
	resolvedCoverSetID, resolvedCoverAssetID, err := campaignCoverManifest.ResolveSelection(catalog.SelectionInput{
		EntityType: "campaign",
		EntityID:   strings.TrimSpace(campaignID),
		SetID:      coverSetID,
		AssetID:    coverAssetID,
	})
	if err == nil {
		return resolvedCoverSetID, resolvedCoverAssetID
	}

	fallbackCoverSetID, fallbackCoverAssetID, fallbackErr := campaignCoverManifest.ResolveSelection(catalog.SelectionInput{
		EntityType: "campaign",
		EntityID:   strings.TrimSpace(campaignID),
		SetID:      catalog.CampaignCoverSetV1,
		AssetID:    "",
	})
	if fallbackErr == nil {
		return fallbackCoverSetID, fallbackCoverAssetID
	}
	return catalog.CampaignCoverSetV1, defaultCampaignCoverAssetID()
}

// defaultCampaignCoverAssetID provides the final deterministic fallback for cover lookups.
func defaultCampaignCoverAssetID() string {
	coverSet, ok := campaignCoverManifest.Sets[catalog.CampaignCoverSetV1]
	if !ok || len(coverSet.AssetIDs) == 0 {
		return ""
	}
	return coverSet.AssetIDs[0]
}
