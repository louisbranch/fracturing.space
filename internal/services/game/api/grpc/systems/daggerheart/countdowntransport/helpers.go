package countdowntransport

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func countdownToneFromProto(value pb.DaggerheartCountdownTone) (string, error) {
	switch value {
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL:
		return rules.CountdownToneNeutral, nil
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS:
		return rules.CountdownToneProgress, nil
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE:
		return rules.CountdownToneConsequence, nil
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED:
		return "", fmt.Errorf("countdown tone is required")
	default:
		return "", fmt.Errorf("countdown tone %v is invalid", value)
	}
}

func countdownPolicyFromProto(value pb.DaggerheartCountdownAdvancementPolicy) (string, error) {
	switch value {
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL:
		return rules.CountdownAdvancementPolicyManual, nil
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD:
		return rules.CountdownAdvancementPolicyActionStandard, nil
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC:
		return rules.CountdownAdvancementPolicyActionDynamic, nil
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST:
		return rules.CountdownAdvancementPolicyLongRest, nil
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED:
		return "", fmt.Errorf("countdown advancement policy is required")
	default:
		return "", fmt.Errorf("countdown advancement policy %v is invalid", value)
	}
}

func countdownLoopBehaviorFromProto(value pb.DaggerheartCountdownLoopBehavior) (string, error) {
	switch value {
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE:
		return rules.CountdownLoopBehaviorNone, nil
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET:
		return rules.CountdownLoopBehaviorReset, nil
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START:
		return rules.CountdownLoopBehaviorResetIncreaseStart, nil
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START:
		return rules.CountdownLoopBehaviorResetDecreaseStart, nil
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED:
		return "", fmt.Errorf("countdown loop behavior is required")
	default:
		return "", fmt.Errorf("countdown loop behavior %v is invalid", value)
	}
}

func countdownToneToProto(value string) pb.DaggerheartCountdownTone {
	switch value {
	case rules.CountdownToneNeutral:
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL
	case rules.CountdownToneProgress:
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS
	case rules.CountdownToneConsequence:
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE
	default:
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED
	}
}

func countdownPolicyToProto(value string) pb.DaggerheartCountdownAdvancementPolicy {
	switch value {
	case rules.CountdownAdvancementPolicyManual:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL
	case rules.CountdownAdvancementPolicyActionStandard:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD
	case rules.CountdownAdvancementPolicyActionDynamic:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC
	case rules.CountdownAdvancementPolicyLongRest:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST
	default:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED
	}
}

func countdownLoopBehaviorToProto(value string) pb.DaggerheartCountdownLoopBehavior {
	switch value {
	case rules.CountdownLoopBehaviorNone:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE
	case rules.CountdownLoopBehaviorReset:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET
	case rules.CountdownLoopBehaviorResetIncreaseStart:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START
	case rules.CountdownLoopBehaviorResetDecreaseStart:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START
	default:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED
	}
}

func countdownStatusToProto(value string) pb.DaggerheartCountdownStatus {
	switch value {
	case rules.CountdownStatusActive:
		return pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE
	case rules.CountdownStatusTriggerPending:
		return pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING
	default:
		return pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_UNSPECIFIED
	}
}

func countdownStatusFromProto(value pb.DaggerheartCountdownStatus) (string, error) {
	switch value {
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE:
		return rules.CountdownStatusActive, nil
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING:
		return rules.CountdownStatusTriggerPending, nil
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_UNSPECIFIED:
		return "", fmt.Errorf("countdown status is required")
	default:
		return "", fmt.Errorf("countdown status %v is invalid", value)
	}
}

func countdownFromStorage(countdown projectionstore.DaggerheartCountdown) rules.Countdown {
	value := rules.Countdown{
		CampaignID:        countdown.CampaignID,
		ID:                countdown.CountdownID,
		Name:              countdown.Name,
		Tone:              countdown.Tone,
		AdvancementPolicy: countdown.AdvancementPolicy,
		StartingValue:     countdown.StartingValue,
		RemainingValue:    countdown.RemainingValue,
		LoopBehavior:      countdown.LoopBehavior,
		Status:            countdown.Status,
		LinkedCountdownID: countdown.LinkedCountdownID,
		Kind:              countdown.Kind,
		Current:           countdown.Current,
		Max:               countdown.Max,
		Direction:         countdown.Direction,
		Looping:           countdown.Looping,
	}
	if value.Tone == "" && countdown.Kind != "" {
		value.Tone = countdown.Kind
	}
	if value.StartingValue == 0 && countdown.Max > 0 {
		value.StartingValue = countdown.Max
	}
	if value.RemainingValue == 0 && (countdown.Current > 0 || countdown.Max > 0) {
		value.RemainingValue = countdown.Current
	}
	if value.LoopBehavior == "" {
		if countdown.Looping {
			value.LoopBehavior = rules.CountdownLoopBehaviorReset
		} else {
			value.LoopBehavior = rules.CountdownLoopBehaviorNone
		}
	}
	if value.AdvancementPolicy == "" {
		value.AdvancementPolicy = rules.CountdownAdvancementPolicyManual
	}
	if value.Status == "" {
		value.Status = rules.CountdownStatusActive
	}
	if countdown.StartingRollMin > 0 && countdown.StartingRollMax > 0 {
		value.StartingRoll = &rules.CountdownStartingRoll{
			Min:   countdown.StartingRollMin,
			Max:   countdown.StartingRollMax,
			Value: countdown.StartingRollValue,
		}
	}
	return value
}

