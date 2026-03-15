package daggerheart

import (
	"log/slog"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

const daggerheartSecondaryNoneWeaponAssetEntityID = "weapon.no-secondary"

// CreationView maps the domain creation model to the workflow-owned creation
// page model.
func (w Workflow) CreationView(creation catalogCreation) campaignworkflow.CharacterCreationView {
	view := newCreationView(creation)
	view.Steps = mapCreationSteps(creation.Progress.Steps)

	cdn := creationImageCDN(w.AssetBaseURL)
	domainByID := domainLookupByID(creation.Domains)

	view.Classes = mapCreationClasses(creation.Classes, domainByID, cdn)
	view.Subclasses = mapCreationSubclasses(creation.Subclasses, cdn)
	view.Ancestries = mapCreationHeritages(creation.Ancestries, catalog.DaggerheartEntityTypeAncestry, catalog.DaggerheartAssetTypeAncestryIllustration, cdn)
	view.Communities = mapCreationHeritages(creation.Communities, catalog.DaggerheartEntityTypeCommunity, catalog.DaggerheartAssetTypeCommunityIllustration, cdn)
	view.PrimaryWeapons = mapCreationWeapons(creation.PrimaryWeapons, cdn)
	view.SecondaryWeapons = mapCreationWeapons(creation.SecondaryWeapons, cdn)
	view.SecondaryWeaponNoneImageURL = creationSecondaryNoneImageURL(cdn)
	view.Armor = mapCreationArmor(creation.Armor, cdn)
	view.PotionItems = mapCreationItems(creation.PotionItems, cdn)
	view.DomainCards = mapCreationDomainCards(creation.DomainCards, cdn)
	view.NextStepPrefetchURLs = creationNextStepPrefetchURLs(view)

	return view
}

// domainView supports class-domain label and watermark assembly.
type domainView struct {
	Name    string
	IconURL string
}

// domainLookupByID builds domain metadata lookup used across class/card mapping.
func domainLookupByID(domains []campaignworkflow.Domain) map[string]domainView {
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
func newCreationView(creation catalogCreation) campaignworkflow.CharacterCreationView {
	return campaignworkflow.CharacterCreationView{
		Ready:                       creation.Progress.Ready,
		NextStep:                    creation.Progress.NextStep,
		UnmetReasons:                append([]string(nil), creation.Progress.UnmetReasons...),
		ClassID:                     creation.Profile.ClassID,
		SubclassID:                  creation.Profile.SubclassID,
		AncestryID:                  creation.Profile.AncestryID,
		CommunityID:                 creation.Profile.CommunityID,
		Agility:                     creation.Profile.Agility,
		Strength:                    creation.Profile.Strength,
		Finesse:                     creation.Profile.Finesse,
		Instinct:                    creation.Profile.Instinct,
		Presence:                    creation.Profile.Presence,
		Knowledge:                   creation.Profile.Knowledge,
		PrimaryWeaponID:             creation.Profile.PrimaryWeaponID,
		SecondaryWeaponID:           creation.Profile.SecondaryWeaponID,
		ArmorID:                     creation.Profile.ArmorID,
		PotionItemID:                creation.Profile.PotionItemID,
		Description:                 creation.Profile.Description,
		Background:                  creation.Profile.Background,
		Experiences:                 mapCreationExperiences(creation.Profile.Experiences),
		DomainCardIDs:               append([]string(nil), creation.Profile.DomainCardIDs...),
		Connections:                 creation.Profile.Connections,
		Steps:                       nil,
		Classes:                     nil,
		Subclasses:                  nil,
		Ancestries:                  nil,
		Communities:                 nil,
		PrimaryWeapons:              nil,
		SecondaryWeapons:            nil,
		SecondaryWeaponNoneImageURL: "",
		Armor:                       nil,
		PotionItems:                 nil,
		DomainCards:                 nil,
		NextStepPrefetchURLs:        nil,
	}
}

// creationNextStepPrefetchURLs derives immediate next-step images for client-side prewarming.
func creationNextStepPrefetchURLs(view campaignworkflow.CharacterCreationView) []string {
	urls := []string{}
	switch view.NextStep {
	case 1:
		urls = append(urls, creationHeritageImageURLs(view.Ancestries)...)
		urls = append(urls, creationHeritageImageURLs(view.Communities)...)
	case 3:
		urls = append(urls, creationWeaponImageURLs(view.PrimaryWeapons)...)
		urls = append(urls, creationWeaponImageURLs(view.SecondaryWeapons)...)
		urls = append(urls, creationArmorImageURLs(view.Armor)...)
		urls = append(urls, creationItemImageURLs(view.PotionItems)...)
	case 5:
		urls = append(urls, creationDomainCardImageURLs(view.DomainCards)...)
	}
	return dedupeNonEmptyURLs(urls)
}

// creationHeritageImageURLs extracts heritage illustrations for the next-step prefetch list.
func creationHeritageImageURLs(items []campaignworkflow.CreationHeritageView) []string {
	urls := make([]string, 0, len(items))
	for _, item := range items {
		urls = append(urls, strings.TrimSpace(item.ImageURL))
	}
	return urls
}

// creationWeaponImageURLs extracts weapon illustrations for the next-step prefetch list.
func creationWeaponImageURLs(items []campaignworkflow.CreationWeaponView) []string {
	urls := make([]string, 0, len(items))
	for _, item := range items {
		urls = append(urls, strings.TrimSpace(item.ImageURL))
	}
	return urls
}

// creationSecondaryNoneImageURL resolves the synthetic secondary-none card art.
func creationSecondaryNoneImageURL(cdn imagecdn.ImageCDN) string {
	return resolveEntityImageURL(
		cdn,
		catalog.DaggerheartEntityTypeWeapon,
		daggerheartSecondaryNoneWeaponAssetEntityID,
		catalog.DaggerheartAssetTypeWeaponIllustration,
	)
}

// creationArmorImageURLs extracts armor illustrations for the next-step prefetch list.
func creationArmorImageURLs(items []campaignworkflow.CreationArmorView) []string {
	urls := make([]string, 0, len(items))
	for _, item := range items {
		urls = append(urls, strings.TrimSpace(item.ImageURL))
	}
	return urls
}

// creationItemImageURLs extracts item illustrations for the next-step prefetch list.
func creationItemImageURLs(items []campaignworkflow.CreationItemView) []string {
	urls := make([]string, 0, len(items))
	for _, item := range items {
		urls = append(urls, strings.TrimSpace(item.ImageURL))
	}
	return urls
}

// creationDomainCardImageURLs extracts domain-card illustrations for the next-step prefetch list.
func creationDomainCardImageURLs(items []campaignworkflow.CreationDomainCardView) []string {
	urls := make([]string, 0, len(items))
	for _, item := range items {
		urls = append(urls, strings.TrimSpace(item.ImageURL))
	}
	return urls
}

// dedupeNonEmptyURLs keeps prefetch metadata stable so the browser avoids redundant image warmups.
func dedupeNonEmptyURLs(urls []string) []string {
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(urls))
	for _, raw := range urls {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		deduped = append(deduped, url)
	}
	return deduped
}

// mapCreationExperiences maps profile experiences to template view rows.
func mapCreationExperiences(experiences []campaignworkflow.Experience) []campaignworkflow.CreationExperienceView {
	mapped := make([]campaignworkflow.CreationExperienceView, 0, len(experiences))
	for _, exp := range experiences {
		mapped = append(mapped, campaignworkflow.CreationExperienceView{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}
	return mapped
}

// mapCreationSteps maps progress step state to template view rows.
func mapCreationSteps(steps []campaignworkflow.Step) []campaignworkflow.CharacterCreationStepView {
	mapped := make([]campaignworkflow.CharacterCreationStepView, 0, len(steps))
	for _, step := range steps {
		mapped = append(mapped, campaignworkflow.CharacterCreationStepView{
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
func mapCreationClasses(classes []campaignworkflow.Class, domainByID map[string]domainView, cdn imagecdn.ImageCDN) []campaignworkflow.CreationClassView {
	mapped := make([]campaignworkflow.CreationClassView, 0, len(classes))
	for _, class := range classes {
		domainNames, domainWatermarks := mapClassDomains(class.DomainIDs, domainByID)
		mapped = append(mapped, campaignworkflow.CreationClassView{
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
func mapClassDomains(domainIDs []string, domainByID map[string]domainView) ([]string, []campaignworkflow.CreationDomainWatermarkView) {
	names := make([]string, 0, len(domainIDs))
	watermarks := make([]campaignworkflow.CreationDomainWatermarkView, 0, 2)
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
			watermarks = append(watermarks, campaignworkflow.CreationDomainWatermarkView{
				ID:      trimmedDomainID,
				Name:    domain.Name,
				IconURL: domain.IconURL,
			})
		}
	}
	return names, watermarks
}

// mapCreationSubclasses maps subclass catalog entries to template view rows.
func mapCreationSubclasses(subclasses []campaignworkflow.Subclass, cdn imagecdn.ImageCDN) []campaignworkflow.CreationSubclassView {
	mapped := make([]campaignworkflow.CreationSubclassView, 0, len(subclasses))
	for _, subclass := range subclasses {
		mapped = append(mapped, campaignworkflow.CreationSubclassView{
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
	heritages []campaignworkflow.Heritage,
	entityType string,
	assetType string,
	cdn imagecdn.ImageCDN,
) []campaignworkflow.CreationHeritageView {
	mapped := make([]campaignworkflow.CreationHeritageView, 0, len(heritages))
	for _, heritage := range heritages {
		mapped = append(mapped, campaignworkflow.CreationHeritageView{
			ID:       heritage.ID,
			Name:     heritage.Name,
			ImageURL: resolveEntityImageURL(cdn, entityType, heritage.ID, assetType),
			Features: mapFeatures(heritage.Features),
		})
	}
	return mapped
}

// mapFeatures maps catalog features to template feature rows.
func mapFeatures(features []campaignworkflow.Feature) []campaignworkflow.CreationClassFeatureView {
	mapped := make([]campaignworkflow.CreationClassFeatureView, 0, len(features))
	for _, feature := range features {
		mapped = append(mapped, mapFeature(feature))
	}
	return mapped
}

// mapFeature maps one catalog feature to a template feature row.
func mapFeature(feature campaignworkflow.Feature) campaignworkflow.CreationClassFeatureView {
	return campaignworkflow.CreationClassFeatureView{
		Name:        feature.Name,
		Description: feature.Description,
	}
}

// mapCreationWeapons maps weapon catalog entries to template rows.
func mapCreationWeapons(weapons []campaignworkflow.Weapon, cdn imagecdn.ImageCDN) []campaignworkflow.CreationWeaponView {
	mapped := make([]campaignworkflow.CreationWeaponView, 0, len(weapons))
	for _, weapon := range weapons {
		imageURL := strings.TrimSpace(weapon.Illustration.URL)
		if imageURL == "" {
			imageURL = resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeWeapon, weapon.ID, catalog.DaggerheartAssetTypeWeaponIllustration)
		}
		mapped = append(mapped, campaignworkflow.CreationWeaponView{
			ID:       weapon.ID,
			Name:     weapon.Name,
			ImageURL: imageURL,
			Burden:   weapon.Burden,
			Trait:    weapon.Trait,
			Range:    weapon.Range,
			Damage:   weapon.Damage,
			Feature:  weapon.Feature,
		})
	}
	return mapped
}

// mapCreationArmor maps armor catalog entries to template rows.
func mapCreationArmor(items []campaignworkflow.Armor, cdn imagecdn.ImageCDN) []campaignworkflow.CreationArmorView {
	mapped := make([]campaignworkflow.CreationArmorView, 0, len(items))
	for _, item := range items {
		imageURL := strings.TrimSpace(item.Illustration.URL)
		if imageURL == "" {
			imageURL = resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeArmor, item.ID, catalog.DaggerheartAssetTypeArmorIllustration)
		}
		mapped = append(mapped, campaignworkflow.CreationArmorView{
			ID:             item.ID,
			Name:           item.Name,
			ImageURL:       imageURL,
			ArmorScore:     item.ArmorScore,
			BaseThresholds: item.BaseThresholds,
			Feature:        item.Feature,
		})
	}
	return mapped
}

// mapCreationItems maps item catalog entries to template rows.
func mapCreationItems(items []campaignworkflow.Item, cdn imagecdn.ImageCDN) []campaignworkflow.CreationItemView {
	mapped := make([]campaignworkflow.CreationItemView, 0, len(items))
	for _, item := range items {
		imageURL := strings.TrimSpace(item.Illustration.URL)
		if imageURL == "" {
			imageURL = resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeItem, item.ID, catalog.DaggerheartAssetTypeItemIllustration)
		}
		mapped = append(mapped, campaignworkflow.CreationItemView{
			ID:          item.ID,
			Name:        item.Name,
			ImageURL:    imageURL,
			Description: item.Description,
		})
	}
	return mapped
}

// mapCreationDomainCards maps domain-card entries to template rows.
func mapCreationDomainCards(cards []campaignworkflow.DomainCard, cdn imagecdn.ImageCDN) []campaignworkflow.CreationDomainCardView {
	mapped := make([]campaignworkflow.CreationDomainCardView, 0, len(cards))
	for _, card := range cards {
		imageURL := strings.TrimSpace(card.Illustration.URL)
		if imageURL == "" {
			imageURL = resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeDomainCard, card.ID, catalog.DaggerheartAssetTypeDomainCardIllustration)
		}
		mapped = append(mapped, campaignworkflow.CreationDomainCardView{
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
