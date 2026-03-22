package daggerheart

import (
	"fmt"
	"slices"
	"strings"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/countdowns"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

const (
	DowntimeMoveTendToWounds    = "tend_to_wounds"
	DowntimeMoveClearStress     = "clear_stress"
	DowntimeMoveRepairArmor     = "repair_armor"
	DowntimeMovePrepare         = "prepare"
	DowntimeMoveTendToAllWounds = "tend_to_all_wounds"
	DowntimeMoveClearAllStress  = "clear_all_stress"
	DowntimeMoveRepairAllArmor  = "repair_all_armor"
	DowntimeMoveWorkOnProject   = "work_on_project"
)

const (
	ProjectAdvanceModeAuto       = "auto"
	ProjectAdvanceModeGMSetDelta = "gm_set_delta"
)

// TierForLevel exposes the Daggerheart level-to-tier mapping to transport and
// recovery orchestration without leaking internal mechanics imports outward.
func TierForLevel(level int) int {
	return mechanics.TierForLevel(level)
}

// RestParticipantInput captures one rest participant and their selected
// downtime moves for atomic rest resolution.
type RestParticipantInput struct {
	CharacterID ids.CharacterID
	Level       int
	State       daggerheartstate.CharacterState
	Moves       []DowntimeSelection
}

// DowntimeSelection captures one typed downtime choice inside an atomic rest.
type DowntimeSelection struct {
	Move                string
	TargetCharacterID   ids.CharacterID
	GroupID             string
	RollSeed            *int64
	CountdownID         ids.CountdownID
	ProjectAdvanceMode  string
	ProjectAdvanceDelta int
	ProjectReason       string
}

// RestPackageInput captures transport-agnostic rest inputs plus typed downtime
// selections.
type RestPackageInput struct {
	RestType              RestType
	Interrupted           bool
	RestSeed              int64
	CurrentGMFear         int
	ConsecutiveShortRests int
	Participants          []RestParticipantInput
	AvailableCountdowns   map[ids.CountdownID]rules.Countdown
	LongTermCountdown     *rules.Countdown
}

// RestPackageResult captures the canonical rest command payload plus the IDs
// that transport should reload after the command commits.
type RestPackageResult struct {
	Payload             daggerheartpayload.RestTakePayload
	ParticipantIDs      []ids.CharacterID
	UpdatedCountdownIDs []ids.CountdownID
}

// ResolveRestPackage builds the canonical atomic rest payload, including
// resolved downtime selections and optional countdown mutations.
func ResolveRestPackage(input RestPackageInput) (RestPackageResult, error) {
	participants, participantStates, err := normalizeRestParticipants(input.Participants)
	if err != nil {
		return RestPackageResult{}, err
	}

	partySize := len(participants)
	outcome, err := ResolveRestOutcome(
		RestState{ConsecutiveShortRests: input.ConsecutiveShortRests},
		input.RestType,
		input.Interrupted,
		input.RestSeed,
		partySize,
	)
	if err != nil {
		return RestPackageResult{}, err
	}

	effectiveType := input.RestType
	if outcome.EffectiveType == RestTypeLong {
		effectiveType = RestTypeLong
	} else {
		effectiveType = RestTypeShort
	}

	payload := daggerheartpayload.RestTakePayload{
		RestType:         restTypeToPayloadString(effectiveType),
		Interrupted:      input.Interrupted,
		GMFearBefore:     input.CurrentGMFear,
		GMFearAfter:      clampGMFear(input.CurrentGMFear + outcome.GMFearGain),
		ShortRestsBefore: input.ConsecutiveShortRests,
		ShortRestsAfter:  outcome.State.ConsecutiveShortRests,
		RefreshRest:      outcome.RefreshRest,
		RefreshLongRest:  outcome.RefreshLongRest,
	}
	payload.Participants = append(payload.Participants, participants...)

	if !outcome.Applied {
		if hasAnyDowntimeSelections(input.Participants) {
			return RestPackageResult{}, fmt.Errorf("interrupted short rests cannot include downtime moves")
		}
		if input.LongTermCountdown != nil {
			return RestPackageResult{}, fmt.Errorf("interrupted short rests cannot advance a countdown")
		}
		return RestPackageResult{Payload: payload, ParticipantIDs: participants}, nil
	}

	countdownStates := make(map[ids.CountdownID]rules.Countdown, len(input.AvailableCountdowns)+1)
	for countdownID, countdown := range input.AvailableCountdowns {
		countdownStates[countdownID] = countdown
	}
	updatedCountdownIDs := make([]ids.CountdownID, 0, 1)
	if outcome.AdvanceCountdown && input.LongTermCountdown != nil {
		countdownStates[ids.CountdownID(strings.TrimSpace(input.LongTermCountdown.ID))] = *input.LongTermCountdown
	}

	if outcome.AdvanceCountdown && input.LongTermCountdown != nil {
		mutation, err := nextCountdownMutation(countdownStates, ids.CountdownID(input.LongTermCountdown.ID), 1, nil, countdowns.CountdownReasonLongRest)
		if err != nil {
			return RestPackageResult{}, err
		}
		payload.CampaignCountdownAdvances = append(payload.CampaignCountdownAdvances, mutation)
		payload.CountdownAdvances = append(payload.CountdownAdvances, mutation)
		updatedCountdownIDs = append(updatedCountdownIDs, mutation.CountdownID)
	}

	groupParticipantCounts := countPrepareGroups(input.Participants)
	for _, participant := range input.Participants {
		if len(participant.Moves) > 2 {
			return RestPackageResult{}, fmt.Errorf("character %q selected %d downtime moves; maximum is 2", participant.CharacterID, len(participant.Moves))
		}
		for _, selection := range participant.Moves {
			movePayload, countdownUpdate, err := resolveDowntimeSelection(
				effectiveType,
				participant,
				selection,
				participantStates,
				groupParticipantCounts,
				countdownStates,
			)
			if err != nil {
				return RestPackageResult{}, err
			}
			payload.DowntimeMoves = append(payload.DowntimeMoves, movePayload)
			if countdownUpdate != nil {
				payload.CampaignCountdownAdvances = append(payload.CampaignCountdownAdvances, *countdownUpdate)
				payload.CountdownAdvances = append(payload.CountdownAdvances, *countdownUpdate)
				updatedCountdownIDs = append(updatedCountdownIDs, countdownUpdate.CountdownID)
			}
		}
	}

	return RestPackageResult{
		Payload:             payload,
		ParticipantIDs:      participants,
		UpdatedCountdownIDs: slices.Compact(updatedCountdownIDs),
	}, nil
}

func normalizeRestParticipants(inputs []RestParticipantInput) ([]ids.CharacterID, map[ids.CharacterID]*daggerheartstate.CharacterState, error) {
	participants := make([]ids.CharacterID, 0, len(inputs))
	states := make(map[ids.CharacterID]*daggerheartstate.CharacterState, len(inputs))
	seen := make(map[ids.CharacterID]struct{}, len(inputs))
	for _, participant := range inputs {
		characterID := ids.CharacterID(strings.TrimSpace(participant.CharacterID.String()))
		if characterID == "" {
			return nil, nil, fmt.Errorf("rest participant character_id is required")
		}
		if _, exists := seen[characterID]; exists {
			return nil, nil, fmt.Errorf("rest participant %q is duplicated", characterID)
		}
		seen[characterID] = struct{}{}
		participants = append(participants, characterID)
		stateCopy := participant.State
		stateCopy.CharacterID = characterID.String()
		states[characterID] = &stateCopy
	}
	return participants, states, nil
}

func hasAnyDowntimeSelections(participants []RestParticipantInput) bool {
	for _, participant := range participants {
		if len(participant.Moves) > 0 {
			return true
		}
	}
	return false
}

func countPrepareGroups(participants []RestParticipantInput) map[string]int {
	counts := make(map[string]int)
	for _, participant := range participants {
		seenForParticipant := map[string]struct{}{}
		for _, move := range participant.Moves {
			if strings.TrimSpace(move.Move) != DowntimeMovePrepare {
				continue
			}
			groupID := strings.TrimSpace(move.GroupID)
			if groupID == "" {
				continue
			}
			if _, seen := seenForParticipant[groupID]; seen {
				continue
			}
			seenForParticipant[groupID] = struct{}{}
			counts[groupID]++
		}
	}
	return counts
}

func resolveDowntimeSelection(
	restType RestType,
	participant RestParticipantInput,
	selection DowntimeSelection,
	participantStates map[ids.CharacterID]*daggerheartstate.CharacterState,
	groupParticipantCounts map[string]int,
	countdownStates map[ids.CountdownID]rules.Countdown,
) (daggerheartpayload.DowntimeMoveAppliedPayload, *daggerheartpayload.CampaignCountdownAdvancePayload, error) {
	move := strings.TrimSpace(strings.ToLower(selection.Move))
	if move == "" {
		return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("downtime move is required for %q", participant.CharacterID)
	}
	if !restTypeAllowsMove(restType, move) {
		return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("%s is not allowed during a %s rest", move, restTypeToPayloadString(restType))
	}

	payload := daggerheartpayload.DowntimeMoveAppliedPayload{
		ActorCharacterID: participant.CharacterID,
		Move:             move,
		RestType:         restTypeToPayloadString(restType),
	}

	switch move {
	case DowntimeMoveTendToWounds:
		targetID, targetState, err := resolveRestTarget(participant.CharacterID, selection.TargetCharacterID, participantStates)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		if selection.RollSeed == nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("tend_to_wounds requires rng")
		}
		amount, err := rollDowntimeAmount(*selection.RollSeed, participant.Level)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		_, after := targetState.Heal(amount)
		payload.TargetCharacterID = targetID
		payload.HP = &after
	case DowntimeMoveClearStress:
		actorState := participantStates[participant.CharacterID]
		if actorState == nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("rest participant %q state is missing", participant.CharacterID)
		}
		if selection.RollSeed == nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("clear_stress requires rng")
		}
		amount, err := rollDowntimeAmount(*selection.RollSeed, participant.Level)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		actorState.SetStress(actorState.Stress - amount)
		payload.TargetCharacterID = participant.CharacterID
		payload.Stress = &actorState.Stress
	case DowntimeMoveRepairArmor:
		targetID, targetState, err := resolveRestTarget(participant.CharacterID, selection.TargetCharacterID, participantStates)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		if selection.RollSeed == nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("repair_armor requires rng")
		}
		amount, err := rollDowntimeAmount(*selection.RollSeed, participant.Level)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		_, after, err := targetState.GainResource(mechanics.ResourceArmor, amount)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		payload.TargetCharacterID = targetID
		payload.Armor = &after
	case DowntimeMovePrepare:
		actorState := participantStates[participant.CharacterID]
		if actorState == nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("rest participant %q state is missing", participant.CharacterID)
		}
		groupID := strings.TrimSpace(selection.GroupID)
		hopeGain := 1
		if groupID != "" && groupParticipantCounts[groupID] >= 2 {
			hopeGain = 2
			payload.GroupID = groupID
		}
		_, after, err := actorState.GainResource(mechanics.ResourceHope, hopeGain)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		payload.TargetCharacterID = participant.CharacterID
		payload.Hope = &after
	case DowntimeMoveTendToAllWounds:
		targetID, targetState, err := resolveRestTarget(participant.CharacterID, selection.TargetCharacterID, participantStates)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		targetState.HP = targetState.HPMax
		payload.TargetCharacterID = targetID
		payload.HP = &targetState.HP
	case DowntimeMoveClearAllStress:
		actorState := participantStates[participant.CharacterID]
		if actorState == nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("rest participant %q state is missing", participant.CharacterID)
		}
		actorState.SetStress(0)
		payload.TargetCharacterID = participant.CharacterID
		payload.Stress = &actorState.Stress
	case DowntimeMoveRepairAllArmor:
		targetID, targetState, err := resolveRestTarget(participant.CharacterID, selection.TargetCharacterID, participantStates)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		targetState.SetArmor(targetState.ArmorMax)
		payload.TargetCharacterID = targetID
		payload.Armor = &targetState.Armor
	case DowntimeMoveWorkOnProject:
		countdownID := ids.CountdownID(strings.TrimSpace(selection.CountdownID.String()))
		if countdownID == "" {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("work_on_project requires countdown_id")
		}
		mode := strings.TrimSpace(strings.ToLower(selection.ProjectAdvanceMode))
		if mode == "" {
			mode = ProjectAdvanceModeAuto
		}
		var (
			delta    int
			override *int
			reason   string
		)
		switch mode {
		case ProjectAdvanceModeAuto:
			delta = 1
			reason = "work_on_project"
		case ProjectAdvanceModeGMSetDelta:
			if selection.ProjectAdvanceDelta == 0 {
				return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("work_on_project gm_set_delta requires non-zero advance_delta")
			}
			if strings.TrimSpace(selection.ProjectReason) == "" {
				return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("work_on_project gm_set_delta requires reason")
			}
			delta = selection.ProjectAdvanceDelta
			reason = strings.TrimSpace(selection.ProjectReason)
		default:
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("work_on_project advance mode %q is invalid", selection.ProjectAdvanceMode)
		}
		mutation, err := nextCountdownMutation(countdownStates, countdownID, delta, override, reason)
		if err != nil {
			return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, err
		}
		payload.CampaignCountdownID = countdownID
		payload.CountdownID = countdownID
		return payload, &mutation, nil
	default:
		return daggerheartpayload.DowntimeMoveAppliedPayload{}, nil, fmt.Errorf("downtime move %q is invalid", move)
	}

	return payload, nil, nil
}

