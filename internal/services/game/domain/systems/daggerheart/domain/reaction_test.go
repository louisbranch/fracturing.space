package domain

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
)

func TestRollReaction_PropagatesRollActionErrors(t *testing.T) {
	originalRollDice := rollDice
	rollDice = func(_ dice.Request) (dice.Result, error) {
		return dice.Result{}, errBoomReactionRoll
	}
	t.Cleanup(func() {
		rollDice = originalRollDice
	})

	_, err := RollReaction(ReactionRequest{Seed: 42})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "roll dice") {
		t.Fatalf("error = %v, want wrapped roll dice message", err)
	}
}

var errBoomReactionRoll = &reactionTestError{message: "boom"}

type reactionTestError struct {
	message string
}

func (e *reactionTestError) Error() string {
	return e.message
}
