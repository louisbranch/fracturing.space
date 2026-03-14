package app

import (
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
)

// campaignThemePromptLimit keeps campaign cards compact and scan-friendly.
const campaignThemePromptLimit = 80
const campaignCoverFallbackPath = "/static/campaign-cover-fallback.svg"
const campaignCoverCardDeliveryWidthPX = 640
const campaignCoverBackgroundPreviewWidthPX = 128
const campaignCoverBackgroundFullWidthPX = 1600

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

// TruncateCampaignTheme keeps card previews concise while preserving context.
func TruncateCampaignTheme(themePrompt string) string {
	return truncateCampaignTheme(themePrompt)
}

// campaignCoverImageURL resolves one campaign cover URL for a web slot with deterministic fallback.
func campaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID string, deliveryWidthPX int) string {
	resolvedCoverSetID, resolvedCoverAssetID := resolveCampaignCoverSelection(campaignID, coverSetID, coverAssetID)
	resolvedCDNAssetID := catalog.ResolveCDNAssetID(resolvedCoverSetID, resolvedCoverAssetID)
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   resolvedCDNAssetID,
		Extension: ".png",
		Delivery:  &imagecdn.Delivery{WidthPX: deliveryWidthPX},
	})
	if err == nil {
		return resolvedAssetURL
	}
	fallbackURL := campaignCoverFallbackPath
	if trimmedAssetID := strings.TrimSpace(resolvedCoverAssetID); trimmedAssetID != "" {
		fallbackURL += "?asset_id=" + url.QueryEscape(trimmedAssetID)
	}
	return fallbackURL
}

// CampaignCoverImageURL resolves the campaign-list card image URL with deterministic fallback.
func CampaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID string) string {
	return campaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID, campaignCoverCardDeliveryWidthPX)
}

// CampaignCoverPreviewImageURL resolves the low-resolution workspace background preview URL.
func CampaignCoverPreviewImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID string) string {
	return campaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID, campaignCoverBackgroundPreviewWidthPX)
}

// CampaignCoverBackgroundImageURL resolves the full workspace background URL.
func CampaignCoverBackgroundImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID string) string {
	return campaignCoverImageURL(assetBaseURL, campaignID, coverSetID, coverAssetID, campaignCoverBackgroundFullWidthPX)
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
