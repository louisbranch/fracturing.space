package daggerheart

import (
	"context"
	"strconv"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const (
	contentTypeClass               = "class"
	contentTypeSubclass            = "subclass"
	contentTypeHeritage            = "heritage"
	contentTypeExperience          = "experience"
	contentTypeAdversary           = "adversary"
	contentTypeBeastform           = "beastform"
	contentTypeCompanionExperience = "companion_experience"
	contentTypeLootEntry           = "loot_entry"
	contentTypeDamageType          = "damage_type"
	contentTypeDomain              = "domain"
	contentTypeDomainCard          = "domain_card"
	contentTypeWeapon              = "weapon"
	contentTypeArmor               = "armor"
	contentTypeItem                = "item"
	contentTypeEnvironment         = "environment"
	contentTypeFeature             = "feature"
	contentTypeHopeFeature         = "hope_feature"
	contentTypeAdversaryFeature    = "adversary_feature"
	contentTypeBeastformFeature    = "beastform_feature"
)

type contentStringKey struct {
	ContentID string
	Field     string
}

type contentStringLookup map[contentStringKey]string

func localeString(locale commonv1.Locale) (string, bool) {
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return "", false
	}
	return i18n.LocaleString(i18n.NormalizeLocale(locale)), true
}

func fetchContentStrings(ctx context.Context, store storage.DaggerheartContentStore, contentType string, ids []string, locale string) (contentStringLookup, error) {
	if store == nil || len(ids) == 0 || locale == "" {
		return contentStringLookup{}, nil
	}
	entries, err := store.ListDaggerheartContentStrings(ctx, contentType, ids, locale)
	if err != nil {
		return nil, err
	}
	lookup := make(contentStringLookup, len(entries))
	for _, entry := range entries {
		lookup[contentStringKey{ContentID: entry.ContentID, Field: entry.Field}] = entry.Text
	}
	return lookup, nil
}

func applyLocalizedString(lookup contentStringLookup, contentID, field string, target *string) {
	if target == nil {
		return
	}
	if text, ok := lookup[contentStringKey{ContentID: contentID, Field: field}]; ok {
		*target = text
	}
}

func collectFeatureIDs(features []storage.DaggerheartFeature) []string {
	ids := make([]string, 0, len(features))
	for _, feature := range features {
		if feature.ID != "" {
			ids = append(ids, feature.ID)
		}
	}
	return ids
}

func applyLocalizedFeatures(features []storage.DaggerheartFeature, lookup contentStringLookup) {
	for i := range features {
		applyLocalizedString(lookup, features[i].ID, "name", &features[i].Name)
		applyLocalizedString(lookup, features[i].ID, "description", &features[i].Description)
	}
}

func applyLocalizedAdversaryFeatures(features []storage.DaggerheartAdversaryFeature, lookup contentStringLookup) {
	for i := range features {
		applyLocalizedString(lookup, features[i].ID, "name", &features[i].Name)
		applyLocalizedString(lookup, features[i].ID, "description", &features[i].Description)
	}
}

func applyLocalizedBeastformFeatures(features []storage.DaggerheartBeastformFeature, lookup contentStringLookup) {
	for i := range features {
		applyLocalizedString(lookup, features[i].ID, "name", &features[i].Name)
		applyLocalizedString(lookup, features[i].ID, "description", &features[i].Description)
	}
}

