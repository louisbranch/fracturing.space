package daggerheart

import (
	"log/slog"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CreationView maps the domain creation model to the template view type.
func (w Workflow) CreationView(creation campaignapp.CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView {
	view := newCreationView(creation)
	view.Steps = mapCreationSteps(creation.Progress.Steps)

	cdn := creationImageCDN(w.AssetBaseURL)
	domainByID := domainLookupByID(creation.Domains)

	view.Classes = mapCreationClasses(creation.Classes, domainByID, cdn)
	view.Subclasses = mapCreationSubclasses(creation.Subclasses, cdn)
	view.Ancestries = mapCreationHeritages(creation.Ancestries, catalog.DaggerheartEntityTypeAncestry, catalog.DaggerheartAssetTypeAncestryIllustration, cdn)
	view.Communities = mapCreationHeritages(creation.Communities, catalog.DaggerheartEntityTypeCommunity, catalog.DaggerheartAssetTypeCommunityIllustration, cdn)
	view.PrimaryWeapons = mapCreationWeapons(creation.PrimaryWeapons)
	view.SecondaryWeapons = mapCreationWeapons(creation.SecondaryWeapons)
	view.Armor = mapCreationArmor(creation.Armor)
	view.PotionItems = mapCreationItems(creation.PotionItems)
	view.DomainCards = mapCreationDomainCards(creation.DomainCards, cdn)

	return view
}

// domainView supports class-domain label and watermark assembly.
type domainView struct {
	Name    string
	IconURL string
}

// domainLookupByID builds domain metadata lookup used across class/card mapping.
func domainLookupByID(domains []campaignapp.CatalogDomain) map[string]domainView {
	domainByID := make(map[string]domainView, len(domains))
	for _, domain := range domains {
		domainID := strings.TrimSpace(domain.ID)
		if domainID == "" {
			continue
		}
		domainName := strings.TrimSpace(domain.Name)
		if domainName == "" {
			domainName = domainID
		}
		domainByID[domainID] = domainView{
			Name:    domainName,
			IconURL: strings.TrimSpace(domain.Icon.URL),
		}
	}
	return domainByID
}

// newCreationView initializes template view state from normalized creation data.
func newCreationView(creation campaignapp.CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView {
	return webtemplates.CampaignCharacterCreationView{
		Ready:             creation.Progress.Ready,
		NextStep:          creation.Progress.NextStep,
		UnmetReasons:      append([]string(nil), creation.Progress.UnmetReasons...),
		ClassID:           creation.Profile.ClassID,
		SubclassID:        creation.Profile.SubclassID,
		AncestryID:        creation.Profile.AncestryID,
		CommunityID:       creation.Profile.CommunityID,
		Agility:           creation.Profile.Agility,
		Strength:          creation.Profile.Strength,
		Finesse:           creation.Profile.Finesse,
		Instinct:          creation.Profile.Instinct,
		Presence:          creation.Profile.Presence,
		Knowledge:         creation.Profile.Knowledge,
		PrimaryWeaponID:   creation.Profile.PrimaryWeaponID,
		SecondaryWeaponID: creation.Profile.SecondaryWeaponID,
		ArmorID:           creation.Profile.ArmorID,
		PotionItemID:      creation.Profile.PotionItemID,
		Background:        creation.Profile.Background,
		Experiences:       mapCreationExperiences(creation.Profile.Experiences),
		DomainCardIDs:     append([]string(nil), creation.Profile.DomainCardIDs...),
		Connections:       creation.Profile.Connections,
		Steps:             nil,
		Classes:           nil,
		Subclasses:        nil,
		Ancestries:        nil,
		Communities:       nil,
		PrimaryWeapons:    nil,
		SecondaryWeapons:  nil,
		Armor:             nil,
		PotionItems:       nil,
		DomainCards:       nil,
	}
}

// mapCreationExperiences maps profile experiences to template view rows.
func mapCreationExperiences(experiences []campaignapp.CampaignCharacterCreationExperience) []webtemplates.CampaignCreationExperienceView {
	mapped := make([]webtemplates.CampaignCreationExperienceView, 0, len(experiences))
	for _, exp := range experiences {
		mapped = append(mapped, webtemplates.CampaignCreationExperienceView{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}
	return mapped
}

// mapCreationSteps maps progress step state to template view rows.
func mapCreationSteps(steps []campaignapp.CampaignCharacterCreationStep) []webtemplates.CampaignCharacterCreationStepView {
	mapped := make([]webtemplates.CampaignCharacterCreationStepView, 0, len(steps))
	for _, step := range steps {
		mapped = append(mapped, webtemplates.CampaignCharacterCreationStepView{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	return mapped
}

// creationImageCDN returns a CDN client only when asset base URL is configured.
func creationImageCDN(assetBaseURL string) imagecdn.ImageCDN {
	if assetBaseURL == "" {
		return nil
	}
	return imagecdn.New(assetBaseURL)
}

// mapCreationClasses maps class catalog entries including derived domain metadata.
func mapCreationClasses(classes []campaignapp.CatalogClass, domainByID map[string]domainView, cdn imagecdn.ImageCDN) []webtemplates.CampaignCreationClassView {
	mapped := make([]webtemplates.CampaignCreationClassView, 0, len(classes))
	for _, class := range classes {
		domainNames, domainWatermarks := mapClassDomains(class.DomainIDs, domainByID)
		mapped = append(mapped, webtemplates.CampaignCreationClassView{
			ID:               class.ID,
			Name:             class.Name,
			ImageURL:         resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeClass, class.ID, catalog.DaggerheartAssetTypeClassIllustration),
			StartingHP:       class.StartingHP,
			StartingEvasion:  class.StartingEvasion,
			HopeFeature:      mapFeature(class.HopeFeature),
			Features:         mapFeatures(class.Features),
			DomainNames:      domainNames,
			DomainWatermarks: domainWatermarks,
		})
	}
	return mapped
}

// mapClassDomains derives class domain labels and up to two watermark icons.
func mapClassDomains(domainIDs []string, domainByID map[string]domainView) ([]string, []webtemplates.CampaignCreationDomainWatermarkView) {
	names := make([]string, 0, len(domainIDs))
	watermarks := make([]webtemplates.CampaignCreationDomainWatermarkView, 0, 2)
	for _, domainID := range domainIDs {
		trimmedDomainID := strings.TrimSpace(domainID)
		if trimmedDomainID == "" {
			continue
		}
		domain, ok := domainByID[trimmedDomainID]
		if !ok {
			continue
		}
		names = append(names, domain.Name)
		if domain.IconURL != "" && len(watermarks) < 2 {
			watermarks = append(watermarks, webtemplates.CampaignCreationDomainWatermarkView{
				ID:      trimmedDomainID,
				Name:    domain.Name,
				IconURL: domain.IconURL,
			})
		}
	}
	return names, watermarks
}

// mapCreationSubclasses maps subclass catalog entries to template view rows.
func mapCreationSubclasses(subclasses []campaignapp.CatalogSubclass, cdn imagecdn.ImageCDN) []webtemplates.CampaignCreationSubclassView {
	mapped := make([]webtemplates.CampaignCreationSubclassView, 0, len(subclasses))
	for _, subclass := range subclasses {
		mapped = append(mapped, webtemplates.CampaignCreationSubclassView{
			ID:             subclass.ID,
			Name:           subclass.Name,
			ImageURL:       resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeSubclass, subclass.ID, catalog.DaggerheartAssetTypeSubclassIllustration),
			ClassID:        subclass.ClassID,
			SpellcastTrait: subclass.SpellcastTrait,
			Foundation:     mapFeatures(subclass.Foundation),
		})
	}
	return mapped
}

// mapCreationHeritages maps ancestry/community catalogs with resolved image URLs.
func mapCreationHeritages(
	heritages []campaignapp.CatalogHeritage,
	entityType string,
	assetType string,
	cdn imagecdn.ImageCDN,
) []webtemplates.CampaignCreationHeritageView {
	mapped := make([]webtemplates.CampaignCreationHeritageView, 0, len(heritages))
	for _, heritage := range heritages {
		mapped = append(mapped, webtemplates.CampaignCreationHeritageView{
			ID:       heritage.ID,
			Name:     heritage.Name,
			ImageURL: resolveEntityImageURL(cdn, entityType, heritage.ID, assetType),
			Features: mapFeatures(heritage.Features),
		})
	}
	return mapped
}

// mapFeatures maps catalog features to template feature rows.
func mapFeatures(features []campaignapp.CatalogFeature) []webtemplates.CampaignCreationClassFeatureView {
	mapped := make([]webtemplates.CampaignCreationClassFeatureView, 0, len(features))
	for _, feature := range features {
		mapped = append(mapped, mapFeature(feature))
	}
	return mapped
}

// mapFeature maps one catalog feature to a template feature row.
func mapFeature(feature campaignapp.CatalogFeature) webtemplates.CampaignCreationClassFeatureView {
	return webtemplates.CampaignCreationClassFeatureView{
		Name:        feature.Name,
		Description: feature.Description,
	}
}

// mapCreationWeapons maps weapon catalog entries to template rows.
func mapCreationWeapons(weapons []campaignapp.CatalogWeapon) []webtemplates.CampaignCreationWeaponView {
	mapped := make([]webtemplates.CampaignCreationWeaponView, 0, len(weapons))
	for _, weapon := range weapons {
		mapped = append(mapped, webtemplates.CampaignCreationWeaponView{
			ID:      weapon.ID,
			Name:    weapon.Name,
			Trait:   weapon.Trait,
			Range:   weapon.Range,
			Damage:  weapon.Damage,
			Feature: weapon.Feature,
		})
	}
	return mapped
}

// mapCreationArmor maps armor catalog entries to template rows.
func mapCreationArmor(items []campaignapp.CatalogArmor) []webtemplates.CampaignCreationArmorView {
	mapped := make([]webtemplates.CampaignCreationArmorView, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, webtemplates.CampaignCreationArmorView{
			ID:             item.ID,
			Name:           item.Name,
			ArmorScore:     item.ArmorScore,
			BaseThresholds: item.BaseThresholds,
			Feature:        item.Feature,
		})
	}
	return mapped
}

// mapCreationItems maps item catalog entries to template rows.
func mapCreationItems(items []campaignapp.CatalogItem) []webtemplates.CampaignCreationItemView {
	mapped := make([]webtemplates.CampaignCreationItemView, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, webtemplates.CampaignCreationItemView{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	return mapped
}

// mapCreationDomainCards maps domain-card entries to template rows.
func mapCreationDomainCards(cards []campaignapp.CatalogDomainCard, cdn imagecdn.ImageCDN) []webtemplates.CampaignCreationDomainCardView {
	mapped := make([]webtemplates.CampaignCreationDomainCardView, 0, len(cards))
	for _, card := range cards {
		imageURL := strings.TrimSpace(card.Illustration.URL)
		if imageURL == "" {
			imageURL = resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeDomainCard, card.ID, catalog.DaggerheartAssetTypeDomainCardIllustration)
		}
		mapped = append(mapped, webtemplates.CampaignCreationDomainCardView{
			ID:          card.ID,
			Name:        card.Name,
			ImageURL:    imageURL,
			DomainID:    card.DomainID,
			DomainName:  card.DomainName,
			Level:       card.Level,
			Type:        card.Type,
			RecallCost:  card.RecallCost,
			FeatureText: card.FeatureText,
		})
	}
	return mapped
}

// resolveEntityImageURL resolves a CDN image URL for a daggerheart entity.
// Returns empty string when CDN is nil or the entity has no mapped asset.
func resolveEntityImageURL(cdn imagecdn.ImageCDN, entityType, entityID, assetType string) string {
	if cdn == nil {
		return ""
	}
	manifest := catalog.DaggerheartAssetsManifest()
	resolved := manifest.ResolveEntityAsset(entityType, entityID, assetType)
	if resolved.CDNAssetID == "" {
		return ""
	}
	url, err := cdn.URL(imagecdn.Request{AssetID: resolved.CDNAssetID, Extension: ".png", Delivery: &imagecdn.Delivery{WidthPX: 384}})
	if err != nil {
		slog.Debug("cdn resolution failed", "entity_id", entityID, "asset_type", assetType, "error", err)
		return ""
	}
	return url
}
