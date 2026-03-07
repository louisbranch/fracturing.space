package gateway

import (
	"strconv"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// daggerheartContentAssetLookup provides keyed lookup for Daggerheart content asset refs.
type daggerheartContentAssetLookup map[string]*daggerheartv1.DaggerheartContentAssetRef

// daggerheartContentAssetLookupFromResponse indexes asset refs by entity and asset type.
func daggerheartContentAssetLookupFromResponse(resp *daggerheartv1.GetDaggerheartContentAssetMapResponse) daggerheartContentAssetLookup {
	lookup := daggerheartContentAssetLookup{}
	if resp == nil || resp.GetAssetMap() == nil {
		return lookup
	}
	for _, asset := range resp.GetAssetMap().GetAssets() {
		if asset == nil {
			continue
		}
		entityID := strings.TrimSpace(asset.GetEntityId())
		entityType := strings.TrimSpace(asset.GetEntityType())
		if entityID == "" || entityType == "" {
			continue
		}
		key := daggerheartContentAssetLookupKey(entityID, entityType, asset.GetType())
		lookup[key] = asset
	}
	return lookup
}

// get resolves one asset ref for an entity and requested asset type.
func (lookup daggerheartContentAssetLookup) get(entityID, entityType string, assetType daggerheartv1.DaggerheartContentAssetType) *daggerheartv1.DaggerheartContentAssetRef {
	if lookup == nil {
		return nil
	}
	return lookup[daggerheartContentAssetLookupKey(entityID, entityType, assetType)]
}

// daggerheartContentAssetLookupKey builds the canonical map key for one asset ref.
func daggerheartContentAssetLookupKey(entityID, entityType string, assetType daggerheartv1.DaggerheartContentAssetType) string {
	return strings.TrimSpace(entityID) + "\x00" + strings.ToLower(strings.TrimSpace(entityType)) + "\x00" + strconv.FormatInt(int64(assetType), 10)
}

// mapCatalogAssetReference projects one proto asset ref into the web catalog shape.
func mapCatalogAssetReference(assetBaseURL string, asset *daggerheartv1.DaggerheartContentAssetRef) campaignapp.CatalogAssetReference {
	if asset == nil {
		return campaignapp.CatalogAssetReference{Status: "unavailable"}
	}
	return campaignapp.CatalogAssetReference{
		URL:     resolveDaggerheartContentAssetURL(assetBaseURL, asset.GetCdnAssetId()),
		Status:  daggerheartContentAssetStatusLabel(asset.GetStatus()),
		SetID:   strings.TrimSpace(asset.GetSetId()),
		AssetID: strings.TrimSpace(asset.GetAssetId()),
	}
}

// daggerheartContentAssetStatusLabel maps proto asset statuses to web labels.
func daggerheartContentAssetStatusLabel(status daggerheartv1.DaggerheartContentAssetResolutionStatus) string {
	switch status {
	case daggerheartv1.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_MAPPED:
		return "mapped"
	case daggerheartv1.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_SET_DEFAULT:
		return "set_default"
	case daggerheartv1.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_UNAVAILABLE:
		return "unavailable"
	default:
		return "unavailable"
	}
}

// resolveDaggerheartContentAssetURL resolves one CDN URL from a pre-resolved CDN asset id.
func resolveDaggerheartContentAssetURL(assetBaseURL, cdnAssetID string) string {
	normalizedAssetID := strings.TrimSpace(cdnAssetID)
	if normalizedAssetID == "" {
		return ""
	}
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   normalizedAssetID,
		Extension: ".png",
	})
	if err != nil {
		return ""
	}
	return resolvedAssetURL
}
