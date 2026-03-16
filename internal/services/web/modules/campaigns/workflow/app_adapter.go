package workflow

import (
	"context"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"golang.org/x/text/language"
)

// pageAppAdapter maps app-owned creation reads into workflow-owned page inputs.
type pageAppAdapter struct {
	app campaignapp.CampaignCharacterCreationPageService
}

// mutationAppAdapter maps app-owned creation mutations into workflow-owned mutation inputs.
type mutationAppAdapter struct {
	app campaignapp.CampaignCharacterCreationMutationService
}

// NewPageAppService adapts the campaigns app page seam to the workflow-owned page contract.
func NewPageAppService(app campaignapp.CampaignCharacterCreationPageService) PageAppService {
	if app == nil {
		return nil
	}
	return pageAppAdapter{app: app}
}

// NewMutationAppService adapts the campaigns app mutation seam to the workflow-owned mutation contract.
func NewMutationAppService(app campaignapp.CampaignCharacterCreationMutationService) MutationAppService {
	if app == nil {
		return nil
	}
	return mutationAppAdapter{app: app}
}

// CampaignCharacterCreationProgress forwards the app read and normalizes it to workflow-owned progress data.
func (a pageAppAdapter) CampaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (Progress, error) {
	progress, err := a.app.CampaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		return Progress{}, err
	}
	return mapProgress(progress), nil
}

// CampaignCharacterCreationCatalog forwards the app read and normalizes it to workflow-owned catalog data.
func (a pageAppAdapter) CampaignCharacterCreationCatalog(ctx context.Context, locale language.Tag) (Catalog, error) {
	catalog, err := a.app.CampaignCharacterCreationCatalog(ctx, locale)
	if err != nil {
		return Catalog{}, err
	}
	return mapCatalog(catalog), nil
}

// CampaignCharacterCreationProfile forwards the app read and normalizes it to workflow-owned profile data.
func (a pageAppAdapter) CampaignCharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (Profile, error) {
	profile, err := a.app.CampaignCharacterCreationProfile(ctx, campaignID, characterID)
	if err != nil {
		return Profile{}, err
	}
	return mapProfile(profile), nil
}

// CampaignCharacterCreationProgress forwards the app read and normalizes it to workflow-owned progress data.
func (a mutationAppAdapter) CampaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (Progress, error) {
	progress, err := a.app.CampaignCharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		return Progress{}, err
	}
	return mapProgress(progress), nil
}

// ApplyCharacterCreationStep forwards workflow step mutations to the campaigns app seam.
func (a mutationAppAdapter) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *StepInput) error {
	return a.app.ApplyCharacterCreationStep(ctx, campaignID, characterID, step)
}

// ResetCharacterCreationWorkflow forwards workflow reset mutations to the campaigns app seam.
func (a mutationAppAdapter) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	return a.app.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}

