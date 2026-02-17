package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
)

func TestRollActionRollDiceError(t *testing.T) {
	originalRollDice := rollDice
	expected := errors.New("forced dice failure")
	rollDice = func(_ dice.Request) (dice.Result, error) {
		return dice.Result{}, expected
	}
	t.Cleanup(func() {
		rollDice = originalRollDice
	})

	_, err := RollAction(ActionRequest{Seed: 12345})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "roll dice") {
		t.Fatalf("error = %v, want wrapped roll dice error", err)
	}
	if !strings.Contains(err.Error(), expected.Error()) {
		t.Fatalf("error = %v, want %q", err, expected.Error())
	}
}
