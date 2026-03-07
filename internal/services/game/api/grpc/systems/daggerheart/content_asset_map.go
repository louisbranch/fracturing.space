package daggerheart

import (
	"context"
	"fmt"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const (
	defaultDaggerheartAssetMapID            = "daggerheart-assets-v1"
	defaultDaggerheartAssetMapSystemID      = "daggerheart"
	defaultDaggerheartAssetMapSystemVersion = "v1"
	defaultDaggerheartAssetMapLocale        = commonv1.Locale_LOCALE_EN_US
)

type contentAssetDescriptor struct {
	EntityType string
	EntityID   string
	AssetType  string
}

func buildDaggerheartContentAssetMap(ctx context.Context, store storage.DaggerheartContentReadStore, requestedLocale commonv1.Locale) (*pb.DaggerheartContentAssetMap, error) {
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

	resolvedLocale := resolveDaggerheartAssetMapLocale(requestedLocale)

	descriptors := collectDaggerheartContentAssetDescriptors(
		classes,
		subclasses,
		heritages,
		domains,
		domainCards,
		adversaries,
		environments,
	)

	assetManifest := catalog.DaggerheartAssetsManifest()
	assets := make([]*pb.DaggerheartContentAssetRef, 0, len(descriptors))
	for _, descriptor := range descriptors {
		resolved := assetManifest.ResolveEntityAsset(descriptor.EntityType, descriptor.EntityID, descriptor.AssetType)
		assets = append(assets, &pb.DaggerheartContentAssetRef{
			Type:       contentAssetTypeToProto(descriptor.AssetType),
			Status:     contentAssetResolutionStatusToProto(resolved.Status),
			EntityType: descriptor.EntityType,
			EntityId:   descriptor.EntityID,
			SetId:      strings.TrimSpace(resolved.SetID),
			AssetId:    strings.TrimSpace(resolved.AssetID),
			CdnAssetId: strings.TrimSpace(resolved.CDNAssetID),
		})
	}

	return &pb.DaggerheartContentAssetMap{
		Id:            fallbackString(strings.TrimSpace(assetManifest.ID), defaultDaggerheartAssetMapID),
		SystemId:      fallbackString(strings.TrimSpace(assetManifest.SystemID), defaultDaggerheartAssetMapSystemID),
		SystemVersion: fallbackString(strings.TrimSpace(assetManifest.SystemVersion), defaultDaggerheartAssetMapSystemVersion),
		Locale:        resolvedLocale,
		Theme:         strings.TrimSpace(assetManifest.Theme),
		Assets:        assets,
	}, nil
}

func collectDaggerheartContentAssetDescriptors(
	classes []storage.DaggerheartClass,
	subclasses []storage.DaggerheartSubclass,
	heritages []storage.DaggerheartHeritage,
	domains []storage.DaggerheartDomain,
	domainCards []storage.DaggerheartDomainCard,
	adversaries []storage.DaggerheartAdversaryEntry,
	environments []storage.DaggerheartEnvironment,
) []contentAssetDescriptor {
	descriptors := make([]contentAssetDescriptor, 0, len(classes)*2+len(subclasses)+len(heritages)+len(domains)*2+len(domainCards)+len(adversaries)+len(environments))
	seen := map[string]struct{}{}

	for _, class := range classes {
		entityID := strings.TrimSpace(class.ID)
		if entityID == "" {
			continue
		}
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeClass, entityID, catalog.DaggerheartAssetTypeClassIllustration)
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeClass, entityID, catalog.DaggerheartAssetTypeClassIcon)
	}
	for _, subclass := range subclasses {
		entityID := strings.TrimSpace(subclass.ID)
		if entityID == "" {
			continue
		}
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeSubclass, entityID, catalog.DaggerheartAssetTypeSubclassIllustration)
	}
	for _, heritage := range heritages {
		entityID := strings.TrimSpace(heritage.ID)
		if entityID == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(heritage.Kind)) {
		case catalog.DaggerheartEntityTypeAncestry:
			appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeAncestry, entityID, catalog.DaggerheartAssetTypeAncestryIllustration)
		case catalog.DaggerheartEntityTypeCommunity:
			appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeCommunity, entityID, catalog.DaggerheartAssetTypeCommunityIllustration)
		}
	}
	for _, domain := range domains {
		entityID := strings.TrimSpace(domain.ID)
		if entityID == "" {
			continue
		}
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeDomain, entityID, catalog.DaggerheartAssetTypeDomainIllustration)
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeDomain, entityID, catalog.DaggerheartAssetTypeDomainIcon)
	}
	for _, domainCard := range domainCards {
		entityID := strings.TrimSpace(domainCard.ID)
		if entityID == "" {
			continue
		}
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeDomainCard, entityID, catalog.DaggerheartAssetTypeDomainCardIllustration)
	}
	for _, adversary := range adversaries {
		entityID := strings.TrimSpace(adversary.ID)
		if entityID == "" {
			continue
		}
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeAdversary, entityID, catalog.DaggerheartAssetTypeAdversaryIllustration)
	}
	for _, environment := range environments {
		entityID := strings.TrimSpace(environment.ID)
		if entityID == "" {
			continue
		}
		appendContentAssetDescriptor(&descriptors, seen, catalog.DaggerheartEntityTypeEnvironment, entityID, catalog.DaggerheartAssetTypeEnvironmentIllustration)
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

func appendContentAssetDescriptor(out *[]contentAssetDescriptor, seen map[string]struct{}, entityType, entityID, assetType string) {
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
	*out = append(*out, contentAssetDescriptor{
		EntityType: normalizedEntityType,
		EntityID:   normalizedEntityID,
		AssetType:  normalizedAssetType,
	})
}

func contentAssetTypeToProto(assetType string) pb.DaggerheartContentAssetType {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case catalog.DaggerheartAssetTypeClassIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_CLASS_ILLUSTRATION
	case catalog.DaggerheartAssetTypeClassIcon:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_CLASS_ICON
	case catalog.DaggerheartAssetTypeSubclassIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_SUBCLASS_ILLUSTRATION
	case catalog.DaggerheartAssetTypeAncestryIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ANCESTRY_ILLUSTRATION
	case catalog.DaggerheartAssetTypeCommunityIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_COMMUNITY_ILLUSTRATION
	case catalog.DaggerheartAssetTypeDomainIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_DOMAIN_ILLUSTRATION
	case catalog.DaggerheartAssetTypeDomainIcon:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_DOMAIN_ICON
	case catalog.DaggerheartAssetTypeDomainCardIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_DOMAIN_CARD_ILLUSTRATION
	case catalog.DaggerheartAssetTypeAdversaryIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ADVERSARY_ILLUSTRATION
	case catalog.DaggerheartAssetTypeEnvironmentIllustration:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ENVIRONMENT_ILLUSTRATION
	default:
		return pb.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_UNSPECIFIED
	}
}

func contentAssetResolutionStatusToProto(status catalog.DaggerheartAssetResolutionStatus) pb.DaggerheartContentAssetResolutionStatus {
	switch status {
	case catalog.DaggerheartAssetResolutionStatusMapped:
		return pb.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_MAPPED
	case catalog.DaggerheartAssetResolutionStatusSetDefault:
		return pb.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_SET_DEFAULT
	case catalog.DaggerheartAssetResolutionStatusUnavailable:
		return pb.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_UNAVAILABLE
	default:
		return pb.DaggerheartContentAssetResolutionStatus_DAGGERHEART_CONTENT_ASSET_RESOLUTION_STATUS_UNSPECIFIED
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