// mapProgress copies app progress into the workflow-owned page contract.
func mapProgress(progress campaignapp.CampaignCharacterCreationProgress) Progress {
	steps := make([]Step, 0, len(progress.Steps))
	for _, step := range progress.Steps {
		steps = append(steps, Step{Step: step.Step, Key: step.Key, Complete: step.Complete})
	}
	return Progress{
		Steps:        steps,
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}

// mapCatalog copies app catalog data into the workflow-owned catalog contract.
func mapCatalog(catalog campaignapp.CampaignCharacterCreationCatalog) Catalog {
	return Catalog{
		AssetTheme:           catalog.AssetTheme,
		Classes:              mapClasses(catalog.Classes),
		Subclasses:           mapSubclasses(catalog.Subclasses),
		Heritages:            mapHeritages(catalog.Heritages),
		CompanionExperiences: mapCompanionExperiences(catalog.CompanionExperiences),
		Domains:              mapDomains(catalog.Domains),
		Weapons:              mapWeapons(catalog.Weapons),
		Armor:                mapArmor(catalog.Armor),
		Items:                mapItems(catalog.Items),
		DomainCards:          mapDomainCards(catalog.DomainCards),
		Adversaries:          mapAdversaries(catalog.Adversaries),
		Environments:         mapEnvironments(catalog.Environments),
	}
}

// mapProfile copies app profile data into the workflow-owned profile contract.
func mapProfile(profile campaignapp.CampaignCharacterCreationProfile) Profile {
	return Profile{
		CharacterName:                profile.CharacterName,
		ClassID:                      profile.ClassID,
		SubclassID:                   profile.SubclassID,
		SubclassCreationRequirements: append([]string(nil), profile.SubclassCreationRequirements...),
		Heritage:                     mapHeritageSelection(profile.Heritage),
		CompanionSheet:               mapCompanionSheet(profile.CompanionSheet),
		Agility:                      profile.Agility,
		Strength:                     profile.Strength,
		Finesse:                      profile.Finesse,
		Instinct:                     profile.Instinct,
		Presence:                     profile.Presence,
		Knowledge:                    profile.Knowledge,
		PrimaryWeaponID:              profile.PrimaryWeaponID,
		SecondaryWeaponID:            profile.SecondaryWeaponID,
		ArmorID:                      profile.ArmorID,
		PotionItemID:                 profile.PotionItemID,
		Background:                   profile.Background,
		Description:                  profile.Description,
		Experiences:                  mapExperiences(profile.Experiences),
		DomainCardIDs:                append([]string(nil), profile.DomainCardIDs...),
		Connections:                  profile.Connections,
	}
}

// mapFeatures copies catalog feature rows into the workflow-owned feature slice.
func mapFeatures(features []campaignapp.CatalogFeature) []Feature {
	mapped := make([]Feature, 0, len(features))
	for _, feature := range features {
		mapped = append(mapped, Feature{Name: feature.Name, Description: feature.Description})
	}
	return mapped
}

// mapCompanionExperiences preserves companion experience identity while
// dropping the app-owned DTO wrapper at the workflow boundary.
func mapCompanionExperiences(experiences []campaignapp.CatalogCompanionExperience) []CompanionExperience {
	mapped := make([]CompanionExperience, 0, len(experiences))
	for _, experience := range experiences {
		mapped = append(mapped, CompanionExperience{
			ID:          experience.ID,
			Name:        experience.Name,
			Description: experience.Description,
		})
	}
	return mapped
}

// mapAssetReference preserves asset identity while dropping the app-owned DTO type.
func mapAssetReference(asset campaignapp.CatalogAssetReference) AssetReference {
	return AssetReference{
		URL:     asset.URL,
		Status:  asset.Status,
		SetID:   asset.SetID,
		AssetID: asset.AssetID,
	}
}

// mapClasses copies workflow class options out of the app-owned catalog DTO.
func mapClasses(classes []campaignapp.CatalogClass) []Class {
	mapped := make([]Class, 0, len(classes))
	for _, class := range classes {
		mapped = append(mapped, Class{
			ID:              class.ID,
			Name:            class.Name,
			DomainIDs:       append([]string(nil), class.DomainIDs...),
			StartingHP:      class.StartingHP,
			StartingEvasion: class.StartingEvasion,
			HopeFeature:     Feature{Name: class.HopeFeature.Name, Description: class.HopeFeature.Description},
			Features:        mapFeatures(class.Features),
			Illustration:    mapAssetReference(class.Illustration),
			Icon:            mapAssetReference(class.Icon),
		})
	}
	return mapped
}

// mapSubclasses copies workflow subclass options out of the app-owned catalog DTO.
func mapSubclasses(subclasses []campaignapp.CatalogSubclass) []Subclass {
	mapped := make([]Subclass, 0, len(subclasses))
	for _, subclass := range subclasses {
		mapped = append(mapped, Subclass{
			ID:                   subclass.ID,
			Name:                 subclass.Name,
			ClassID:              subclass.ClassID,
			SpellcastTrait:       subclass.SpellcastTrait,
			CreationRequirements: append([]string(nil), subclass.CreationRequirements...),
			Foundation:           mapFeatures(subclass.Foundation),
			Illustration:         mapAssetReference(subclass.Illustration),
		})
	}
	return mapped
}

// mapHeritageSelection copies app profile heritage into the workflow-owned contract.
func mapHeritageSelection(selection campaignapp.CampaignCharacterCreationHeritageSelection) HeritageSelection {
	return HeritageSelection{
		AncestryLabel:           selection.AncestryLabel,
		FirstFeatureAncestryID:  selection.FirstFeatureAncestryID,
		FirstFeatureID:          selection.FirstFeatureID,
		SecondFeatureAncestryID: selection.SecondFeatureAncestryID,
		SecondFeatureID:         selection.SecondFeatureID,
		CommunityID:             selection.CommunityID,
	}
}

// mapCompanionSheet copies the app companion sheet into the workflow-owned contract.
func mapCompanionSheet(sheet *campaignapp.CampaignCharacterCreationCompanionSheet) *CompanionSheet {
	if sheet == nil {
		return nil
	}
	return &CompanionSheet{
		AnimalKind:        sheet.AnimalKind,
		Name:              sheet.Name,
		Evasion:           sheet.Evasion,
		Experiences:       mapExperiences(sheet.Experiences),
		AttackDescription: sheet.AttackDescription,
		AttackRange:       sheet.AttackRange,
		DamageDieSides:    sheet.DamageDieSides,
		DamageType:        sheet.DamageType,
	}
}

// mapHeritages copies workflow heritage options out of the app-owned catalog DTO.
func mapHeritages(heritages []campaignapp.CatalogHeritage) []Heritage {
	mapped := make([]Heritage, 0, len(heritages))
	for _, heritage := range heritages {
		mapped = append(mapped, Heritage{
			ID:           heritage.ID,
			Name:         heritage.Name,
			Kind:         heritage.Kind,
			Features:     mapFeatures(heritage.Features),
			Illustration: mapAssetReference(heritage.Illustration),
		})
	}
	return mapped
}

// mapDomains copies workflow domain metadata out of the app-owned catalog DTO.
func mapDomains(domains []campaignapp.CatalogDomain) []Domain {
	mapped := make([]Domain, 0, len(domains))
	for _, domain := range domains {
		mapped = append(mapped, Domain{
			ID:           domain.ID,
			Name:         domain.Name,
			Illustration: mapAssetReference(domain.Illustration),
			Icon:         mapAssetReference(domain.Icon),
		})
	}
	return mapped
}

// mapWeapons copies workflow weapon options out of the app-owned catalog DTO.
func mapWeapons(weapons []campaignapp.CatalogWeapon) []Weapon {
	mapped := make([]Weapon, 0, len(weapons))
	for _, weapon := range weapons {
		mapped = append(mapped, Weapon{
			ID:           weapon.ID,
			Name:         weapon.Name,
			Category:     weapon.Category,
			Tier:         weapon.Tier,
			Burden:       weapon.Burden,
			Trait:        weapon.Trait,
			Range:        weapon.Range,
			Damage:       weapon.Damage,
			Feature:      weapon.Feature,
			Illustration: mapAssetReference(weapon.Illustration),
		})
	}
	return mapped
}

// mapArmor copies workflow armor options out of the app-owned catalog DTO.
func mapArmor(armor []campaignapp.CatalogArmor) []Armor {
	mapped := make([]Armor, 0, len(armor))
	for _, entry := range armor {
		mapped = append(mapped, Armor{
			ID:             entry.ID,
			Name:           entry.Name,
			Tier:           entry.Tier,
			ArmorScore:     entry.ArmorScore,
			BaseThresholds: entry.BaseThresholds,
			Feature:        entry.Feature,
			Illustration:   mapAssetReference(entry.Illustration),
		})
	}
	return mapped
}

// mapItems copies workflow item options out of the app-owned catalog DTO.
func mapItems(items []campaignapp.CatalogItem) []Item {
	mapped := make([]Item, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, Item{
			ID:           item.ID,
			Name:         item.Name,
			Description:  item.Description,
			Illustration: mapAssetReference(item.Illustration),
		})
	}
	return mapped
}

