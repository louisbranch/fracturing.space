package daggerheart

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
)

// Life state identifiers for Daggerheart characters.
const (
	LifeStateAlive        = "alive"
	LifeStateUnconscious  = "unconscious"
	LifeStateBlazeOfGlory = "blaze_of_glory"
	LifeStateDead         = "dead"
)

// Death move identifiers for Daggerheart.
const (
	DeathMoveBlazeOfGlory = "blaze_of_glory"
	DeathMoveAvoidDeath   = "avoid_death"
	DeathMoveRiskItAll    = "risk_it_all"
)

var deathMoveOrder = map[string]int{
	DeathMoveBlazeOfGlory: 1,
	DeathMoveAvoidDeath:   2,
	DeathMoveRiskItAll:    3,
}

var lifeStateOrder = map[string]int{
	LifeStateAlive:        1,
	LifeStateUnconscious:  2,
	LifeStateBlazeOfGlory: 3,
	LifeStateDead:         4,
}

// NormalizeDeathMove validates and normalizes a death move value.
func NormalizeDeathMove(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("death move must not be empty")
	}
	lowered := strings.ToLower(trimmed)
	if _, ok := deathMoveOrder[lowered]; !ok {
		return "", fmt.Errorf("death move %q is not supported", trimmed)
	}
	return lowered, nil
}

// NormalizeLifeState validates and normalizes a life state value.
func NormalizeLifeState(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("life state must not be empty")
	}
	lowered := strings.ToLower(trimmed)
	if _, ok := lifeStateOrder[lowered]; !ok {
		return "", fmt.Errorf("life state %q is not supported", trimmed)
	}
	return lowered, nil
}

// DeathMoveInput captures the inputs for resolving a death move.
type DeathMoveInput struct {
	Move             string
	Level            int
	HP               int
	HPMax            int
	Hope             int
	HopeMax          int
	Stress           int
	StressMax        int
	RiskItAllHPClear *int
	RiskItAllStClear *int
	Seed             int64
}

// DeathMoveOutcome captures the resolved effects of a death move.
type DeathMoveOutcome struct {
	Move          string
	LifeState     string
	HPBefore      int
	HPAfter       int
	HopeBefore    int
	HopeAfter     int
	HopeMaxBefore int
	HopeMaxAfter  int
	StressBefore  int
	StressAfter   int
	HopeDie       *int
	FearDie       *int
	ScarGained    bool
	HPCleared     int
	StressCleared int
}

// ResolveDeathMove applies the SRD death move rules.
func ResolveDeathMove(input DeathMoveInput) (DeathMoveOutcome, error) {
	move, err := NormalizeDeathMove(input.Move)
	if err != nil {
		return DeathMoveOutcome{}, err
	}
	if input.Level <= 0 {
		return DeathMoveOutcome{}, fmt.Errorf("level must be positive")
	}
	if input.HPMax < 0 {
		return DeathMoveOutcome{}, fmt.Errorf("hp_max must be non-negative")
	}
	if input.StressMax < 0 {
		return DeathMoveOutcome{}, fmt.Errorf("stress_max must be non-negative")
	}
	if input.HopeMax < 0 || input.HopeMax > HopeMax {
		return DeathMoveOutcome{}, fmt.Errorf("hope_max must be in range 0..%d", HopeMax)
	}

	outcome := DeathMoveOutcome{
		Move:          move,
		HPBefore:      input.HP,
		HPAfter:       input.HP,
		HopeBefore:    input.Hope,
		HopeAfter:     input.Hope,
		HopeMaxBefore: input.HopeMax,
		HopeMaxAfter:  input.HopeMax,
		StressBefore:  input.Stress,
		StressAfter:   input.Stress,
	}

	switch move {
	case DeathMoveBlazeOfGlory:
		outcome.LifeState = LifeStateBlazeOfGlory
		return outcome, nil
	case DeathMoveAvoidDeath:
		roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 1}}, Seed: input.Seed})
		if err != nil {
			return DeathMoveOutcome{}, err
		}
		hopeDie := roll.Rolls[0].Results[0]
		outcome.HopeDie = &hopeDie
		outcome.LifeState = LifeStateUnconscious
		if hopeDie <= input.Level {
			outcome.ScarGained = true
			if outcome.HopeMaxAfter > 0 {
				outcome.HopeMaxAfter--
			}
		}
		if outcome.HopeAfter > outcome.HopeMaxAfter {
			outcome.HopeAfter = outcome.HopeMaxAfter
		}
		return outcome, nil
	case DeathMoveRiskItAll:
		roll, err := dice.RollDice(dice.Request{Dice: []dice.Spec{{Sides: 12, Count: 2}}, Seed: input.Seed})
		if err != nil {
			return DeathMoveOutcome{}, err
		}
		hopeDie := roll.Rolls[0].Results[0]
		fearDie := roll.Rolls[0].Results[1]
		outcome.HopeDie = &hopeDie
		outcome.FearDie = &fearDie
		switch {
		case hopeDie == fearDie:
			outcome.LifeState = LifeStateAlive
			outcome.HPCleared = input.HPMax - input.HP
			outcome.StressCleared = input.Stress
			outcome.HPAfter = input.HPMax
			outcome.StressAfter = 0
		case hopeDie > fearDie:
			outcome.LifeState = LifeStateAlive
			hpClear := 0
			stressClear := 0
			if input.RiskItAllHPClear != nil {
				hpClear = *input.RiskItAllHPClear
			}
			if input.RiskItAllStClear != nil {
				stressClear = *input.RiskItAllStClear
			}
			if input.RiskItAllHPClear == nil && input.RiskItAllStClear == nil {
				hpClear = hopeDie
			}
			if hpClear < 0 || stressClear < 0 {
				return DeathMoveOutcome{}, fmt.Errorf("risk_it_all clear values must be non-negative")
			}
			if hpClear+stressClear > hopeDie {
				return DeathMoveOutcome{}, fmt.Errorf("risk_it_all clear values exceed hope die")
			}
			outcome.HPCleared = hpClear
			outcome.StressCleared = stressClear
			outcome.HPAfter = min(input.HP+hpClear, input.HPMax)
			outcome.StressAfter = max(input.Stress-stressClear, 0)
		default:
			outcome.LifeState = LifeStateDead
		}
		return outcome, nil
	default:
		return DeathMoveOutcome{}, fmt.Errorf("death move %q is not supported", move)
	}
}
