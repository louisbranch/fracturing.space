package snapshot

import (
	"errors"
	"testing"
)

func TestApplyGmFearGain(t *testing.T) {
	fear := GmFear{CampaignID: "camp-1", Value: 2}
	updated, before, after, err := ApplyGmFearGain(fear, 3)
	if err != nil {
		t.Fatalf("apply gm fear gain: %v", err)
	}
	if before != 2 {
		t.Fatalf("expected before 2, got %d", before)
	}
	if after != 5 {
		t.Fatalf("expected after 5, got %d", after)
	}
	if updated.Value != 5 {
		t.Fatalf("expected updated gm fear 5, got %d", updated.Value)
	}
}

func TestApplyGmFearGainInvalidAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount int
	}{
		{name: "zero", amount: 0},
		{name: "negative", amount: -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ApplyGmFearGain(GmFear{Value: 1}, tt.amount)
			if !errors.Is(err, ErrInvalidGMFearAmount) {
				t.Fatalf("expected ErrInvalidGMFearAmount, got %v", err)
			}
		})
	}
}

func TestApplyGmFearGainExceedsCap(t *testing.T) {
	_, _, _, err := ApplyGmFearGain(GmFear{Value: 10}, 3)
	if !errors.Is(err, ErrGMFearExceedsCap) {
		t.Fatalf("expected ErrGMFearExceedsCap, got %v", err)
	}
}

func TestApplyGmFearSpend(t *testing.T) {
	fear := GmFear{CampaignID: "camp-1", Value: 5}
	updated, before, after, err := ApplyGmFearSpend(fear, 3)
	if err != nil {
		t.Fatalf("apply gm fear spend: %v", err)
	}
	if before != 5 {
		t.Fatalf("expected before 5, got %d", before)
	}
	if after != 2 {
		t.Fatalf("expected after 2, got %d", after)
	}
	if updated.Value != 2 {
		t.Fatalf("expected updated gm fear 2, got %d", updated.Value)
	}
}

func TestApplyGmFearSpendInsufficient(t *testing.T) {
	_, _, _, err := ApplyGmFearSpend(GmFear{Value: 1}, 3)
	if !errors.Is(err, ErrInsufficientGMFear) {
		t.Fatalf("expected ErrInsufficientGMFear, got %v", err)
	}
}