// mapDomainCards copies workflow domain-card options out of the app-owned catalog DTO.
func mapDomainCards(cards []campaignapp.CatalogDomainCard) []DomainCard {
	mapped := make([]DomainCard, 0, len(cards))
	for _, card := range cards {
		mapped = append(mapped, DomainCard{
			ID:           card.ID,
			Name:         card.Name,
			DomainID:     card.DomainID,
			DomainName:   card.DomainName,
			Level:        card.Level,
			Type:         card.Type,
			RecallCost:   card.RecallCost,
			FeatureText:  card.FeatureText,
			Illustration: mapAssetReference(card.Illustration),
		})
	}
	return mapped
}

// mapAdversaries copies adversary catalog rows into workflow-owned data.
func mapAdversaries(adversaries []campaignapp.CatalogAdversary) []Adversary {
	mapped := make([]Adversary, 0, len(adversaries))
	for _, adversary := range adversaries {
		mapped = append(mapped, Adversary{
			ID:           adversary.ID,
			Name:         adversary.Name,
			Illustration: mapAssetReference(adversary.Illustration),
		})
	}
	return mapped
}

// mapEnvironments copies environment catalog rows into workflow-owned data.
func mapEnvironments(environments []campaignapp.CatalogEnvironment) []Environment {
	mapped := make([]Environment, 0, len(environments))
	for _, environment := range environments {
		mapped = append(mapped, Environment{
			ID:           environment.ID,
			Name:         environment.Name,
			Illustration: mapAssetReference(environment.Illustration),
		})
	}
	return mapped
}

// mapExperiences copies profile experiences into the workflow-owned profile contract.
func mapExperiences(experiences []campaignapp.CampaignCharacterCreationExperience) []Experience {
	mapped := make([]Experience, 0, len(experiences))
	for _, experience := range experiences {
		mapped = append(mapped, Experience{Name: experience.Name, Modifier: experience.Modifier})
	}
	return mapped
}
