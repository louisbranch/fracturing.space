package daggerheart

import (
	"fmt"
	"strings"
)

const (
	CountdownKindProgress    = "progress"
	CountdownKindConsequence = "consequence"
)

const (
	CountdownDirectionIncrease = "increase"
	CountdownDirectionDecrease = "decrease"
)

type CountdownUpdate struct {
	Before    int
	After     int
	Delta     int
	Looped    bool
	Reason    string
	Countdown Countdown
}

type Countdown struct {
	CampaignID string
	ID         string
	Name       string
	Kind       string
	Current    int
	Max        int
	Direction  string
	Looping    bool
}

func NormalizeCountdownKind(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "", fmt.Errorf("countdown kind is required")
	}
	switch trimmed {
	case CountdownKindProgress, CountdownKindConsequence:
		return trimmed, nil
	default:
		return "", fmt.Errorf("countdown kind %q is invalid", value)
	}
}

func NormalizeCountdownDirection(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "", fmt.Errorf("countdown direction is required")
	}
	switch trimmed {
	case CountdownDirectionIncrease, CountdownDirectionDecrease:
		return trimmed, nil
	default:
		return "", fmt.Errorf("countdown direction %q is invalid", value)
	}
}

func ApplyCountdownUpdate(countdown Countdown, delta int, override *int) (CountdownUpdate, error) {
	if countdown.Max <= 0 {
		return CountdownUpdate{}, fmt.Errorf("countdown max must be positive")
	}
	if countdown.Current < 0 || countdown.Current > countdown.Max {
		return CountdownUpdate{}, fmt.Errorf("countdown current must be in range 0..%d", countdown.Max)
	}
	if override == nil && delta == 0 {
		return CountdownUpdate{}, fmt.Errorf("countdown update requires delta or current override")
	}

	before := countdown.Current
	after := before
	looped := false

	if override != nil {
		after = *override
	} else {
		after = before + delta
	}

	if after > countdown.Max {
		if countdown.Looping {
			looped = true
			after = 0
		} else {
			after = countdown.Max
		}
	}
	if after < 0 {
		if countdown.Looping {
			looped = true
			after = countdown.Max
		} else {
			after = 0
		}
	}

	update := CountdownUpdate{
		Before:    before,
		After:     after,
		Delta:     after - before,
		Looped:    looped,
		Countdown: countdown,
	}
	update.Countdown.Current = after
	return update, nil
}
