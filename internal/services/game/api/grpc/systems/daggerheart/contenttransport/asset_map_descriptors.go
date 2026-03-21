package contenttransport

import (
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

// collectDaggerheartAssetDescriptors normalizes the catalog families into the
// deterministic entity/asset lookup set expected by the published manifest.
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

	appendClassAssetDescriptors(&descriptors, seen, classes)
	appendSubclassAssetDescriptors(&descriptors, seen, subclasses)
	appendHeritageAssetDescriptors(&descriptors, seen, heritages)
	appendDomainAssetDescriptors(&descriptors, seen, domains)
	appendDomainCardAssetDescriptors(&descriptors, seen, domainCards)
	appendAdversaryAssetDescriptors(&descriptors, seen, adversaries)
	appendEnvironmentAssetDescriptors(&descriptors, seen, environments)
	appendWeaponAssetDescriptors(&descriptors, seen, weapons)
	appendArmorAssetDescriptors(&descriptors, seen, armor)
	appendItemAssetDescriptors(&descriptors, seen, items)

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

// appendAssetDescriptor records one normalized descriptor unless it is blank or
// already present in the asset lookup set.
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

// appendClassAssetDescriptors adds the published class illustration and icon
// lookups for each class entry.
func appendClassAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, classes []contentstore.DaggerheartClass) {
	for _, class := range classes {
		entityID := strings.TrimSpace(class.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeClass, entityID, catalog.DaggerheartAssetTypeClassIllustration)
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeClass, entityID, catalog.DaggerheartAssetTypeClassIcon)
	}
}

// appendSubclassAssetDescriptors adds the published subclass illustration
// lookup for each subclass entry.
func appendSubclassAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, subclasses []contentstore.DaggerheartSubclass) {
	for _, subclass := range subclasses {
		entityID := strings.TrimSpace(subclass.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeSubclass, entityID, catalog.DaggerheartAssetTypeSubclassIllustration)
	}
}

// appendHeritageAssetDescriptors maps heritage kinds to their published entity
// families before emitting illustration lookups.
func appendHeritageAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, heritages []contentstore.DaggerheartHeritage) {
	for _, heritage := range heritages {
		entityID := strings.TrimSpace(heritage.ID)
		if entityID == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(heritage.Kind)) {
		case catalog.DaggerheartEntityTypeAncestry:
			appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeAncestry, entityID, catalog.DaggerheartAssetTypeAncestryIllustration)
		case catalog.DaggerheartEntityTypeCommunity:
			appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeCommunity, entityID, catalog.DaggerheartAssetTypeCommunityIllustration)
		}
	}
}

// appendDomainAssetDescriptors adds the published illustration and icon
// lookups for each domain entry.
func appendDomainAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, domains []contentstore.DaggerheartDomain) {
	for _, domain := range domains {
		entityID := strings.TrimSpace(domain.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeDomain, entityID, catalog.DaggerheartAssetTypeDomainIllustration)
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeDomain, entityID, catalog.DaggerheartAssetTypeDomainIcon)
	}
}

// appendDomainCardAssetDescriptors adds the published illustration lookup for
// each domain card entry.
func appendDomainCardAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, domainCards []contentstore.DaggerheartDomainCard) {
	for _, domainCard := range domainCards {
		entityID := strings.TrimSpace(domainCard.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeDomainCard, entityID, catalog.DaggerheartAssetTypeDomainCardIllustration)
	}
}

// appendAdversaryAssetDescriptors adds the published illustration lookup for
// each adversary entry.
func appendAdversaryAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, adversaries []contentstore.DaggerheartAdversaryEntry) {
	for _, adversary := range adversaries {
		entityID := strings.TrimSpace(adversary.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeAdversary, entityID, catalog.DaggerheartAssetTypeAdversaryIllustration)
	}
}

// appendEnvironmentAssetDescriptors adds the published illustration lookup for
// each environment entry.
func appendEnvironmentAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, environments []contentstore.DaggerheartEnvironment) {
	for _, environment := range environments {
		entityID := strings.TrimSpace(environment.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeEnvironment, entityID, catalog.DaggerheartAssetTypeEnvironmentIllustration)
	}
}

// appendWeaponAssetDescriptors adds the published illustration lookup for each
// weapon entry.
func appendWeaponAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, weapons []contentstore.DaggerheartWeapon) {
	for _, weapon := range weapons {
		entityID := strings.TrimSpace(weapon.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeWeapon, entityID, catalog.DaggerheartAssetTypeWeaponIllustration)
	}
}

// appendArmorAssetDescriptors adds the published illustration lookup for each
// armor entry.
func appendArmorAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, armor []contentstore.DaggerheartArmor) {
	for _, item := range armor {
		entityID := strings.TrimSpace(item.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeArmor, entityID, catalog.DaggerheartAssetTypeArmorIllustration)
	}
}

// appendItemAssetDescriptors adds the published illustration lookup for each
// item entry.
func appendItemAssetDescriptors(out *[]daggerheartAssetDescriptor, seen map[string]struct{}, items []contentstore.DaggerheartItem) {
	for _, item := range items {
		entityID := strings.TrimSpace(item.ID)
		if entityID == "" {
			continue
		}
		appendAssetDescriptor(out, seen, catalog.DaggerheartEntityTypeItem, entityID, catalog.DaggerheartAssetTypeItemIllustration)
	}
}
