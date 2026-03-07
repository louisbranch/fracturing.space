package daggerheart

import "testing"

func TestApplyGMFearSpend(t *testing.T) {
	t.Run("valid spend", func(t *testing.T) {
		before, after, err := ApplyGMFearSpend(3, 2)
		if err != nil {
			t.Fatalf("ApplyGMFearSpend() error = %v", err)
		}
		if before != 3 || after != 1 {
			t.Fatalf("before/after = %d/%d, want 3/1", before, after)
		}
	})

	t.Run("amount must be positive", func(t *testing.T) {
		_, _, err := ApplyGMFearSpend(3, 0)
		if err == nil {
			t.Fatal("expected positive amount error")
		}
	})

	t.Run("insufficient fear", func(t *testing.T) {
		_, _, err := ApplyGMFearSpend(1, 2)
		if err == nil {
			t.Fatal("expected insufficient fear error")
		}
	})
}

func TestApplyGMFearGain(t *testing.T) {
	t.Run("valid gain", func(t *testing.T) {
		before, after, err := ApplyGMFearGain(3, 2)
		if err != nil {
			t.Fatalf("ApplyGMFearGain() error = %v", err)
		}
		if before != 3 || after != 5 {
			t.Fatalf("before/after = %d/%d, want 3/5", before, after)
		}
	})

	t.Run("amount must be positive", func(t *testing.T) {
		_, _, err := ApplyGMFearGain(3, 0)
		if err == nil {
			t.Fatal("expected positive amount error")
		}
	})

	t.Run("cap exceeded", func(t *testing.T) {
		_, _, err := ApplyGMFearGain(GMFearMax, 1)
		if err == nil {
			t.Fatal("expected cap exceeded error")
		}
	})
}
