package contenttransport

import (
	"context"
	"strconv"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func localizeAdversaries(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, adversaries []contentstore.DaggerheartAdversaryEntry) error {
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

func localizeBeastforms(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, beastforms []contentstore.DaggerheartBeastformEntry) error {
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

func localizeCompanionExperiences(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, experiences []contentstore.DaggerheartCompanionExperienceEntry) error {
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