func resolveRestTarget(
	actorID ids.CharacterID,
	targetID ids.CharacterID,
	participantStates map[ids.CharacterID]*daggerheartstate.CharacterState,
) (ids.CharacterID, *daggerheartstate.CharacterState, error) {
	normalizedTarget := ids.CharacterID(strings.TrimSpace(targetID.String()))
	if normalizedTarget == "" {
		normalizedTarget = ids.CharacterID(strings.TrimSpace(actorID.String()))
	}
	targetState := participantStates[normalizedTarget]
	if targetState == nil {
		return "", nil, fmt.Errorf("target character %q is not participating in this rest", normalizedTarget)
	}
	return normalizedTarget, targetState, nil
}

func rollDowntimeAmount(seed int64, level int) (int, error) {
	roll, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 4, Count: 1}},
		Seed: seed,
	})
	if err != nil {
		return 0, err
	}
	return roll.Total + TierForLevel(level), nil
}

func restTypeAllowsMove(restType RestType, move string) bool {
	switch restType {
	case RestTypeShort:
		switch move {
		case DowntimeMoveTendToWounds, DowntimeMoveClearStress, DowntimeMoveRepairArmor, DowntimeMovePrepare:
			return true
		}
	case RestTypeLong:
		switch move {
		case DowntimeMoveTendToAllWounds, DowntimeMoveClearAllStress, DowntimeMoveRepairAllArmor, DowntimeMovePrepare, DowntimeMoveWorkOnProject:
			return true
		}
	}
	return false
}

