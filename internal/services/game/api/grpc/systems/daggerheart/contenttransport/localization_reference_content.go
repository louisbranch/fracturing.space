package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func localizeClasses(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, classes []contentstore.DaggerheartClass) error {
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

func localizeSubclasses(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, subclasses []contentstore.DaggerheartSubclass) error {
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

func localizeHeritages(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, heritages []contentstore.DaggerheartHeritage) error {
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

func localizeExperiences(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, experiences []contentstore.DaggerheartExperienceEntry) error {
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
