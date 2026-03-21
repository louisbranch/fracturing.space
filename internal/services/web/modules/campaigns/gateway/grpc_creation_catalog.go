package gateway

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

const creationIllustrationDeliveryWidthPX = 384
const creationIconDeliveryWidthPX = 64

// CharacterCreationCatalog centralizes this web behavior in one helper seam.
func (g characterCreationReadGateway) CharacterCreationCatalog(ctx context.Context, localeTag language.Tag) (campaignapp.CampaignCharacterCreationCatalog, error) {
	if g.read.DaggerheartContent == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart content client is not configured")
	}
	if g.read.DaggerheartAsset == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart asset client is not configured")
	}
	locale := platformi18n.LocaleForTag(localeTag)
	locale = platformi18n.NormalizeLocale(locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}

	resp, err := g.read.DaggerheartContent.GetContentCatalog(ctx, &daggerheartv1.GetDaggerheartContentCatalogRequest{Locale: locale})
	if err != nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, err
	}
	if resp == nil || resp.GetCatalog() == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, nil
	}

	assetMapResp, err := g.read.DaggerheartAsset.GetAssetMap(ctx, &daggerheartv1.GetDaggerheartAssetMapRequest{Locale: locale})
	if err != nil {
		assetMapResp = nil
	}
	assetLookup := daggerheartAssetLookupFromResponse(assetMapResp)

	return campaignCharacterCreationCatalogFromProto(resp.GetCatalog(), g.assetBaseURL, assetLookup, assetMapResp), nil
}

// campaignCharacterCreationCatalogFromProto maps the proto content catalog to web-facing domain catalog models.
func campaignCharacterCreationCatalogFromProto(
	catalogResp *daggerheartv1.DaggerheartContentCatalog,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
	assetMapResp *daggerheartv1.GetDaggerheartAssetMapResponse,
) campaignapp.CampaignCharacterCreationCatalog {
	if catalogResp == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}
	}
	return campaignapp.CampaignCharacterCreationCatalog{
		AssetTheme:           catalogAssetTheme(assetMapResp),
		Classes:              mapCatalogClasses(catalogResp.GetClasses(), assetBaseURL, assetLookup),
		Subclasses:           mapCatalogSubclasses(catalogResp.GetSubclasses(), assetBaseURL, assetLookup),
		Heritages:            mapCatalogHeritages(catalogResp.GetHeritages(), assetBaseURL, assetLookup),
		CompanionExperiences: mapCatalogCompanionExperiences(catalogResp.GetCompanionExperiences()),
		Domains:              mapCatalogDomains(catalogResp.GetDomains(), assetBaseURL, assetLookup),
		Weapons:              mapCatalogWeapons(catalogResp.GetWeapons(), assetBaseURL, assetLookup),
		Armor:                mapCatalogArmor(catalogResp.GetArmor(), assetBaseURL, assetLookup),
		Items:                mapCatalogItems(catalogResp.GetItems(), assetBaseURL, assetLookup),
		DomainCards:          mapCatalogDomainCards(catalogResp.GetDomainCards(), assetBaseURL, assetLookup),
		Adversaries:          mapCatalogAdversaries(catalogResp.GetAdversaries(), assetBaseURL, assetLookup),
		Environments:         mapCatalogEnvironments(catalogResp.GetEnvironments(), assetBaseURL, assetLookup),
	}
}

// mapCatalogCompanionExperiences normalizes catalog companion experiences into
// the app-owned creation catalog shape.
func mapCatalogCompanionExperiences(experiences []*daggerheartv1.DaggerheartCompanionExperienceEntry) []campaignapp.CatalogCompanionExperience {
	mapped := make([]campaignapp.CatalogCompanionExperience, 0, len(experiences))
	for _, experience := range experiences {
		if experience == nil {
			continue
		}
		id := strings.TrimSpace(experience.GetId())
		if id == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogCompanionExperience{
			ID:          id,
			Name:        strings.TrimSpace(experience.GetName()),
			Description: strings.TrimSpace(experience.GetDescription()),
		})
	}
	return mapped
}

// catalogAssetTheme resolves an optional asset-map theme value used by presentation workflows.
func catalogAssetTheme(resp *daggerheartv1.GetDaggerheartAssetMapResponse) string {
	if resp == nil || resp.GetAssetMap() == nil {
		return ""
	}
	return strings.TrimSpace(resp.GetAssetMap().GetTheme())
}