func nextCountdownMutation(
	countdownStates map[ids.CountdownID]rules.Countdown,
	countdownID ids.CountdownID,
	delta int,
	override *int,
	reason string,
) (daggerheartpayload.CampaignCountdownAdvancePayload, error) {
	current, ok := countdownStates[countdownID]
	if !ok {
		return daggerheartpayload.CampaignCountdownAdvancePayload{}, fmt.Errorf("countdown %q is not available", countdownID)
	}
	if override != nil {
		return daggerheartpayload.CampaignCountdownAdvancePayload{}, fmt.Errorf("countdown override is no longer supported")
	}
	mutation, err := countdowns.ResolveCountdownAdvance(countdowns.CountdownAdvanceInput{
		Countdown: current,
		Amount:    delta,
		Reason:    strings.TrimSpace(reason),
	})
	if err != nil {
		return daggerheartpayload.CampaignCountdownAdvancePayload{}, err
	}
	countdownStates[countdownID] = mutation.Advance.Countdown
	return mutation.Payload, nil
}

func clampGMFear(value int) int {
	if value < daggerheartstate.GMFearMin {
		return daggerheartstate.GMFearMin
	}
	if value > daggerheartstate.GMFearMax {
		return daggerheartstate.GMFearMax
	}
	return value
}

func restTypeToPayloadString(restType RestType) string {
	if restType == RestTypeLong {
		return "long"
	}
	return "short"
}