func localizeClasses(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, classes []storage.DaggerheartClass) error {
	localeValue, ok := localeString(locale)
	if !ok || len(classes) == 0 {
		return nil
	}
	classIDs := make([]string, 0, len(classes))
	featureIDs := make([]string, 0)
	hopeIDs := make([]string, 0, len(classes))
	for _, class := range classes {
		classIDs = append(classIDs, class.ID)
		featureIDs = append(featureIDs, collectFeatureIDs(class.Features)...)
		hopeIDs = append(hopeIDs, "hope_feature:"+class.ID)
	}
	classLookup, err := fetchContentStrings(ctx, store, contentTypeClass, classIDs, localeValue)
	if err != nil {
		return err
	}
	featureLookup, err := fetchContentStrings(ctx, store, contentTypeFeature, featureIDs, localeValue)
	if err != nil {
		return err
	}
	hopeLookup, err := fetchContentStrings(ctx, store, contentTypeHopeFeature, hopeIDs, localeValue)
	if err != nil {
		return err
	}
	for i := range classes {
		applyLocalizedString(classLookup, classes[i].ID, "name", &classes[i].Name)
		applyLocalizedFeatures(classes[i].Features, featureLookup)
		hopeID := "hope_feature:" + classes[i].ID
		applyLocalizedString(hopeLookup, hopeID, "name", &classes[i].HopeFeature.Name)
		applyLocalizedString(hopeLookup, hopeID, "description", &classes[i].HopeFeature.Description)
	}
	return nil
}

func localizeSubclasses(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, subclasses []storage.DaggerheartSubclass) error {
	localeValue, ok := localeString(locale)
	if !ok || len(subclasses) == 0 {
		return nil
	}
	ids := make([]string, 0, len(subclasses))
	featureIDs := make([]string, 0)
	for _, subclass := range subclasses {
		ids = append(ids, subclass.ID)
		featureIDs = append(featureIDs, collectFeatureIDs(subclass.FoundationFeatures)...)
		featureIDs = append(featureIDs, collectFeatureIDs(subclass.SpecializationFeatures)...)
		featureIDs = append(featureIDs, collectFeatureIDs(subclass.MasteryFeatures)...)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeSubclass, ids, localeValue)
	if err != nil {
		return err
	}
	featureLookup, err := fetchContentStrings(ctx, store, contentTypeFeature, featureIDs, localeValue)
	if err != nil {
		return err
	}
	for i := range subclasses {
		applyLocalizedString(lookup, subclasses[i].ID, "name", &subclasses[i].Name)
		applyLocalizedFeatures(subclasses[i].FoundationFeatures, featureLookup)
		applyLocalizedFeatures(subclasses[i].SpecializationFeatures, featureLookup)
		applyLocalizedFeatures(subclasses[i].MasteryFeatures, featureLookup)
	}
	return nil
}

func localizeHeritages(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, heritages []storage.DaggerheartHeritage) error {
	localeValue, ok := localeString(locale)
	if !ok || len(heritages) == 0 {
		return nil
	}
	ids := make([]string, 0, len(heritages))
	featureIDs := make([]string, 0)
	for _, heritage := range heritages {
		ids = append(ids, heritage.ID)
		featureIDs = append(featureIDs, collectFeatureIDs(heritage.Features)...)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeHeritage, ids, localeValue)
	if err != nil {
		return err
	}
	featureLookup, err := fetchContentStrings(ctx, store, contentTypeFeature, featureIDs, localeValue)
	if err != nil {
		return err
	}
	for i := range heritages {
		applyLocalizedString(lookup, heritages[i].ID, "name", &heritages[i].Name)
		applyLocalizedFeatures(heritages[i].Features, featureLookup)
	}
	return nil
}

