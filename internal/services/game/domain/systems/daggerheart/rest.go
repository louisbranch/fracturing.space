package daggerheart

import (
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
)

// RestType represents the type of rest taken.
type RestType int

const (
	RestTypeShort RestType = iota
	RestTypeLong
)

// RestState tracks consecutive short rests.
type RestState struct {
	ConsecutiveShortRests int
}

var (
	// ErrInvalidRestSequence indicates too many short rests in a row.
	ErrInvalidRestSequence = apperrors.New(apperrors.CodeDaggerheartInvalidRestSequence, "too many short rests in a row")
)

// RestOutcome captures rest consequences and state updates.
type RestOutcome struct {
	Applied          bool
	EffectiveType    RestType
	GMFearGain       int
	AdvanceCountdown bool
	RefreshRest      bool
	RefreshLongRest  bool
	State            RestState
}

// ResolveRestOutcome applies rest rules and consequences.
func ResolveRestOutcome(state RestState, restType RestType, interrupted bool, seed int64, partySize int) (RestOutcome, error) {
	if restType == RestTypeShort && state.ConsecutiveShortRests >= 3 {
		return RestOutcome{}, ErrInvalidRestSequence
	}

	if restType == RestTypeShort && interrupted {
		return RestOutcome{
			Applied:       false,
			EffectiveType: restType,
			State:         state,
		}, nil
	}

	effective := restType
	if restType == RestTypeLong && interrupted {
		effective = RestTypeShort
	}

	gmFearGain, advanceCountdown, err := restConsequences(effective, seed, partySize)
	if err != nil {
		return RestOutcome{}, err
	}

	updated := state
	if effective == RestTypeShort {
		updated.ConsecutiveShortRests++
	} else {
		updated.ConsecutiveShortRests = 0
	}

	return RestOutcome{
		Applied:          true,
		EffectiveType:    effective,
		GMFearGain:       gmFearGain,
		AdvanceCountdown: advanceCountdown,
		RefreshRest:      true,
		RefreshLongRest:  effective == RestTypeLong,
		State:            updated,
	}, nil
}

func restConsequences(restType RestType, seed int64, partySize int) (gmFearGain int, advanceCountdown bool, err error) {
	roll, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 4, Count: 1}},
		Seed: seed,
	})
	if err != nil {
		return 0, false, err
	}

	rollValue := roll.Total
	if restType == RestTypeShort {
		return rollValue, false, nil
	}
	if partySize < 0 {
		partySize = 0
	}
	return rollValue + partySize, true, nil
}
