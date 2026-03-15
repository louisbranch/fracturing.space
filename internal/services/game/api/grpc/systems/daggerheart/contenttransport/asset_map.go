package contenttransport

import (
	"context"
	"fmt"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

const (
	defaultDaggerheartAssetMapID            = "daggerheart-assets-v1"
	defaultDaggerheartAssetMapSystemID      = "daggerheart"
	defaultDaggerheartAssetMapSystemVersion = "v1"
	defaultDaggerheartAssetMapLocale        = commonv1.Locale_LOCALE_EN_US
)

type daggerheartAssetDescriptor struct {
	EntityType string
	EntityID   string
	AssetType  string
}

func buildDaggerheartAssetMap(ctx context.Context, store contentstore.DaggerheartContentReadStore, requestedLocale commonv1.Locale) (*pb.DaggerheartAssetMap, error) {
	classes, err := store.ListDaggerheartClasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	subclasses, err := store.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subclasses: %w", err)
	}
	heritages, err := store.ListDaggerheartHeritages(ctx)
	if err != nil {
		return nil, fmt.Errorf("list heritages: %w", err)
	}
	domains, err := store.ListDaggerheartDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	domainCards, err := store.ListDaggerheartDomainCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list domain cards: %w", err)
	}
	adversaries, err := store.ListDaggerheartAdversaryEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list adversaries: %w", err)
	}
	environments, err := store.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	weapons, err := store.ListDaggerheartWeapons(ctx)
	if err != nil {
		return nil, fmt.Errorf("list weapons: %w", err)
	}
	armor, err := store.ListDaggerheartArmor(ctx)
	if err != nil {
		return nil, fmt.Errorf("list armor: %w", err)
	}
	items, err := store.ListDaggerheartItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}

	resolvedLocale := resolveDaggerheartAssetMapLocale(requestedLocale)

	descriptors := collectDaggerheartAssetDescriptors(
		classes,
		subclasses,
		heritages,
		domains,
		domainCards,
		adversaries,
		environments,
		weapons,
		armor,
		items,
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

func collectDaggerheartAssetDescriptors(
	classes []contentstore.DaggerheartClass,
	subclasses []contentstore.DaggerheartSubclass,
	heritages []contentstore.DaggerheartHeritage,
	domains []contentstore.DaggerheartDomain,
	domainCards []contentstore.DaggerheartDomainCard,
	adversaries []contentstore.DaggerheartAdversaryEntry,
	environments []contentstore.DaggerheartEnvironment,
	weapons []contentstore.DaggerheartWeapon,
	armor []contentstore.DaggerheartArmor,
	items []contentstore.DaggerheartItem,
) []daggerheartAssetDescriptor {
	descriptors := make([]daggerheartAssetDescriptor, 0, len(classes)*2+len(subclasses)+len(heritages)+len(domains)*2+len(domainCards)+len(adversaries)+len(environments)+len(weapons)+len(armor)+len(items))
	seen := map[string]struct{}{}

	for _, class := range classes {
		entityID := strings.TrimSpace(class.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeClass, entityID, catalog.DaggerheartAssetTypeClassIllustration)
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeClass, entityID, catalog.DaggerheartAssetTypeClassIcon)
	}
	for _, subclass := range subclasses {
		entityID := strings.TrimSpace(subclass.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeSubclass, entityID, catalog.DaggerheartAssetTypeSubclassIllustration)
	}
	for _, heritage := range heritages {
		entityID := strings.TrimSpace(heritage.ID)
		if entityID == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(heritage.Kind)) {
		case catalog.DaggerheartEntityTypeAncestry:
			appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeAncestry, entityID, catalog.DaggerheartAssetTypeAncestryIllustration)
		case catalog.DaggerheartEntityTypeCommunity:
			appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeCommunity, entityID, catalog.DaggerheartAssetTypeCommunityIllustration)
		}
	}
	for _, domain := range domains {
		entityID := strings.TrimSpace(domain.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeDomain, entityID, catalog.DaggerheartAssetTypeDomainIllustration)
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeDomain, entityID, catalog.DaggerheartAssetTypeDomainIcon)
	}
	for _, domainCard := range domainCards {
		entityID := strings.TrimSpace(domainCard.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeDomainCard, entityID, catalog.DaggerheartAssetTypeDomainCardIllustration)
	}
	for _, adversary := range adversaries {
		entityID := strings.TrimSpace(adversary.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeAdversary, entityID, catalog.DaggerheartAssetTypeAdversaryIllustration)
	}
	for _, environment := range environments {
		entityID := strings.TrimSpace(environment.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeEnvironment, entityID, catalog.DaggerheartAssetTypeEnvironmentIllustration)
	}
	for _, weapon := range weapons {
		entityID := strings.TrimSpace(weapon.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeWeapon, entityID, catalog.DaggerheartAssetTypeWeaponIllustration)
	}
	for _, item := range armor {
		entityID := strings.TrimSpace(item.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeArmor, entityID, catalog.DaggerheartAssetTypeArmorIllustration)
	}
	for _, item := range items {
		entityID := strings.TrimSpace(item.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeItem, entityID, catalog.DaggerheartAssetTypeItemIllustration)
	}

	sort.SliceStable(descriptors, func(i, j int) bool {
		left := descriptors[i]
		right := descriptors[j]
		if left.EntityType != right.EntityType {
			return left.EntityType < right.EntityType
		}
		if left.EntityID != right.EntityID {
			return left.EntityID < right.EntityID
		}
		return left.AssetType < right.AssetType
	})
	return descriptors
}

func appendAssetDescriptor(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, entityType, entityID, assetType string) {
	normalizedEntityType := strings.ToLower(strings.TrimSpace(entityType))
	normalizedEntityID := strings.TrimSpace(entityID)
	normalizedAssetType := strings.ToLower(strings.TrimSpace(assetType))
	if normalizedEntityType == "" || normalizedEntityID == "" || normalizedAssetType == "" {
		return
	}
	key := normalizedEntityType + "\x00" + normalizedEntityID + "\x00" + normalizedAssetType
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, daggerheartAssetDescriptor{
		EntityType: normalizedEntityType,
		EntityID:   normalizedEntityID,
		AssetType:  normalizedAssetType,
	})
}

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

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

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
