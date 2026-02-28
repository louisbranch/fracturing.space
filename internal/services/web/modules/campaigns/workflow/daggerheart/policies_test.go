package daggerheart

import "testing"

func TestIsAllowedPotionItemID(t *testing.T) {
	t.Run("allowed potion IDs", func(t *testing.T) {
		t.Parallel()

		cases := []string{
			allowedPotionMinorHealth,
			allowedPotionMinorStamina,
		}
		for _, itemID := range cases {
			if !isAllowedPotionItemID(itemID) {
				t.Fatalf("isAllowedPotionItemID(%q) = false, want true", itemID)
			}
		}
	})

	t.Run("rejected potion IDs", func(t *testing.T) {
		t.Parallel()

		cases := []string{
			"item.elixir-of-power",
			"  ",
			"",
		}
		for _, itemID := range cases {
			if isAllowedPotionItemID(itemID) {
				t.Fatalf("isAllowedPotionItemID(%q) = true, want false", itemID)
			}
		}
	})
}
