package contenttransport

import (
	"context"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

const (
	defaultDaggerheartAssetMapID            = "daggerheart-assets-v1"
	defaultDaggerheartAssetMapSystemID      = "daggerheart"
	defaultDaggerheartAssetMapSystemVersion = "v1"
	defaultDaggerheartAssetMapLocale        = commonv1.Locale_LOCALE_EN_US
)

// daggerheartAssetDescriptor identifies one entity-to-asset lookup performed
// against the published Daggerheart asset manifest.
type daggerheartAssetDescriptor struct {
	EntityType string
	EntityID   string
	AssetType  string
}

// buildDaggerheartAssetMap resolves the published asset selectors for the
// current Daggerheart catalog content.
func buildDaggerheartAssetMap(ctx context.Context, store contentstore.DaggerheartContentReadStore, requestedLocale commonv1.Locale) (*pb.DaggerheartAssetMap, error) {
	content, err := loadDaggerheartAssetMapContent(ctx, store)
	if err != nil {
		return nil, err
	}

	resolvedLocale := resolveDaggerheartAssetMapLocale(requestedLocale)

	descriptors := collectDaggerheartAssetDescriptors(
		content.classes,
		content.subclasses,
		content.heritages,
		content.domains,
		content.domainCards,
		content.adversaries,
		content.environments,
		content.weapons,
		content.armor,
		content.items,
	)

	assetManifest := catalog.DaggerheartAssetsManifest()
	assets := make([]*pb.DaggerheartAssetRef, 0, len(descriptors))
	for _, descriptor := range descriptors {
		resolved := assetManifest.ResolveEntityAsset(descriptor.EntityType, descriptor.EntityID, descriptor.AssetType)
		assets = append(assets, &pb.DaggerheartAssetRef{
			Type:       assetTypeToProto(descriptor.AssetType),
			Status:     assetStatusToProto(resolved.Status),
			EntityType: descriptor.EntityType,
			EntityId:   descriptor.EntityID,
			SetId:      strings.TrimSpace(resolved.SetID),
			AssetId:    strings.TrimSpace(resolved.AssetID),
			CdnAssetId: strings.TrimSpace(resolved.CDNAssetID),
		})
	}

	return &pb.DaggerheartAssetMap{
		Id:            fallbackString(strings.TrimSpace(assetManifest.ID), defaultDaggerheartAssetMapID),
		SystemId:      fallbackString(strings.TrimSpace(assetManifest.SystemID), defaultDaggerheartAssetMapSystemID),
		SystemVersion: fallbackString(strings.TrimSpace(assetManifest.SystemVersion), defaultDaggerheartAssetMapSystemVersion),
		Locale:        resolvedLocale,
		Theme:         strings.TrimSpace(assetManifest.Theme),
		Assets:        assets,
	}, nil
}
