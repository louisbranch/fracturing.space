package daggerhearttools

import "testing"

func TestNormalizeCountdownCreateInput(t *testing.T) {
	t.Run("drops invalid randomized start when fixed start is present", func(t *testing.T) {
		input := countdownCreateInput{
			FixedStartingValue: 4,
			RandomizedStart:    &rangeInput{Min: 0, Max: 2},
		}

		got := normalizeCountdownCreateInput(input)
		if got.RandomizedStart != nil {
			t.Fatalf("RandomizedStart = %#v, want nil", got.RandomizedStart)
		}
		if got.FixedStartingValue != 4 {
			t.Fatalf("FixedStartingValue = %d, want 4", got.FixedStartingValue)
		}
	})

	t.Run("falls back to one when both starts are invalid", func(t *testing.T) {
		input := countdownCreateInput{
			RandomizedStart: &rangeInput{Min: 0, Max: 0},
		}

		got := normalizeCountdownCreateInput(input)
		if got.RandomizedStart != nil {
			t.Fatalf("RandomizedStart = %#v, want nil", got.RandomizedStart)
		}
		if got.FixedStartingValue != 1 {
			t.Fatalf("FixedStartingValue = %d, want 1", got.FixedStartingValue)
		}
	})
}

func TestGmMoveApplyRequestFromInput(t *testing.T) {
	t.Run("requires exactly one spend target", func(t *testing.T) {
		_, err := gmMoveApplyRequestFromInput("camp", "sess", "scene", gmMoveApplyInput{FearSpent: 1})
		if err == nil || err.Error() != "one gm move spend target is required" {
			t.Fatalf("error = %v, want one-target error", err)
		}
	})

	t.Run("rejects multiple spend targets", func(t *testing.T) {
		_, err := gmMoveApplyRequestFromInput("camp", "sess", "scene", gmMoveApplyInput{
			FearSpent: 1,
			DirectMove: &gmMoveDirectMoveInput{
				Kind: "ADDITIONAL_MOVE",
			},
			AdversaryFeature: &gmMoveAdversaryFeatureInput{
				AdversaryID: "adv",
				FeatureID:   "feature",
			},
		})
		if err == nil || err.Error() != "only one gm move spend target may be provided" {
			t.Fatalf("error = %v, want multi-target error", err)
		}
	})

	t.Run("builds direct move request", func(t *testing.T) {
		req, err := gmMoveApplyRequestFromInput("camp", "sess", "scene", gmMoveApplyInput{
			FearSpent: 2,
			DirectMove: &gmMoveDirectMoveInput{
				Kind:  "ADDITIONAL_MOVE",
				Shape: "REVEAL_DANGER",
			},
		})
		if err != nil {
			t.Fatalf("gmMoveApplyRequestFromInput: %v", err)
		}
		if req.GetCampaignId() != "camp" || req.GetSessionId() != "sess" || req.GetSceneId() != "scene" {
			t.Fatalf("request scope = %#v", req)
		}
		if req.GetFearSpent() != 2 {
			t.Fatalf("FearSpent = %d, want 2", req.GetFearSpent())
		}
		if req.GetDirectMove() == nil {
			t.Fatalf("DirectMove = nil, want direct move target")
		}
	})
}
