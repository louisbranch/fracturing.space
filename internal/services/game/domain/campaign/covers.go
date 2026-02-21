package campaign

import (
	"hash/fnv"
	"strings"
)

var campaignCoverAssetCatalog = []string{
	"abandoned_castle_courtyard",
	"ancient_forest_shrine",
	"arcane_observatory",
	"arcane_ruins",
	"broken_tower",
	"cliffside_monastery",
	"coastal_cliffs",
	"crystal_cavern",
	"cursed_farmlands",
	"deserted_harbour",
	"dwarven_gate",
	"enchanted_meadow",
	"forgotten_battlefield",
	"frozen_keep",
	"hidden_waterfall_sanctuary",
	"moutain_pass",
	"roadside_tavern",
	"royal_capital_skyline",
	"sunken_temple",
	"unholy_swamp",
}

var campaignCoverAssetSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(campaignCoverAssetCatalog))
	for _, id := range campaignCoverAssetCatalog {
		set[id] = struct{}{}
	}
	return set
}()

func isCampaignCoverAssetID(raw string) bool {
	coverAssetID := strings.TrimSpace(raw)
	if coverAssetID == "" {
		return false
	}
	_, ok := campaignCoverAssetSet[coverAssetID]
	return ok
}

func defaultCampaignCoverAssetID(campaignID string) string {
	if len(campaignCoverAssetCatalog) == 0 {
		return ""
	}

	trimmedCampaignID := strings.TrimSpace(campaignID)
	if trimmedCampaignID == "" {
		return campaignCoverAssetCatalog[0]
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(trimmedCampaignID))
	index := hasher.Sum32() % uint32(len(campaignCoverAssetCatalog))
	return campaignCoverAssetCatalog[index]
}
