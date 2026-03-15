package discovery

import (
	"testing"

	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
)

func TestProtoContract_DiscoveryServiceSymbolsExist(t *testing.T) {
	var _ discoveryv1.DiscoveryServiceServer
	if discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_UNSPECIFIED != 0 {
		t.Fatal("unexpected difficulty enum baseline")
	}
}

func TestProtoContract_EnumValues(t *testing.T) {
	if discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_UNSPECIFIED != 0 {
		t.Fatal("gm mode unspecified should be 0")
	}
	if discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI != 2 {
		t.Fatal("gm mode AI should be 2")
	}
	if discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_UNSPECIFIED != 0 {
		t.Fatal("intent unspecified should be 0")
	}
	if discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER != 2 {
		t.Fatal("intent starter should be 2")
	}
}

func TestProtoContract_DiscoveryEntryFields(t *testing.T) {
	entry := &discoveryv1.DiscoveryEntry{
		EntryId:                 "entry-1",
		Kind:                    discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceId:                "campaign-1",
		GmMode:                  discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI,
		Intent:                  discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER,
		Level:                   1,
		CharacterCount:          1,
		Storyline:               "test storyline",
		Tags:                    []string{"solo", "beginner"},
		PreviewHook:             "A dark bell tolls.",
		PreviewPlaystyleLabel:   "Guardian defender",
		PreviewCharacterName:    "Mira Vale",
		PreviewCharacterSummary: "A steadfast guardian.",
	}
	if entry.GetEntryId() == "" || entry.GetSourceId() == "" {
		t.Fatal("entry id/source id round-trip failed")
	}
	if entry.GetGmMode() != discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI {
		t.Fatal("gm_mode round-trip failed")
	}
	if entry.GetIntent() != discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER {
		t.Fatal("intent round-trip failed")
	}
	if len(entry.GetTags()) != 2 {
		t.Fatal("tags round-trip failed")
	}
	if entry.GetPreviewCharacterName() == "" || entry.GetPreviewHook() == "" {
		t.Fatal("preview fields round-trip failed")
	}
}
