package command

import "testing"

func TestActiveSessionPolicyHelpers(t *testing.T) {
	t.Run("allowed during active session", func(t *testing.T) {
		got := AllowedDuringActiveSession()

		if got.Classification != ActiveSessionClassificationAllowed {
			t.Fatalf("Classification = %q, want %q", got.Classification, ActiveSessionClassificationAllowed)
		}
		if got.AllowInGameSystemActor {
			t.Fatal("AllowInGameSystemActor = true, want false")
		}
	})

	t.Run("blocked during active session", func(t *testing.T) {
		got := BlockedDuringActiveSession()

		if got.Classification != ActiveSessionClassificationBlocked {
			t.Fatalf("Classification = %q, want %q", got.Classification, ActiveSessionClassificationBlocked)
		}
		if got.AllowInGameSystemActor {
			t.Fatal("AllowInGameSystemActor = true, want false")
		}
	})

	t.Run("blocked except in-game system actor", func(t *testing.T) {
		got := BlockedDuringActiveSessionExceptInGameSystemActor()

		if got.Classification != ActiveSessionClassificationBlocked {
			t.Fatalf("Classification = %q, want %q", got.Classification, ActiveSessionClassificationBlocked)
		}
		if !got.AllowInGameSystemActor {
			t.Fatal("AllowInGameSystemActor = false, want true")
		}
	})
}
