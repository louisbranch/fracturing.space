package daggerheart

import "strings"

const (
	// allowedDaggerheartPotionMinorIDs contains potion item IDs permitted by the
	// character creation workflow.
	allowedPotionMinorHealth  = "item.minor-health-potion"
	allowedPotionMinorStamina = "item.minor-stamina-potion"
)

var allowedPotionItemIDs = map[string]struct{}{
	allowedPotionMinorHealth:  {},
	allowedPotionMinorStamina: {},
}

func isAllowedPotionItemID(rawItemID string) bool {
	itemID := strings.TrimSpace(rawItemID)
	_, ok := allowedPotionItemIDs[itemID]
	return ok
}
