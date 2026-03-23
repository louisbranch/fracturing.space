package rules

import (
	"fmt"
	"strings"
)

const (
	CountdownToneNeutral     = "neutral"
	CountdownToneProgress    = "progress"
	CountdownToneConsequence = "consequence"
)

const (
	CountdownAdvancementPolicyManual         = "manual"
	CountdownAdvancementPolicyActionStandard = "action_standard"
	CountdownAdvancementPolicyActionDynamic  = "action_dynamic"
	CountdownAdvancementPolicyLongRest       = "long_rest"
)

const (
	CountdownLoopBehaviorNone               = "none"
	CountdownLoopBehaviorReset              = "reset"
	CountdownLoopBehaviorResetIncreaseStart = "reset_increase_start"
	CountdownLoopBehaviorResetDecreaseStart = "reset_decrease_start"
)

const (
	CountdownStatusActive         = "active"
	CountdownStatusTriggerPending = "trigger_pending"
)

type CountdownStartingRoll struct {
	Min   int
	Max   int
	Value int
}

type Countdown struct {
	CampaignID        string
	ID                string
	Name              string
	Tone              string
	AdvancementPolicy string
	StartingValue     int
	RemainingValue    int
	LoopBehavior      string
	Status            string
	LinkedCountdownID string
	StartingRoll      *CountdownStartingRoll
}

type CountdownAdvance struct {
	BeforeRemaining int
	AfterRemaining  int
	AdvancedBy      int
	Triggered       bool
	StatusBefore    string
	StatusAfter     string
	Countdown       Countdown
}

type CountdownTriggerResolution struct {
	StartingValueBefore  int
	StartingValueAfter   int
	RemainingValueBefore int
	RemainingValueAfter  int
	StatusBefore         string
	StatusAfter          string
	Countdown            Countdown
}

func NormalizeCountdownTone(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "", fmt.Errorf("countdown tone is required")
	}
	switch trimmed {
	case CountdownToneNeutral, CountdownToneProgress, CountdownToneConsequence:
		return trimmed, nil
	default:
		return "", fmt.Errorf("countdown tone %q is invalid", value)
	}
}

func NormalizeCountdownAdvancementPolicy(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "", fmt.Errorf("countdown advancement policy is required")
	}
	switch trimmed {
	case CountdownAdvancementPolicyManual, CountdownAdvancementPolicyActionStandard, CountdownAdvancementPolicyActionDynamic, CountdownAdvancementPolicyLongRest:
		return trimmed, nil
	default:
		return "", fmt.Errorf("countdown advancement policy %q is invalid", value)
	}
}

func NormalizeCountdownLoopBehavior(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "", fmt.Errorf("countdown loop behavior is required")
	}
	switch trimmed {
	case CountdownLoopBehaviorNone, CountdownLoopBehaviorReset, CountdownLoopBehaviorResetIncreaseStart, CountdownLoopBehaviorResetDecreaseStart:
		return trimmed, nil
	default:
		return "", fmt.Errorf("countdown loop behavior %q is invalid", value)
	}
}

func NormalizeCountdownStatus(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return CountdownStatusActive, nil
	}
	switch trimmed {
	case CountdownStatusActive, CountdownStatusTriggerPending:
		return trimmed, nil
	default:
		return "", fmt.Errorf("countdown status %q is invalid", value)
	}
}

func ValidateCountdown(countdown Countdown) error {
	if _, err := NormalizeCountdownTone(countdown.Tone); err != nil {
		return err
	}
	if _, err := NormalizeCountdownAdvancementPolicy(countdown.AdvancementPolicy); err != nil {
		return err
	}
	if _, err := NormalizeCountdownLoopBehavior(countdown.LoopBehavior); err != nil {
		return err
	}
	status, err := NormalizeCountdownStatus(countdown.Status)
	if err != nil {
		return err
	}
	if countdown.StartingValue <= 0 {
		return fmt.Errorf("countdown starting value must be positive")
	}
	if countdown.RemainingValue < 0 || countdown.RemainingValue > countdown.StartingValue {
		return fmt.Errorf("countdown remaining value must be in range 0..%d", countdown.StartingValue)
	}
	if countdown.RemainingValue == 0 && status != CountdownStatusTriggerPending && countdown.LoopBehavior != CountdownLoopBehaviorNone {
		return fmt.Errorf("countdown at zero must be trigger_pending until resolved")
	}
	if countdown.StartingRoll != nil {
		if countdown.StartingRoll.Min <= 0 || countdown.StartingRoll.Max < countdown.StartingRoll.Min {
			return fmt.Errorf("countdown starting roll range is invalid")
		}
		if countdown.StartingRoll.Value < countdown.StartingRoll.Min || countdown.StartingRoll.Value > countdown.StartingRoll.Max {
			return fmt.Errorf("countdown starting roll value must be in range %d..%d", countdown.StartingRoll.Min, countdown.StartingRoll.Max)
		}
	}
	return nil
}

