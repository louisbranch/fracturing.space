package contenttransport

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

// assetTypeToProto maps manifest asset kinds onto the transport enum exposed
// by the asset-map endpoint.
func assetTypeToProto(assetType string) pb.DaggerheartAssetType {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case catalog.DaggerheartAssetTypeClassIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION
	case catalog.DaggerheartAssetTypeClassIcon:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ICON
	case catalog.DaggerheartAssetTypeSubclassIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_SUBCLASS_ILLUSTRATION
	case catalog.DaggerheartAssetTypeAncestryIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ANCESTRY_ILLUSTRATION
	case catalog.DaggerheartAssetTypeCommunityIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_COMMUNITY_ILLUSTRATION
	case catalog.DaggerheartAssetTypeDomainIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ILLUSTRATION
	case catalog.DaggerheartAssetTypeDomainIcon:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ICON
	case catalog.DaggerheartAssetTypeDomainCardIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_CARD_ILLUSTRATION
	case catalog.DaggerheartAssetTypeAdversaryIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ADVERSARY_ILLUSTRATION
	case catalog.DaggerheartAssetTypeEnvironmentIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ENVIRONMENT_ILLUSTRATION
	case catalog.DaggerheartAssetTypeWeaponIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_WEAPON_ILLUSTRATION
	case catalog.DaggerheartAssetTypeArmorIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ARMOR_ILLUSTRATION
	case catalog.DaggerheartAssetTypeItemIllustration:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ITEM_ILLUSTRATION
	default:
		return pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_UNSPECIFIED
	}
}

// assetStatusToProto maps manifest resolution outcomes onto the transport enum
// exposed by the asset-map endpoint.
func assetStatusToProto(status catalog.DaggerheartAssetResolutionStatus) pb.DaggerheartAssetStatus {
	switch status {
	case catalog.DaggerheartAssetResolutionStatusMapped:
		return pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED
	case catalog.DaggerheartAssetResolutionStatusSetDefault:
		return pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_SET_DEFAULT
	case catalog.DaggerheartAssetResolutionStatusUnavailable:
		return pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_UNAVAILABLE
	default:
		return pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_UNSPECIFIED
	}
}

// fallbackString preserves manifest values when present and supplies transport
// defaults when the manifest omits them.
func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// resolveDaggerheartAssetMapLocale ensures the asset-map response always emits
// a supported locale, defaulting when the request is unspecified or unknown.
func resolveDaggerheartAssetMapLocale(locale commonv1.Locale) commonv1.Locale {
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return defaultDaggerheartAssetMapLocale
	}
	normalized := i18n.NormalizeLocale(locale)
	if normalized == commonv1.Locale_LOCALE_UNSPECIFIED {
		return defaultDaggerheartAssetMapLocale
	}
	return normalized
}
