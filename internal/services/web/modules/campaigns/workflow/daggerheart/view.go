package daggerheart

import (
	"log/slog"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CreationView maps the domain creation model to the template view type.
func (w Workflow) CreationView(creation campaignapp.CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView {
	// Build domain name lookup for resolving class domain names.
	domainNameByID := make(map[string]string, len(creation.Domains))
	for _, domain := range creation.Domains {
		domainNameByID[domain.ID] = domain.Name
	}

	experiences := make([]webtemplates.CampaignCreationExperienceView, 0, len(creation.Profile.Experiences))
	for _, exp := range creation.Profile.Experiences {
		experiences = append(experiences, webtemplates.CampaignCreationExperienceView{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}

	view := webtemplates.CampaignCharacterCreationView{
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
		Description:       creation.Profile.Description,
		Experiences:       experiences,
		DomainCardIDs:     append([]string(nil), creation.Profile.DomainCardIDs...),
		Connections:       creation.Profile.Connections,
		Steps:             make([]webtemplates.CampaignCharacterCreationStepView, 0, len(creation.Progress.Steps)),
		Classes:           make([]webtemplates.CampaignCreationClassView, 0, len(creation.Classes)),
		Subclasses:        make([]webtemplates.CampaignCreationSubclassView, 0, len(creation.Subclasses)),
		Ancestries:        make([]webtemplates.CampaignCreationHeritageView, 0, len(creation.Ancestries)),
		Communities:       make([]webtemplates.CampaignCreationHeritageView, 0, len(creation.Communities)),
		PrimaryWeapons:    make([]webtemplates.CampaignCreationWeaponView, 0, len(creation.PrimaryWeapons)),
		SecondaryWeapons:  make([]webtemplates.CampaignCreationWeaponView, 0, len(creation.SecondaryWeapons)),
		Armor:             make([]webtemplates.CampaignCreationArmorView, 0, len(creation.Armor)),
		PotionItems:       make([]webtemplates.CampaignCreationItemView, 0, len(creation.PotionItems)),
		DomainCards:       make([]webtemplates.CampaignCreationDomainCardView, 0, len(creation.DomainCards)),
	}
	for _, step := range creation.Progress.Steps {
		view.Steps = append(view.Steps, webtemplates.CampaignCharacterCreationStepView{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	var cdn imagecdn.ImageCDN
	if w.AssetBaseURL != "" {
		cdn = imagecdn.New(w.AssetBaseURL)
	}
	for _, class := range creation.Classes {
		domainNames := make([]string, 0, len(class.DomainIDs))
		for _, domainID := range class.DomainIDs {
			if name, ok := domainNameByID[domainID]; ok {
				domainNames = append(domainNames, name)
			}
		}
		imageURL := resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeClass, class.ID, catalog.DaggerheartAssetTypeClassIllustration)
		hopeFeature := webtemplates.CampaignCreationClassFeatureView{
			Name:        class.HopeFeature.Name,
			Description: class.HopeFeature.Description,
		}
		features := make([]webtemplates.CampaignCreationClassFeatureView, 0, len(class.Features))
		for _, f := range class.Features {
			features = append(features, webtemplates.CampaignCreationClassFeatureView{
				Name:        f.Name,
				Description: f.Description,
			})
		}
		view.Classes = append(view.Classes, webtemplates.CampaignCreationClassView{
			ID:              class.ID,
			Name:            class.Name,
			ImageURL:        imageURL,
			StartingHP:      class.StartingHP,
			StartingEvasion: class.StartingEvasion,
			HopeFeature:     hopeFeature,
			Features:        features,
			DomainNames:     domainNames,
		})
	}
	for _, subclass := range creation.Subclasses {
		foundation := make([]webtemplates.CampaignCreationClassFeatureView, 0, len(subclass.Foundation))
		for _, f := range subclass.Foundation {
			foundation = append(foundation, webtemplates.CampaignCreationClassFeatureView{
				Name:        f.Name,
				Description: f.Description,
			})
		}
		subclassImageURL := resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeSubclass, subclass.ID, catalog.DaggerheartAssetTypeSubclassIllustration)
		view.Subclasses = append(view.Subclasses, webtemplates.CampaignCreationSubclassView{
			ID:             subclass.ID,
			Name:           subclass.Name,
			ImageURL:       subclassImageURL,
			ClassID:        subclass.ClassID,
			SpellcastTrait: subclass.SpellcastTrait,
			Foundation:     foundation,
		})
	}
	for _, ancestry := range creation.Ancestries {
		ancestryImageURL := resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeAncestry, ancestry.ID, catalog.DaggerheartAssetTypeAncestryIllustration)
		ancestryFeatures := make([]webtemplates.CampaignCreationClassFeatureView, 0, len(ancestry.Features))
		for _, f := range ancestry.Features {
			ancestryFeatures = append(ancestryFeatures, webtemplates.CampaignCreationClassFeatureView{
				Name:        f.Name,
				Description: f.Description,
			})
		}
		view.Ancestries = append(view.Ancestries, webtemplates.CampaignCreationHeritageView{
			ID:       ancestry.ID,
			Name:     ancestry.Name,
			ImageURL: ancestryImageURL,
			Features: ancestryFeatures,
		})
	}
	for _, community := range creation.Communities {
		communityImageURL := resolveEntityImageURL(cdn, catalog.DaggerheartEntityTypeCommunity, community.ID, catalog.DaggerheartAssetTypeCommunityIllustration)
		communityFeatures := make([]webtemplates.CampaignCreationClassFeatureView, 0, len(community.Features))
		for _, f := range community.Features {
			communityFeatures = append(communityFeatures, webtemplates.CampaignCreationClassFeatureView{
				Name:        f.Name,
				Description: f.Description,
			})
		}
		view.Communities = append(view.Communities, webtemplates.CampaignCreationHeritageView{
			ID:       community.ID,
			Name:     community.Name,
			ImageURL: communityImageURL,
			Features: communityFeatures,
		})
	}
	for _, weapon := range creation.PrimaryWeapons {
		view.PrimaryWeapons = append(view.PrimaryWeapons, webtemplates.CampaignCreationWeaponView{
			ID:      weapon.ID,
			Name:    weapon.Name,
			Trait:   weapon.Trait,
			Range:   weapon.Range,
			Damage:  weapon.Damage,
			Feature: weapon.Feature,
		})
	}
	for _, weapon := range creation.SecondaryWeapons {
		view.SecondaryWeapons = append(view.SecondaryWeapons, webtemplates.CampaignCreationWeaponView{
			ID:      weapon.ID,
			Name:    weapon.Name,
			Trait:   weapon.Trait,
			Range:   weapon.Range,
			Damage:  weapon.Damage,
			Feature: weapon.Feature,
		})
	}
	for _, armor := range creation.Armor {
		view.Armor = append(view.Armor, webtemplates.CampaignCreationArmorView{
			ID:             armor.ID,
			Name:           armor.Name,
			ArmorScore:     armor.ArmorScore,
			BaseThresholds: armor.BaseThresholds,
			Feature:        armor.Feature,
		})
	}
	for _, item := range creation.PotionItems {
		view.PotionItems = append(view.PotionItems, webtemplates.CampaignCreationItemView{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	for _, card := range creation.DomainCards {
		view.DomainCards = append(view.DomainCards, webtemplates.CampaignCreationDomainCardView{
			ID:          card.ID,
			Name:        card.Name,
			DomainID:    card.DomainID,
			DomainName:  card.DomainName,
			Level:       card.Level,
			Type:        card.Type,
			RecallCost:  card.RecallCost,
			FeatureText: card.FeatureText,
		})
	}
	return view
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