// mapCatalogClasses projects class catalog records into web domain types.
func mapCatalogClasses(
	classes []*daggerheartv1.DaggerheartClass,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogClass {
	mapped := make([]campaignapp.CatalogClass, 0, len(classes))
	for _, class := range classes {
		if class == nil {
			continue
		}
		classID := strings.TrimSpace(class.GetId())
		if classID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogClass{
			ID:              classID,
			Name:            strings.TrimSpace(class.GetName()),
			DomainIDs:       trimNonEmptyValues(class.GetDomainIds()),
			StartingHP:      class.GetStartingHp(),
			StartingEvasion: class.GetStartingEvasion(),
			HopeFeature:     mapCatalogHopeFeature(class.GetHopeFeature()),
			Features:        mapCatalogFeatures(class.GetFeatures()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(classID, "class", daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION),
				creationIllustrationDeliveryWidthPX,
			),
			Icon: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(classID, "class", daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ICON),
				creationIconDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogHopeFeature projects one proto class hope feature into a web domain feature.
func mapCatalogHopeFeature(feature *daggerheartv1.DaggerheartHopeFeature) campaignapp.CatalogFeature {
	if feature == nil {
		return campaignapp.CatalogFeature{}
	}
	return campaignapp.CatalogFeature{
		Name:        strings.TrimSpace(feature.GetName()),
		Description: strings.TrimSpace(feature.GetDescription()),
	}
}

// mapCatalogSubclasses projects subclass catalog records into web domain types.
func mapCatalogSubclasses(
	subclasses []*daggerheartv1.DaggerheartSubclass,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogSubclass {
	mapped := make([]campaignapp.CatalogSubclass, 0, len(subclasses))
	for _, subclass := range subclasses {
		if subclass == nil {
			continue
		}
		subclassID := strings.TrimSpace(subclass.GetId())
		if subclassID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogSubclass{
			ID:                   subclassID,
			Name:                 strings.TrimSpace(subclass.GetName()),
			ClassID:              strings.TrimSpace(subclass.GetClassId()),
			SpellcastTrait:       strings.TrimSpace(subclass.GetSpellcastTrait()),
			CreationRequirements: mapSubclassCreationRequirements(subclass.GetCreationRequirements()),
			Foundation:           mapCatalogFeatures(subclass.GetFoundationFeatures()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					subclassID,
					"subclass",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_SUBCLASS_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapSubclassCreationRequirements keeps the web layer on stable string labels.
func mapSubclassCreationRequirements(requirements []daggerheartv1.DaggerheartCreationRequirement) []string {
	mapped := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		switch requirement {
		case daggerheartv1.DaggerheartCreationRequirement_DAGGERHEART_CREATION_REQUIREMENT_COMPANION_SHEET:
			mapped = append(mapped, "companion_sheet_required")
		}
	}
	return mapped
}

// mapCatalogHeritages projects heritage catalog records into web domain types.
func mapCatalogHeritages(
	heritages []*daggerheartv1.DaggerheartHeritage,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogHeritage {
	mapped := make([]campaignapp.CatalogHeritage, 0, len(heritages))
	for _, heritage := range heritages {
		if heritage == nil {
			continue
		}
		heritageID := strings.TrimSpace(heritage.GetId())
		if heritageID == "" {
			continue
		}
		kind := daggerheartHeritageKindLabel(heritage.GetKind())
		mapped = append(mapped, campaignapp.CatalogHeritage{
			ID:       heritageID,
			Name:     strings.TrimSpace(heritage.GetName()),
			Kind:     kind,
			Features: mapCatalogFeatures(heritage.GetFeatures()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(heritageID, kind, heritageAssetType(kind)),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// heritageAssetType resolves the image asset type by normalized heritage kind.
func heritageAssetType(kind string) daggerheartv1.DaggerheartAssetType {
	switch kind {
	case "ancestry":
		return daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ANCESTRY_ILLUSTRATION
	case "community":
		return daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_COMMUNITY_ILLUSTRATION
	default:
		return daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_UNSPECIFIED
	}
}

// mapCatalogDomains projects domain catalog records into web domain types.
func mapCatalogDomains(
	domains []*daggerheartv1.DaggerheartDomain,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogDomain {
	mapped := make([]campaignapp.CatalogDomain, 0, len(domains))
	for _, domain := range domains {
		if domain == nil {
			continue
		}
		domainID := strings.TrimSpace(domain.GetId())
		if domainID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogDomain{
			ID:   domainID,
			Name: strings.TrimSpace(domain.GetName()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					domainID,
					"domain",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
			Icon: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					domainID,
					"domain",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ICON,
				),
				creationIconDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogWeapons projects weapon catalog records into web domain types.
func mapCatalogWeapons(
	weapons []*daggerheartv1.DaggerheartWeapon,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogWeapon {
	mapped := make([]campaignapp.CatalogWeapon, 0, len(weapons))
	for _, weapon := range weapons {
		if weapon == nil {
			continue
		}
		weaponID := strings.TrimSpace(weapon.GetId())
		if weaponID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogWeapon{
			ID:           weaponID,
			Name:         strings.TrimSpace(weapon.GetName()),
			Category:     daggerheartWeaponCategoryLabel(weapon.GetCategory()),
			Tier:         weapon.GetTier(),
			Burden:       weapon.GetBurden(),
			Trait:        strings.TrimSpace(weapon.GetTrait()),
			Range:        strings.TrimSpace(weapon.GetRange()),
			Damage:       formatDamageDice(weapon.GetDamageDice()),
			Feature:      strings.TrimSpace(weapon.GetFeature()),
			DisplayOrder: weapon.GetDisplayOrder(),
			DisplayGroup: daggerheartWeaponDisplayGroupLabel(weapon.GetDisplayGroup()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					weaponID,
					"weapon",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_WEAPON_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogArmor projects armor catalog records into web domain types.
func mapCatalogArmor(
	armorSet []*daggerheartv1.DaggerheartArmor,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogArmor {
	mapped := make([]campaignapp.CatalogArmor, 0, len(armorSet))
	for _, armor := range armorSet {
		if armor == nil {
			continue
		}
		armorID := strings.TrimSpace(armor.GetId())
		if armorID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogArmor{
			ID:             armorID,
			Name:           strings.TrimSpace(armor.GetName()),
			Tier:           armor.GetTier(),
			ArmorScore:     armor.GetArmorScore(),
			BaseThresholds: fmt.Sprintf("Major %d / Severe %d", armor.GetBaseMajorThreshold(), armor.GetBaseSevereThreshold()),
			Feature:        strings.TrimSpace(armor.GetFeature()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					armorID,
					"armor",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ARMOR_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogItems projects item catalog records into web domain types.
func mapCatalogItems(
	items []*daggerheartv1.DaggerheartItem,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogItem {
	mapped := make([]campaignapp.CatalogItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		itemID := strings.TrimSpace(item.GetId())
		if itemID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogItem{
			ID:          itemID,
			Name:        strings.TrimSpace(item.GetName()),
			Description: strings.TrimSpace(item.GetDescription()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					itemID,
					"item",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ITEM_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogDomainCards projects domain-card catalog records into web domain types.
func mapCatalogDomainCards(
	domainCards []*daggerheartv1.DaggerheartDomainCard,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogDomainCard {
	mapped := make([]campaignapp.CatalogDomainCard, 0, len(domainCards))
	for _, domainCard := range domainCards {
		if domainCard == nil {
			continue
		}
		domainCardID := strings.TrimSpace(domainCard.GetId())
		if domainCardID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogDomainCard{
			ID:          domainCardID,
			Name:        strings.TrimSpace(domainCard.GetName()),
			DomainID:    strings.TrimSpace(domainCard.GetDomainId()),
			Level:       domainCard.GetLevel(),
			Type:        daggerheartDomainCardTypeLabel(domainCard.GetType()),
			RecallCost:  domainCard.GetRecallCost(),
			FeatureText: strings.TrimSpace(domainCard.GetFeatureText()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					domainCardID,
					"domain_card",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_CARD_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogAdversaries projects adversary catalog records into web domain types.
func mapCatalogAdversaries(
	adversaries []*daggerheartv1.DaggerheartAdversaryEntry,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogAdversary {
	mapped := make([]campaignapp.CatalogAdversary, 0, len(adversaries))
	for _, adversary := range adversaries {
		if adversary == nil {
			continue
		}
		adversaryID := strings.TrimSpace(adversary.GetId())
		if adversaryID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogAdversary{
			ID:   adversaryID,
			Name: strings.TrimSpace(adversary.GetName()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					adversaryID,
					"adversary",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ADVERSARY_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogEnvironments projects environment catalog records into web domain types.
func mapCatalogEnvironments(
	environments []*daggerheartv1.DaggerheartEnvironment,
	assetBaseURL string,
	assetLookup daggerheartAssetLookup,
) []campaignapp.CatalogEnvironment {
	mapped := make([]campaignapp.CatalogEnvironment, 0, len(environments))
	for _, environment := range environments {
		if environment == nil {
			continue
		}
		environmentID := strings.TrimSpace(environment.GetId())
		if environmentID == "" {
			continue
		}
		mapped = append(mapped, campaignapp.CatalogEnvironment{
			ID:   environmentID,
			Name: strings.TrimSpace(environment.GetName()),
			Illustration: mapCatalogAssetReference(
				assetBaseURL,
				assetLookup.get(
					environmentID,
					"environment",
					daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ENVIRONMENT_ILLUSTRATION,
				),
				creationIllustrationDeliveryWidthPX,
			),
		})
	}
	return mapped
}

// mapCatalogFeatures filters and normalizes feature entries into web domain types.
func mapCatalogFeatures(features []*daggerheartv1.DaggerheartFeature) []campaignapp.CatalogFeature {
	mapped := make([]campaignapp.CatalogFeature, 0, len(features))
	for _, feature := range features {
		mappedFeature := mapCatalogFeature(feature)
		if mappedFeature.Name == "" {
			continue
		}
		mapped = append(mapped, mappedFeature)
	}
	return mapped
}

// mapCatalogFeature projects one proto feature into a web domain feature.
func mapCatalogFeature(feature *daggerheartv1.DaggerheartFeature) campaignapp.CatalogFeature {
	if feature == nil {
		return campaignapp.CatalogFeature{}
	}
	return campaignapp.CatalogFeature{
		Name:        strings.TrimSpace(feature.GetName()),
		Description: strings.TrimSpace(feature.GetDescription()),
	}
}

// trimNonEmptyValues keeps stable order while removing empty values.
func trimNonEmptyValues(values []string) []string {
	mapped := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		mapped = append(mapped, trimmed)
	}
	return mapped
}
