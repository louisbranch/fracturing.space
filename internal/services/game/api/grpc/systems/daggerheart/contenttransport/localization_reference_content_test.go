package contenttransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func TestLocalizeClasses(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "class-1", ContentType: contentTypeClass, Field: "name", Locale: "en-US", Text: "Sentinel"},
		{ContentID: "feature-1", ContentType: contentTypeFeature, Field: "name", Locale: "en-US", Text: "Bulwark"},
		{ContentID: "feature-1", ContentType: contentTypeFeature, Field: "description", Locale: "en-US", Text: "Protects allies"},
		{ContentID: "hope_feature:class-1", ContentType: contentTypeHopeFeature, Field: "name", Locale: "en-US", Text: "Renewal"},
		{ContentID: "hope_feature:class-1", ContentType: contentTypeHopeFeature, Field: "description", Locale: "en-US", Text: "Restores courage"},
	}

	classes := []contentstore.DaggerheartClass{{
		ID:   "class-1",
		Name: "Guardian",
		Features: []contentstore.DaggerheartFeature{{
			ID:          "feature-1",
			Name:        "Shield",
			Description: "Protect allies",
		}},
		HopeFeature: contentstore.DaggerheartHopeFeature{
			Name:        "Hope",
			Description: "Restore courage",
		},
	}}

	if err := localizeClasses(context.Background(), store, commonv1.Locale_LOCALE_EN_US, classes); err != nil {
		t.Fatalf("localizeClasses error: %v", err)
	}

	if classes[0].Name != "Sentinel" {
		t.Fatalf("class name = %q, want %q", classes[0].Name, "Sentinel")
	}
	if classes[0].Features[0].Name != "Bulwark" || classes[0].Features[0].Description != "Protects allies" {
		t.Fatalf("feature localization mismatch: %+v", classes[0].Features[0])
	}
	if classes[0].HopeFeature.Name != "Renewal" || classes[0].HopeFeature.Description != "Restores courage" {
		t.Fatalf("hope feature localization mismatch: %+v", classes[0].HopeFeature)
	}
}

func TestLocalizeSubclasses(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "sub-1", ContentType: contentTypeSubclass, Field: "name", Locale: "en-US", Text: "Spellblade"},
		{ContentID: "feature-1", ContentType: contentTypeFeature, Field: "name", Locale: "en-US", Text: "Opening stance"},
		{ContentID: "feature-2", ContentType: contentTypeFeature, Field: "name", Locale: "en-US", Text: "Arc surge"},
		{ContentID: "feature-3", ContentType: contentTypeFeature, Field: "name", Locale: "en-US", Text: "Final form"},
	}

	subclasses := []contentstore.DaggerheartSubclass{{
		ID:   "sub-1",
		Name: "Bladeweaver",
		FoundationFeatures: []contentstore.DaggerheartFeature{{
			ID:   "feature-1",
			Name: "Foundation",
		}},
		SpecializationFeatures: []contentstore.DaggerheartFeature{{
			ID:   "feature-2",
			Name: "Specialization",
		}},
		MasteryFeatures: []contentstore.DaggerheartFeature{{
			ID:   "feature-3",
			Name: "Mastery",
		}},
	}}

	if err := localizeSubclasses(context.Background(), store, commonv1.Locale_LOCALE_EN_US, subclasses); err != nil {
		t.Fatalf("localizeSubclasses error: %v", err)
	}

	if subclasses[0].Name != "Spellblade" {
		t.Fatalf("subclass name = %q, want %q", subclasses[0].Name, "Spellblade")
	}
	if subclasses[0].FoundationFeatures[0].Name != "Opening stance" {
		t.Fatalf("foundation feature name = %q", subclasses[0].FoundationFeatures[0].Name)
	}
	if subclasses[0].SpecializationFeatures[0].Name != "Arc surge" {
		t.Fatalf("specialization feature name = %q", subclasses[0].SpecializationFeatures[0].Name)
	}
	if subclasses[0].MasteryFeatures[0].Name != "Final form" {
		t.Fatalf("mastery feature name = %q", subclasses[0].MasteryFeatures[0].Name)
	}
}