func SceneCountdownToProto(countdown projectionstore.DaggerheartCountdown) *pb.DaggerheartSceneCountdown {
	if countdown.Tone == "" && countdown.Kind != "" {
		countdown.Tone = countdown.Kind
	}
	if countdown.StartingValue == 0 && countdown.Max > 0 {
		countdown.StartingValue = countdown.Max
	}
	if countdown.RemainingValue == 0 && (countdown.Current > 0 || countdown.Max > 0) {
		countdown.RemainingValue = countdown.Current
	}
	if countdown.LoopBehavior == "" {
		if countdown.Looping {
			countdown.LoopBehavior = rules.CountdownLoopBehaviorReset
		} else {
			countdown.LoopBehavior = rules.CountdownLoopBehaviorNone
		}
	}
	if countdown.AdvancementPolicy == "" {
		countdown.AdvancementPolicy = rules.CountdownAdvancementPolicyManual
	}
	if countdown.Status == "" {
		countdown.Status = rules.CountdownStatusActive
	}
	value := &pb.DaggerheartSceneCountdown{
		CountdownId:       countdown.CountdownID,
		CampaignId:        countdown.CampaignID,
		SessionId:         countdown.SessionID,
		SceneId:           countdown.SceneID,
		Name:              countdown.Name,
		Tone:              countdownToneToProto(countdown.Tone),
		AdvancementPolicy: countdownPolicyToProto(countdown.AdvancementPolicy),
		StartingValue:     int32(countdown.StartingValue),
		RemainingValue:    int32(countdown.RemainingValue),
		LoopBehavior:      countdownLoopBehaviorToProto(countdown.LoopBehavior),
		Status:            countdownStatusToProto(countdown.Status),
		LinkedCountdownId: countdown.LinkedCountdownID,
	}
	if countdown.StartingRollMin > 0 && countdown.StartingRollMax > 0 {
		value.StartingRoll = &pb.DaggerheartCountdownStartingRoll{
			Min:   int32(countdown.StartingRollMin),
			Max:   int32(countdown.StartingRollMax),
			Value: int32(countdown.StartingRollValue),
		}
	}
	return value
}

func CampaignCountdownToProto(countdown projectionstore.DaggerheartCountdown) *pb.DaggerheartCampaignCountdown {
	if countdown.Tone == "" && countdown.Kind != "" {
		countdown.Tone = countdown.Kind
	}
	if countdown.StartingValue == 0 && countdown.Max > 0 {
		countdown.StartingValue = countdown.Max
	}
	if countdown.RemainingValue == 0 && (countdown.Current > 0 || countdown.Max > 0) {
		countdown.RemainingValue = countdown.Current
	}
	if countdown.LoopBehavior == "" {
		if countdown.Looping {
			countdown.LoopBehavior = rules.CountdownLoopBehaviorReset
		} else {
			countdown.LoopBehavior = rules.CountdownLoopBehaviorNone
		}
	}
	if countdown.AdvancementPolicy == "" {
		countdown.AdvancementPolicy = rules.CountdownAdvancementPolicyManual
	}
	if countdown.Status == "" {
		countdown.Status = rules.CountdownStatusActive
	}
	value := &pb.DaggerheartCampaignCountdown{
		CountdownId:       countdown.CountdownID,
		Name:              countdown.Name,
		Tone:              countdownToneToProto(countdown.Tone),
		AdvancementPolicy: countdownPolicyToProto(countdown.AdvancementPolicy),
		StartingValue:     int32(countdown.StartingValue),
		RemainingValue:    int32(countdown.RemainingValue),
		LoopBehavior:      countdownLoopBehaviorToProto(countdown.LoopBehavior),
		Status:            countdownStatusToProto(countdown.Status),
		LinkedCountdownId: countdown.LinkedCountdownID,
	}
	if countdown.StartingRollMin > 0 && countdown.StartingRollMax > 0 {
		value.StartingRoll = &pb.DaggerheartCountdownStartingRoll{
			Min:   int32(countdown.StartingRollMin),
			Max:   int32(countdown.StartingRollMax),
			Value: int32(countdown.StartingRollValue),
		}
	}
	return value
}

func AdvanceSummaryToProto(countdown projectionstore.DaggerheartCountdown, summary CountdownAdvanceSummary, reason string) *pb.DaggerheartCountdownAdvance {
	return &pb.DaggerheartCountdownAdvance{
		CountdownId:       countdown.CountdownID,
		Name:              countdown.Name,
		Tone:              countdownToneToProto(countdown.Tone),
		AdvancementPolicy: countdownPolicyToProto(countdown.AdvancementPolicy),
		StartingValue:     int32(countdown.StartingValue),
		RemainingBefore:   int32(summary.BeforeRemaining),
		RemainingAfter:    int32(summary.AfterRemaining),
		AdvancedBy:        int32(summary.AdvancedBy),
		StatusBefore:      countdownStatusToProto(summary.StatusBefore),
		StatusAfter:       countdownStatusToProto(summary.StatusAfter),
		Triggered:         summary.Triggered,
		Reason:            reason,
	}
}
