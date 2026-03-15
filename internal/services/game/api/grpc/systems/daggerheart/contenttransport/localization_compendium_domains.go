package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func localizeDamageTypes(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, entries []contentstore.DaggerheartDamageTypeEntry) error {
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

func localizeDomains(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, domains []contentstore.DaggerheartDomain) error {
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

func localizeDomainCards(ctx context.Context, store contentstore.DaggerheartContentReadStore, locale commonv1.Locale, cards []contentstore.DaggerheartDomainCard) error {
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
