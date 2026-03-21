package contenttransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestLocalizeDomainCards(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "card-1", ContentType: contentTypeDomainCard, Field: "name", Locale: "en-US", Text: "Inferno"},
		{ContentID: "card-1", ContentType: contentTypeDomainCard, Field: "usage_limit", Locale: "en-US", Text: "Twice per rest"},
		{ContentID: "card-1", ContentType: contentTypeDomainCard, Field: "feature_text", Locale: "en-US", Text: "Unleash a wall of fire"},
	}

	cards := []contentstore.DaggerheartDomainCard{{
		ID:          "card-1",
		Name:        "Fireball",
		UsageLimit:  "Once",
		FeatureText: "Deals fire damage",
	}}

	if err := localizeDomainCards(context.Background(), store, commonv1.Locale_LOCALE_EN_US, cards); err != nil {
		t.Fatalf("localizeDomainCards error: %v", err)
	}

	got := cards[0]
	if got.Name != "Inferno" || got.UsageLimit != "Twice per rest" || got.FeatureText != "Unleash a wall of fire" {
		t.Fatalf("domain card localization mismatch: %+v", got)
	}
}
