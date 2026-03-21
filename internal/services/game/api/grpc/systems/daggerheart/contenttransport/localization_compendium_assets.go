package contenttransport

import (
	"context"
	"strconv"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func localizeLootEntries(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, entries []contentstore.DaggerheartLootEntry) error {
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

func localizeWeapons(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, weapons []contentstore.DaggerheartWeapon) error {
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

func localizeArmor(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, armor []contentstore.DaggerheartArmor) error {
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

func localizeItems(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, items []contentstore.DaggerheartItem) error {
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

func localizeEnvironments(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, envs []contentstore.DaggerheartEnvironment) error {
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
