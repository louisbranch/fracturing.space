package catalog

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
)

func TestBuiltinEntries_ReturnsThreeEntries(t *testing.T) {
	entries, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
}

func TestBuiltinEntries_StarterCampaignShape(t *testing.T) {
	entries, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	for i, e := range entries {
		if e.EntryID == "" || e.SourceID == "" {
			t.Fatalf("entries[%d] missing ids", i)
		}
		if e.Kind != discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER {
			t.Fatalf("entries[%d].kind = %v, want CAMPAIGN_STARTER", i, e.Kind)
		}
		if e.DifficultyTier != discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER {
			t.Fatalf("entries[%d].difficulty = %v, want BEGINNER", i, e.DifficultyTier)
		}
		if e.GmMode != discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI {
			t.Fatalf("entries[%d].gm_mode = %v, want AI", i, e.GmMode)
		}
		if e.Intent != discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER {
			t.Fatalf("entries[%d].intent = %v, want STARTER", i, e.Intent)
		}
		if e.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
			t.Fatalf("entries[%d].system = %v, want DAGGERHEART", i, e.System)
		}
		if e.Storyline == "" {
			t.Fatalf("entries[%d].storyline is empty", i)
		}
	}
}

func TestBuiltinEntries_ReturnsDeepCopies(t *testing.T) {
	a, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	a[0].Title = "mutated"
	a[0].Tags[0] = "mutated"

	b, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	if b[0].Title == "mutated" {
		t.Fatal("title mutation leaked")
	}
	if b[0].Tags[0] == "mutated" {
		t.Fatal("tags mutation leaked")
	}
}
