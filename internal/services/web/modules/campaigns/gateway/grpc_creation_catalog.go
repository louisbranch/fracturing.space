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

// CharacterCreationCatalog centralizes this web behavior in one helper seam.
func (g GRPCGateway) CharacterCreationCatalog(ctx context.Context, localeTag language.Tag) (campaignapp.CampaignCharacterCreationCatalog, error) {
	if g.DaggerheartClient == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.daggerheart_content_client_is_not_configured", "daggerheart content client is not configured")
	}
	locale := platformi18n.LocaleForTag(localeTag)
	locale = platformi18n.NormalizeLocale(locale)
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		locale = commonv1.Locale_LOCALE_EN_US
	}

	resp, err := g.DaggerheartClient.GetContentCatalog(ctx, &daggerheartv1.GetDaggerheartContentCatalogRequest{Locale: locale})
	if err != nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, err
	}
	if resp == nil || resp.GetCatalog() == nil {
		return campaignapp.CampaignCharacterCreationCatalog{}, nil
	}
	assetMapResp, err := g.DaggerheartClient.GetContentAssetMap(ctx, &daggerheartv1.GetDaggerheartContentAssetMapRequest{Locale: locale})
	if err != nil {
		assetMapResp = nil
	}
	assetLookup := daggerheartContentAssetLookupFromResponse(assetMapResp)

	catalogResp := resp.GetCatalog()
	catalog := campaignapp.CampaignCharacterCreationCatalog{
		AssetTheme: strings.TrimSpace(assetMapResp.GetAssetMap().GetTheme()),
	}

	catalog.Classes = make([]campaignapp.CatalogClass, 0, len(catalogResp.GetClasses()))
	for _, class := range catalogResp.GetClasses() {
		if class == nil {
			continue
		}
		classID := strings.TrimSpace(class.GetId())
		if classID == "" {
			continue
		}
		domainIDs := make([]string, 0, len(class.GetDomainIds()))
		for _, domainID := range class.GetDomainIds() {
			trimmedDomainID := strings.TrimSpace(domainID)
			if trimmedDomainID == "" {
				continue
			}
			domainIDs = append(domainIDs, trimmedDomainID)
		}
		features := make([]campaignapp.CatalogFeature, 0, len(class.GetFeatures()))
		for _, f := range class.GetFeatures() {
			name := strings.TrimSpace(f.GetName())
			if name != "" {
				features = append(features, campaignapp.CatalogFeature{
					Name:        name,
					Description: strings.TrimSpace(f.GetDescription()),
				})
			}
		}
		hopeFeature := campaignapp.CatalogFeature{}
		if hf := class.GetHopeFeature(); hf != nil {
			hopeFeature = campaignapp.CatalogFeature{
				Name:        strings.TrimSpace(hf.GetName()),
				Description: strings.TrimSpace(hf.GetDescription()),
			}
		}
		classIllustration := mapCatalogAssetReference(
			g.AssetBaseURL,
			assetLookup.get(classID, "class", daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_CLASS_ILLUSTRATION),
		)
		classIcon := mapCatalogAssetReference(
			g.AssetBaseURL,
			assetLookup.get(classID, "class", daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_CLASS_ICON),
		)
		catalog.Classes = append(catalog.Classes, campaignapp.CatalogClass{
			ID:              classID,
			Name:            strings.TrimSpace(class.GetName()),
			DomainIDs:       domainIDs,
			StartingHP:      class.GetStartingHp(),
			StartingEvasion: class.GetStartingEvasion(),
			HopeFeature:     hopeFeature,
			Features:        features,
			Illustration:    classIllustration,
			Icon:            classIcon,
		})
	}

	catalog.Subclasses = make([]campaignapp.CatalogSubclass, 0, len(catalogResp.GetSubclasses()))
	for _, subclass := range catalogResp.GetSubclasses() {
		if subclass == nil {
			continue
		}
		subclassID := strings.TrimSpace(subclass.GetId())
		if subclassID == "" {
			continue
		}
		foundation := make([]campaignapp.CatalogFeature, 0, len(subclass.GetFoundationFeatures()))
		for _, f := range subclass.GetFoundationFeatures() {
			name := strings.TrimSpace(f.GetName())
			if name != "" {
				foundation = append(foundation, campaignapp.CatalogFeature{
					Name:        name,
					Description: strings.TrimSpace(f.GetDescription()),
				})
			}
		}
		catalog.Subclasses = append(catalog.Subclasses, campaignapp.CatalogSubclass{
			ID:             subclassID,
			Name:           strings.TrimSpace(subclass.GetName()),
			ClassID:        strings.TrimSpace(subclass.GetClassId()),
			SpellcastTrait: strings.TrimSpace(subclass.GetSpellcastTrait()),
			Foundation:     foundation,
			Illustration: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(
					subclassID,
					"subclass",
					daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_SUBCLASS_ILLUSTRATION,
				),
			),
		})
	}

	catalog.Heritages = make([]campaignapp.CatalogHeritage, 0, len(catalogResp.GetHeritages()))
	for _, heritage := range catalogResp.GetHeritages() {
		if heritage == nil {
			continue
		}
		heritageID := strings.TrimSpace(heritage.GetId())
		if heritageID == "" {
			continue
		}
		heritageFeatures := make([]campaignapp.CatalogFeature, 0, len(heritage.GetFeatures()))
		for _, f := range heritage.GetFeatures() {
			name := strings.TrimSpace(f.GetName())
			if name != "" {
				heritageFeatures = append(heritageFeatures, campaignapp.CatalogFeature{
					Name:        name,
					Description: strings.TrimSpace(f.GetDescription()),
				})
			}
		}
		kind := daggerheartHeritageKindLabel(heritage.GetKind())
		assetType := daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_UNSPECIFIED
		switch kind {
		case "ancestry":
			assetType = daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ANCESTRY_ILLUSTRATION
		case "community":
			assetType = daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_COMMUNITY_ILLUSTRATION
		}
		catalog.Heritages = append(catalog.Heritages, campaignapp.CatalogHeritage{
			ID:       heritageID,
			Name:     strings.TrimSpace(heritage.GetName()),
			Kind:     kind,
			Features: heritageFeatures,
			Illustration: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(heritageID, kind, assetType),
			),
		})
	}

	catalog.Domains = make([]campaignapp.CatalogDomain, 0, len(catalogResp.GetDomains()))
	for _, domain := range catalogResp.GetDomains() {
		if domain == nil {
			continue
		}
		domainID := strings.TrimSpace(domain.GetId())
		if domainID == "" {
			continue
		}
		catalog.Domains = append(catalog.Domains, campaignapp.CatalogDomain{
			ID:   domainID,
			Name: strings.TrimSpace(domain.GetName()),
			Illustration: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(
					domainID,
					"domain",
					daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_DOMAIN_ILLUSTRATION,
				),
			),
			Icon: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(
					domainID,
					"domain",
					daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_DOMAIN_ICON,
				),
			),
		})
	}

	catalog.Weapons = make([]campaignapp.CatalogWeapon, 0, len(catalogResp.GetWeapons()))
	for _, weapon := range catalogResp.GetWeapons() {
		if weapon == nil {
			continue
		}
		weaponID := strings.TrimSpace(weapon.GetId())
		if weaponID == "" {
			continue
		}
		catalog.Weapons = append(catalog.Weapons, campaignapp.CatalogWeapon{
			ID:       weaponID,
			Name:     strings.TrimSpace(weapon.GetName()),
			Category: daggerheartWeaponCategoryLabel(weapon.GetCategory()),
			Tier:     weapon.GetTier(),
			Trait:    strings.TrimSpace(weapon.GetTrait()),
			Range:    strings.TrimSpace(weapon.GetRange()),
			Damage:   formatDamageDice(weapon.GetDamageDice()),
			Feature:  strings.TrimSpace(weapon.GetFeature()),
		})
	}

	catalog.Armor = make([]campaignapp.CatalogArmor, 0, len(catalogResp.GetArmor()))
	for _, armor := range catalogResp.GetArmor() {
		if armor == nil {
			continue
		}
		armorID := strings.TrimSpace(armor.GetId())
		if armorID == "" {
			continue
		}
		catalog.Armor = append(catalog.Armor, campaignapp.CatalogArmor{
			ID:             armorID,
			Name:           strings.TrimSpace(armor.GetName()),
			Tier:           armor.GetTier(),
			ArmorScore:     armor.GetArmorScore(),
			BaseThresholds: fmt.Sprintf("Major %d / Severe %d", armor.GetBaseMajorThreshold(), armor.GetBaseSevereThreshold()),
			Feature:        strings.TrimSpace(armor.GetFeature()),
		})
	}

	catalog.Items = make([]campaignapp.CatalogItem, 0, len(catalogResp.GetItems()))
	for _, item := range catalogResp.GetItems() {
		if item == nil {
			continue
		}
		itemID := strings.TrimSpace(item.GetId())
		if itemID == "" {
			continue
		}
		catalog.Items = append(catalog.Items, campaignapp.CatalogItem{
			ID:          itemID,
			Name:        strings.TrimSpace(item.GetName()),
			Description: strings.TrimSpace(item.GetDescription()),
		})
	}

	catalog.DomainCards = make([]campaignapp.CatalogDomainCard, 0, len(catalogResp.GetDomainCards()))
	for _, domainCard := range catalogResp.GetDomainCards() {
		if domainCard == nil {
			continue
		}
		domainCardID := strings.TrimSpace(domainCard.GetId())
		if domainCardID == "" {
			continue
		}
		catalog.DomainCards = append(catalog.DomainCards, campaignapp.CatalogDomainCard{
			ID:          domainCardID,
			Name:        strings.TrimSpace(domainCard.GetName()),
			DomainID:    strings.TrimSpace(domainCard.GetDomainId()),
			Level:       domainCard.GetLevel(),
			Type:        daggerheartDomainCardTypeLabel(domainCard.GetType()),
			RecallCost:  domainCard.GetRecallCost(),
			FeatureText: strings.TrimSpace(domainCard.GetFeatureText()),
			Illustration: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(
					domainCardID,
					"domain_card",
					daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_DOMAIN_CARD_ILLUSTRATION,
				),
			),
		})
	}

	catalog.Adversaries = make([]campaignapp.CatalogAdversary, 0, len(catalogResp.GetAdversaries()))
	for _, adversary := range catalogResp.GetAdversaries() {
		if adversary == nil {
			continue
		}
		adversaryID := strings.TrimSpace(adversary.GetId())
		if adversaryID == "" {
			continue
		}
		catalog.Adversaries = append(catalog.Adversaries, campaignapp.CatalogAdversary{
			ID:   adversaryID,
			Name: strings.TrimSpace(adversary.GetName()),
			Illustration: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(
					adversaryID,
					"adversary",
					daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ADVERSARY_ILLUSTRATION,
				),
			),
		})
	}

	catalog.Environments = make([]campaignapp.CatalogEnvironment, 0, len(catalogResp.GetEnvironments()))
	for _, environment := range catalogResp.GetEnvironments() {
		if environment == nil {
			continue
		}
		environmentID := strings.TrimSpace(environment.GetId())
		if environmentID == "" {
			continue
		}
		catalog.Environments = append(catalog.Environments, campaignapp.CatalogEnvironment{
			ID:   environmentID,
			Name: strings.TrimSpace(environment.GetName()),
			Illustration: mapCatalogAssetReference(
				g.AssetBaseURL,
				assetLookup.get(
					environmentID,
					"environment",
					daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ENVIRONMENT_ILLUSTRATION,
				),
			),
		})
	}

	return catalog, nil
}