func localizeExperiences(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, experiences []storage.DaggerheartExperienceEntry) error {
	localeValue, ok := localeString(locale)
	if !ok || len(experiences) == 0 {
		return nil
	}
	ids := make([]string, 0, len(experiences))
	for _, entry := range experiences {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeExperience, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range experiences {
		applyLocalizedString(lookup, experiences[i].ID, "name", &experiences[i].Name)
		applyLocalizedString(lookup, experiences[i].ID, "description", &experiences[i].Description)
	}
	return nil
}

func localizeAdversaries(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, adversaries []storage.DaggerheartAdversaryEntry) error {
	localeValue, ok := localeString(locale)
	if !ok || len(adversaries) == 0 {
		return nil
	}
	ids := make([]string, 0, len(adversaries))
	featureIDs := make([]string, 0)
	for _, entry := range adversaries {
		ids = append(ids, entry.ID)
		for _, feature := range entry.Features {
			if feature.ID != "" {
				featureIDs = append(featureIDs, feature.ID)
			}
		}
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeAdversary, ids, localeValue)
	if err != nil {
		return err
	}
	featureLookup, err := fetchContentStrings(ctx, store, contentTypeAdversaryFeature, featureIDs, localeValue)
	if err != nil {
		return err
	}
	for i := range adversaries {
		applyLocalizedString(lookup, adversaries[i].ID, "name", &adversaries[i].Name)
		applyLocalizedString(lookup, adversaries[i].ID, "description", &adversaries[i].Description)
		applyLocalizedString(lookup, adversaries[i].ID, "motives", &adversaries[i].Motives)
		applyLocalizedString(lookup, adversaries[i].ID, "attack_name", &adversaries[i].StandardAttack.Name)
		applyLocalizedString(lookup, adversaries[i].ID, "attack_range", &adversaries[i].StandardAttack.Range)
		applyLocalizedAdversaryFeatures(adversaries[i].Features, featureLookup)
	}
	return nil
}

func localizeBeastforms(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, beastforms []storage.DaggerheartBeastformEntry) error {
	localeValue, ok := localeString(locale)
	if !ok || len(beastforms) == 0 {
		return nil
	}
	ids := make([]string, 0, len(beastforms))
	featureIDs := make([]string, 0)
	for _, entry := range beastforms {
		ids = append(ids, entry.ID)
		for _, feature := range entry.Features {
			if feature.ID != "" {
				featureIDs = append(featureIDs, feature.ID)
			}
		}
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeBeastform, ids, localeValue)
	if err != nil {
		return err
	}
	featureLookup, err := fetchContentStrings(ctx, store, contentTypeBeastformFeature, featureIDs, localeValue)
	if err != nil {
		return err
	}
	for i := range beastforms {
		applyLocalizedString(lookup, beastforms[i].ID, "name", &beastforms[i].Name)
		applyLocalizedString(lookup, beastforms[i].ID, "examples", &beastforms[i].Examples)
		for idx := range beastforms[i].Advantages {
			field := "advantage." + strconv.Itoa(idx)
			applyLocalizedString(lookup, beastforms[i].ID, field, &beastforms[i].Advantages[idx])
		}
		applyLocalizedBeastformFeatures(beastforms[i].Features, featureLookup)
	}
	return nil
}

func localizeCompanionExperiences(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, experiences []storage.DaggerheartCompanionExperienceEntry) error {
	localeValue, ok := localeString(locale)
	if !ok || len(experiences) == 0 {
		return nil
	}
	ids := make([]string, 0, len(experiences))
	for _, entry := range experiences {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeCompanionExperience, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range experiences {
		applyLocalizedString(lookup, experiences[i].ID, "name", &experiences[i].Name)
		applyLocalizedString(lookup, experiences[i].ID, "description", &experiences[i].Description)
	}
	return nil
}

func localizeLootEntries(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, entries []storage.DaggerheartLootEntry) error {
	localeValue, ok := localeString(locale)
	if !ok || len(entries) == 0 {
		return nil
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeLootEntry, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range entries {
		applyLocalizedString(lookup, entries[i].ID, "name", &entries[i].Name)
		applyLocalizedString(lookup, entries[i].ID, "description", &entries[i].Description)
	}
	return nil
}

func localizeDamageTypes(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, entries []storage.DaggerheartDamageTypeEntry) error {
	localeValue, ok := localeString(locale)
	if !ok || len(entries) == 0 {
		return nil
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeDamageType, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range entries {
		applyLocalizedString(lookup, entries[i].ID, "name", &entries[i].Name)
		applyLocalizedString(lookup, entries[i].ID, "description", &entries[i].Description)
	}
	return nil
}

func localizeDomains(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, domains []storage.DaggerheartDomain) error {
	localeValue, ok := localeString(locale)
	if !ok || len(domains) == 0 {
		return nil
	}
	ids := make([]string, 0, len(domains))
	for _, entry := range domains {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeDomain, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range domains {
		applyLocalizedString(lookup, domains[i].ID, "name", &domains[i].Name)
		applyLocalizedString(lookup, domains[i].ID, "description", &domains[i].Description)
	}
	return nil
}

func localizeDomainCards(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, cards []storage.DaggerheartDomainCard) error {
	localeValue, ok := localeString(locale)
	if !ok || len(cards) == 0 {
		return nil
	}
	ids := make([]string, 0, len(cards))
	for _, entry := range cards {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeDomainCard, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range cards {
		applyLocalizedString(lookup, cards[i].ID, "name", &cards[i].Name)
		applyLocalizedString(lookup, cards[i].ID, "usage_limit", &cards[i].UsageLimit)
		applyLocalizedString(lookup, cards[i].ID, "feature_text", &cards[i].FeatureText)
	}
	return nil
}

func localizeWeapons(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, weapons []storage.DaggerheartWeapon) error {
	localeValue, ok := localeString(locale)
	if !ok || len(weapons) == 0 {
		return nil
	}
	ids := make([]string, 0, len(weapons))
	for _, entry := range weapons {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeWeapon, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range weapons {
		applyLocalizedString(lookup, weapons[i].ID, "name", &weapons[i].Name)
		applyLocalizedString(lookup, weapons[i].ID, "feature", &weapons[i].Feature)
	}
	return nil
}

func localizeArmor(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, armor []storage.DaggerheartArmor) error {
	localeValue, ok := localeString(locale)
	if !ok || len(armor) == 0 {
		return nil
	}
	ids := make([]string, 0, len(armor))
	for _, entry := range armor {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeArmor, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range armor {
		applyLocalizedString(lookup, armor[i].ID, "name", &armor[i].Name)
		applyLocalizedString(lookup, armor[i].ID, "feature", &armor[i].Feature)
	}
	return nil
}

func localizeItems(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, items []storage.DaggerheartItem) error {
	localeValue, ok := localeString(locale)
	if !ok || len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, entry := range items {
		ids = append(ids, entry.ID)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeItem, ids, localeValue)
	if err != nil {
		return err
	}
	for i := range items {
		applyLocalizedString(lookup, items[i].ID, "name", &items[i].Name)
		applyLocalizedString(lookup, items[i].ID, "description", &items[i].Description)
		applyLocalizedString(lookup, items[i].ID, "effect_text", &items[i].EffectText)
	}
	return nil
}

func localizeEnvironments(ctx context.Context, store storage.DaggerheartContentStore, locale commonv1.Locale, envs []storage.DaggerheartEnvironment) error {
	localeValue, ok := localeString(locale)
	if !ok || len(envs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(envs))
	featureIDs := make([]string, 0)
	for _, entry := range envs {
		ids = append(ids, entry.ID)
		featureIDs = append(featureIDs, collectFeatureIDs(entry.Features)...)
	}
	lookup, err := fetchContentStrings(ctx, store, contentTypeEnvironment, ids, localeValue)
	if err != nil {
		return err
	}
	featureLookup, err := fetchContentStrings(ctx, store, contentTypeFeature, featureIDs, localeValue)
	if err != nil {
		return err
	}
	for i := range envs {
		applyLocalizedString(lookup, envs[i].ID, "name", &envs[i].Name)
		for idx := range envs[i].Impulses {
			field := "impulse." + strconv.Itoa(idx)
			applyLocalizedString(lookup, envs[i].ID, field, &envs[i].Impulses[idx])
		}
		for idx := range envs[i].Prompts {
			field := "prompt." + strconv.Itoa(idx)
			applyLocalizedString(lookup, envs[i].ID, field, &envs[i].Prompts[idx])
		}
		applyLocalizedFeatures(envs[i].Features, featureLookup)
	}
	return nil
}