func ApplyCountdownAdvance(countdown Countdown, amount int) (CountdownAdvance, error) {
	if err := ValidateCountdown(countdown); err != nil {
		return CountdownAdvance{}, err
	}
	if amount <= 0 {
		return CountdownAdvance{}, fmt.Errorf("countdown advance amount must be positive")
	}
	if countdown.Status == CountdownStatusTriggerPending {
		return CountdownAdvance{}, fmt.Errorf("countdown trigger must be resolved before advancing again")
	}
	before := countdown.RemainingValue
	after := before - amount
	if after < 0 {
		after = 0
	}
	advanced := before - after
	if advanced == 0 {
		return CountdownAdvance{}, fmt.Errorf("countdown advance must change remaining value")
	}
	triggered := before > 0 && after == 0
	statusAfter := CountdownStatusActive
	if triggered {
		statusAfter = CountdownStatusTriggerPending
	}

	update := CountdownAdvance{
		BeforeRemaining: before,
		AfterRemaining:  after,
		AdvancedBy:      advanced,
		Triggered:       triggered,
		StatusBefore:    countdown.Status,
		StatusAfter:     statusAfter,
		Countdown:       countdown,
	}
	update.Countdown.RemainingValue = after
	update.Countdown.Status = statusAfter
	return update, nil
}

func ResolveCountdownTrigger(countdown Countdown) (CountdownTriggerResolution, error) {
	if err := ValidateCountdown(countdown); err != nil {
		return CountdownTriggerResolution{}, err
	}
	if countdown.Status != CountdownStatusTriggerPending {
		return CountdownTriggerResolution{}, fmt.Errorf("countdown trigger is not pending")
	}

	beforeStart := countdown.StartingValue
	beforeRemaining := countdown.RemainingValue
	afterStart := beforeStart
	afterRemaining := beforeRemaining

	switch countdown.LoopBehavior {
	case CountdownLoopBehaviorNone:
		afterRemaining = 0
	case CountdownLoopBehaviorReset:
		afterRemaining = beforeStart
	case CountdownLoopBehaviorResetIncreaseStart:
		afterStart = beforeStart + 1
		afterRemaining = afterStart
	case CountdownLoopBehaviorResetDecreaseStart:
		afterStart = beforeStart - 1
		if afterStart < 1 {
			afterStart = 1
		}
		afterRemaining = afterStart
	default:
		return CountdownTriggerResolution{}, fmt.Errorf("countdown loop behavior %q is invalid", countdown.LoopBehavior)
	}

	result := CountdownTriggerResolution{
		StartingValueBefore:  beforeStart,
		StartingValueAfter:   afterStart,
		RemainingValueBefore: beforeRemaining,
		RemainingValueAfter:  afterRemaining,
		StatusBefore:         countdown.Status,
		StatusAfter:          CountdownStatusActive,
		Countdown:            countdown,
	}
	result.Countdown.StartingValue = afterStart
	result.Countdown.RemainingValue = afterRemaining
	result.Countdown.Status = CountdownStatusActive
	return result, nil
}

func DynamicCountdownAdvanceAmount(tone, outcome string) (int, error) {
	normalizedTone, err := NormalizeCountdownTone(tone)
	if err != nil {
		return 0, err
	}
	normalizedOutcome := strings.TrimSpace(strings.ToLower(outcome))
	switch normalizedOutcome {
	case "failure_with_fear":
		if normalizedTone == CountdownToneProgress {
			return 0, nil
		}
		if normalizedTone == CountdownToneConsequence {
			return 3, nil
		}
	case "failure_with_hope":
		if normalizedTone == CountdownToneProgress {
			return 0, nil
		}
		if normalizedTone == CountdownToneConsequence {
			return 2, nil
		}
	case "success_with_fear":
		if normalizedTone == CountdownToneProgress {
			return 1, nil
		}
		if normalizedTone == CountdownToneConsequence {
			return 1, nil
		}
	case "success_with_hope":
		if normalizedTone == CountdownToneProgress {
			return 2, nil
		}
		if normalizedTone == CountdownToneConsequence {
			return 0, nil
		}
	case "critical_success":
		if normalizedTone == CountdownToneProgress {
			return 3, nil
		}
		if normalizedTone == CountdownToneConsequence {
			return 0, nil
		}
	}
	return 0, fmt.Errorf("dynamic countdown advance is undefined for tone %q and outcome %q", tone, outcome)
}
