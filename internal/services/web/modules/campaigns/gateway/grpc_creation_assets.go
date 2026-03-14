package gateway

import (
	"strconv"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// daggerheartAssetLookup provides keyed lookup for Daggerheart content asset refs.
type daggerheartAssetLookup map[string]*daggerheartv1.DaggerheartAssetRef

// daggerheartAssetLookupFromResponse indexes asset refs by entity and asset type.
func daggerheartAssetLookupFromResponse(resp *daggerheartv1.GetDaggerheartAssetMapResponse) daggerheartAssetLookup {
	lookup := daggerheartAssetLookup{}
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
		key := daggerheartAssetLookupKey(entityID, entityType, asset.GetType())
		lookup[key] = asset
	}
	return lookup
}

// get resolves one asset ref for an entity and requested asset type.
func (lookup daggerheartAssetLookup) get(entityID, entityType string, assetType daggerheartv1.DaggerheartAssetType) *daggerheartv1.DaggerheartAssetRef {
	if lookup == nil {
		return nil
	}
	return lookup[daggerheartAssetLookupKey(entityID, entityType, assetType)]
}

// daggerheartAssetLookupKey builds the canonical map key for one asset ref.
func daggerheartAssetLookupKey(entityID, entityType string, assetType daggerheartv1.DaggerheartAssetType) string {
	return strings.TrimSpace(entityID) + "\x00" + strings.ToLower(strings.TrimSpace(entityType)) + "\x00" + strconv.FormatInt(int64(assetType), 10)
}

// mapCatalogAssetReference projects one proto asset ref into the web catalog shape.
func mapCatalogAssetReference(assetBaseURL string, asset *daggerheartv1.DaggerheartAssetRef, deliveryWidthPX int) campaignapp.CatalogAssetReference {
	if asset == nil {
		return campaignapp.CatalogAssetReference{Status: "unavailable"}
	}
	return campaignapp.CatalogAssetReference{
		URL:     resolveDaggerheartAssetURL(assetBaseURL, asset.GetCdnAssetId(), deliveryWidthPX),
		Status:  daggerheartAssetStatusLabel(asset.GetStatus()),
		SetID:   strings.TrimSpace(asset.GetSetId()),
		AssetID: strings.TrimSpace(asset.GetAssetId()),
	}
}

// daggerheartAssetStatusLabel maps proto asset statuses to web labels.
func daggerheartAssetStatusLabel(status daggerheartv1.DaggerheartAssetStatus) string {
	switch status {
	case daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED:
		return "mapped"
	case daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_SET_DEFAULT:
		return "set_default"
	case daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_UNAVAILABLE:
		return "unavailable"
	default:
		return "unavailable"
	}
}

// resolveDaggerheartAssetURL resolves one CDN URL from a pre-resolved CDN asset id.
func resolveDaggerheartAssetURL(assetBaseURL, cdnAssetID string, deliveryWidthPX int) string {
	normalizedAssetID := strings.TrimSpace(cdnAssetID)
	if normalizedAssetID == "" {
		return ""
	}
	resolvedAssetURL, err := imagecdn.New(assetBaseURL).URL(imagecdn.Request{
		AssetID:   normalizedAssetID,
		Extension: ".png",
		Delivery:  &imagecdn.Delivery{WidthPX: deliveryWidthPX},
	})
	if err != nil {
		return ""
	}
	return resolvedAssetURL
}
