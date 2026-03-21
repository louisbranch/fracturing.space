package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
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

func fetchContentStrings(ctx context.Context, store contentstore.DaggerheartContentReadStore, contentType string, ids []string, locale string) (contentStringLookup, error) {
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

func collectFeatureIDs(features []contentstore.DaggerheartFeature) []string {
	ids := make([]string, 0, len(features))
	for _, feature := range features {
		if feature.ID != "" {
			ids = append(ids, feature.ID)
		}
	}
	return ids
}

func applyLocalizedFeatures(features []contentstore.DaggerheartFeature, lookup contentStringLookup) {
	for i := range features {
		applyLocalizedString(lookup, features[i].ID, "name", &features[i].Name)
		applyLocalizedString(lookup, features[i].ID, "description", &features[i].Description)
	}
}

func applyLocalizedAdversaryFeatures(features []contentstore.DaggerheartAdversaryFeature, lookup contentStringLookup) {
	for i := range features {
		applyLocalizedString(lookup, features[i].ID, "name", &features[i].Name)
		applyLocalizedString(lookup, features[i].ID, "description", &features[i].Description)
	}
}

func applyLocalizedBeastformFeatures(features []contentstore.DaggerheartBeastformFeature, lookup contentStringLookup) {
	for i := range features {
		applyLocalizedString(lookup, features[i].ID, "name", &features[i].Name)
		applyLocalizedString(lookup, features[i].ID, "description", &features[i].Description)
	}
}
