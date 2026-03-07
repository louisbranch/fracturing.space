package gateway

import (
	"context"
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// CharacterCreationProgress centralizes this web behavior in one helper seam.
func (g GRPCGateway) CharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProgress, error) {
	if g.CharacterClient == nil {
		return campaignapp.CampaignCharacterCreationProgress{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.CharacterClient.GetCharacterCreationProgress(ctx, &statev1.GetCharacterCreationProgressRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacterCreationProgress{}, err
	}
	if resp == nil || resp.GetProgress() == nil {
		return campaignapp.CampaignCharacterCreationProgress{Steps: []campaignapp.CampaignCharacterCreationStep{}, UnmetReasons: []string{}}, nil
	}

	progress := resp.GetProgress()
	steps := make([]campaignapp.CampaignCharacterCreationStep, 0, len(progress.GetSteps()))
	for _, step := range progress.GetSteps() {
		if step == nil {
			continue
		}
		steps = append(steps, campaignapp.CampaignCharacterCreationStep{
			Step:     step.GetStep(),
			Key:      strings.TrimSpace(step.GetKey()),
			Complete: step.GetComplete(),
		})
	}
	unmetReasons := make([]string, 0, len(progress.GetUnmetReasons()))
	for _, reason := range progress.GetUnmetReasons() {
		trimmedReason := strings.TrimSpace(reason)
		if trimmedReason == "" {
			continue
		}
		unmetReasons = append(unmetReasons, trimmedReason)
	}
	return campaignapp.CampaignCharacterCreationProgress{
		Steps:        steps,
		NextStep:     progress.GetNextStep(),
		Ready:        progress.GetReady(),
		UnmetReasons: unmetReasons,
	}, nil
}

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
		classIllustration := mapCatalogAssetReference(
			g.AssetBaseURL,
			assetLookup.get(classID, "class", daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_CLASS_ILLUSTRATION),
		)
		classIcon := mapCatalogAssetReference(
			g.AssetBaseURL,
			assetLookup.get(classID, "class", daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_CLASS_ICON),
		)
		catalog.Classes = append(catalog.Classes, campaignapp.CatalogClass{
			ID:           classID,
			Name:         strings.TrimSpace(class.GetName()),
			DomainIDs:    domainIDs,
			Illustration: classIllustration,
			Icon:         classIcon,
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
		catalog.Subclasses = append(catalog.Subclasses, campaignapp.CatalogSubclass{
			ID:      subclassID,
			Name:    strings.TrimSpace(subclass.GetName()),
			ClassID: strings.TrimSpace(subclass.GetClassId()),
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
		kind := daggerheartHeritageKindLabel(heritage.GetKind())
		assetType := daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_UNSPECIFIED
		switch kind {
		case "ancestry":
			assetType = daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_ANCESTRY_ILLUSTRATION
		case "community":
			assetType = daggerheartv1.DaggerheartContentAssetType_DAGGERHEART_CONTENT_ASSET_TYPE_COMMUNITY_ILLUSTRATION
		}
		catalog.Heritages = append(catalog.Heritages, campaignapp.CatalogHeritage{
			ID:   heritageID,
			Name: strings.TrimSpace(heritage.GetName()),
			Kind: kind,
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
			ID:   armorID,
			Name: strings.TrimSpace(armor.GetName()),
			Tier: armor.GetTier(),
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
			ID:   itemID,
			Name: strings.TrimSpace(item.GetName()),
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
			ID:       domainCardID,
			Name:     strings.TrimSpace(domainCard.GetName()),
			DomainID: strings.TrimSpace(domainCard.GetDomainId()),
			Level:    domainCard.GetLevel(),
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

// CharacterCreationProfile centralizes this web behavior in one helper seam.
func (g GRPCGateway) CharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProfile, error) {
	if g.CharacterClient == nil {
		return campaignapp.CampaignCharacterCreationProfile{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.CharacterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacterCreationProfile{}, err
	}
	if resp == nil || resp.GetProfile() == nil || resp.GetProfile().GetDaggerheart() == nil {
		return campaignapp.CampaignCharacterCreationProfile{}, nil
	}
	profile := resp.GetProfile().GetDaggerheart()

	startingWeaponIDs := make([]string, 0, len(profile.GetStartingWeaponIds()))
	for _, weaponID := range profile.GetStartingWeaponIds() {
		trimmedWeaponID := strings.TrimSpace(weaponID)
		if trimmedWeaponID == "" {
			continue
		}
		startingWeaponIDs = append(startingWeaponIDs, trimmedWeaponID)
	}
	primaryWeaponID := ""
	secondaryWeaponID := ""
	if len(startingWeaponIDs) > 0 {
		primaryWeaponID = startingWeaponIDs[0]
	}
	if len(startingWeaponIDs) > 1 {
		secondaryWeaponID = startingWeaponIDs[1]
	}

	domainCardIDs := make([]string, 0, len(profile.GetDomainCardIds()))
	for _, domainCardID := range profile.GetDomainCardIds() {
		trimmedDomainCardID := strings.TrimSpace(domainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		domainCardIDs = append(domainCardIDs, trimmedDomainCardID)
	}

	experienceName := ""
	experienceModifier := ""
	if len(profile.GetExperiences()) > 0 && profile.GetExperiences()[0] != nil {
		experienceName = strings.TrimSpace(profile.GetExperiences()[0].GetName())
		experienceModifier = strconv.FormatInt(int64(profile.GetExperiences()[0].GetModifier()), 10)
	}

	return campaignapp.CampaignCharacterCreationProfile{
		ClassID:            strings.TrimSpace(profile.GetClassId()),
		SubclassID:         strings.TrimSpace(profile.GetSubclassId()),
		AncestryID:         strings.TrimSpace(profile.GetAncestryId()),
		CommunityID:        strings.TrimSpace(profile.GetCommunityId()),
		Agility:            int32ValueString(profile.GetAgility()),
		Strength:           int32ValueString(profile.GetStrength()),
		Finesse:            int32ValueString(profile.GetFinesse()),
		Instinct:           int32ValueString(profile.GetInstinct()),
		Presence:           int32ValueString(profile.GetPresence()),
		Knowledge:          int32ValueString(profile.GetKnowledge()),
		PrimaryWeaponID:    primaryWeaponID,
		SecondaryWeaponID:  secondaryWeaponID,
		ArmorID:            strings.TrimSpace(profile.GetStartingArmorId()),
		PotionItemID:       strings.TrimSpace(profile.GetStartingPotionItemId()),
		Background:         strings.TrimSpace(profile.GetBackground()),
		ExperienceName:     experienceName,
		ExperienceModifier: experienceModifier,
		DomainCardIDs:      domainCardIDs,
		Connections:        strings.TrimSpace(profile.GetConnections()),
	}, nil
}
