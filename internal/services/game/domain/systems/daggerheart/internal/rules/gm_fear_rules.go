package rules

import "errors"

const (
	GMFearMin     = 0
	GMFearMax     = 12
	GMFearDefault = 0
)

// ApplyGMFearSpend validates and applies a GM fear spend.
func ApplyGMFearSpend(current, amount int) (int, int, error) {
	if amount <= 0 {
		return 0, 0, errors.New("gm fear amount must be greater than zero")
	}
	if current < amount {
		return 0, 0, errors.New("gm fear is insufficient")
	}
	before := current
	after := before - amount
	return before, after, nil
}

// ApplyGMFearGain validates and applies a GM fear gain.
func ApplyGMFearGain(current, amount int) (int, int, error) {
	if amount <= 0 {
		return 0, 0, errors.New("gm fear amount must be greater than zero")
	}
	before := current
	after := before + amount
	if after > GMFearMax {
		return 0, 0, errors.New("gm fear exceeds cap")
	}
	return before, after, nil
}
