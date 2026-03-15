package contenttransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func TestLocalizeEnvironments(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "env-1", ContentType: contentTypeEnvironment, Field: "name", Locale: "en-US", Text: "Mistwood"},
		{ContentID: "env-1", ContentType: contentTypeEnvironment, Field: "impulse.0", Locale: "en-US", Text: "Separate the party"},
		{ContentID: "env-1", ContentType: contentTypeEnvironment, Field: "prompt.0", Locale: "en-US", Text: "What moved in the fog?"},
		{ContentID: "feature-1", ContentType: contentTypeFeature, Field: "name", Locale: "en-US", Text: "Dense undergrowth"},
		{ContentID: "feature-1", ContentType: contentTypeFeature, Field: "description", Locale: "en-US", Text: "Slows every pursuit"},
	}

	envs := []contentstore.DaggerheartEnvironment{{
		ID:       "env-1",
		Name:     "Forest",
		Impulses: []string{"Mislead"},
		Prompts:  []string{"What hides in the fog?"},
		Features: []contentstore.DaggerheartFeature{{
			ID:          "feature-1",
			Name:        "Dense terrain",
			Description: "Slows travelers",
		}},
	}}

	if err := localizeEnvironments(context.Background(), store, commonv1.Locale_LOCALE_EN_US, envs); err != nil {
		t.Fatalf("localizeEnvironments error: %v", err)
	}

	got := envs[0]
	if got.Name != "Mistwood" {
		t.Fatalf("environment name = %q, want %q", got.Name, "Mistwood")
	}
	if got.Impulses[0] != "Separate the party" || got.Prompts[0] != "What moved in the fog?" {
		t.Fatalf("environment string localization mismatch: %+v", got)
	}
	if got.Features[0].Name != "Dense undergrowth" || got.Features[0].Description != "Slows every pursuit" {
		t.Fatalf("environment feature localization mismatch: %+v", got.Features[0])
	}
}
