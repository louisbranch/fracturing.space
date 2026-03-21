package contenttransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestLocalizeAdversaries(t *testing.T) {
	store := newFakeContentStore()
	store.contentStrings = []contentstore.DaggerheartContentString{
		{ContentID: "adv-1", ContentType: contentTypeAdversary, Field: "name", Locale: "en-US", Text: "Goblin raider"},
		{ContentID: "adv-1", ContentType: contentTypeAdversary, Field: "description", Locale: "en-US", Text: "Fast ambusher"},
		{ContentID: "adv-1", ContentType: contentTypeAdversary, Field: "motives", Locale: "en-US", Text: "Take everything"},
		{ContentID: "adv-1", ContentType: contentTypeAdversary, Field: "attack_name", Locale: "en-US", Text: "Jagged strike"},
		{ContentID: "adv-1", ContentType: contentTypeAdversary, Field: "attack_range", Locale: "en-US", Text: "Close"},
		{ContentID: "adv-feature-1", ContentType: contentTypeAdversaryFeature, Field: "name", Locale: "en-US", Text: "Skitter"},
		{ContentID: "adv-feature-1", ContentType: contentTypeAdversaryFeature, Field: "description", Locale: "en-US", Text: "Moves before retaliation"},
	}

	adversaries := []contentstore.DaggerheartAdversaryEntry{{
		ID:          "adv-1",
		Name:        "Goblin",
		Description: "Small raider",
		Motives:     "Raid",
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:  "Strike",
			Range: "Melee",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{{
			ID:          "adv-feature-1",
			Name:        "Cunning",
			Description: "Strikes fast",
		}},
	}}

	if err := localizeAdversaries(context.Background(), store, commonv1.Locale_LOCALE_EN_US, adversaries); err != nil {
		t.Fatalf("localizeAdversaries error: %v", err)
	}

	got := adversaries[0]
	if got.Name != "Goblin raider" || got.Description != "Fast ambusher" || got.Motives != "Take everything" {
		t.Fatalf("adversary localization mismatch: %+v", got)
	}
	if got.StandardAttack.Name != "Jagged strike" || got.StandardAttack.Range != "Close" {
		t.Fatalf("attack localization mismatch: %+v", got.StandardAttack)
	}
	if got.Features[0].Name != "Skitter" || got.Features[0].Description != "Moves before retaliation" {
		t.Fatalf("feature localization mismatch: %+v", got.Features[0])
	}
}
