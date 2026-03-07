package daggerheart

import "strings"

// RestApplicationInput captures transport-agnostic rest application inputs.
type RestApplicationInput struct {
	RestType               RestType
	Interrupted            bool
	Outcome                RestOutcome
	CurrentGMFear          int
	ConsecutiveShortRests  int
	CharacterIDs           []string
	LongTermCountdownState *Countdown
}

// ResolveRestApplication builds a canonical rest payload from resolved rest
// mechanics and optional countdown state.
func ResolveRestApplication(input RestApplicationInput) (RestTakePayload, error) {
	gmFearBefore := input.CurrentGMFear
	gmFearAfter := gmFearBefore + input.Outcome.GMFearGain
	if gmFearAfter > GMFearMax {
		gmFearAfter = GMFearMax
	}

	payload := RestTakePayload{
		RestType:         restTypeToPayloadString(input.RestType),
		Interrupted:      input.Interrupted,
		GMFearBefore:     gmFearBefore,
		GMFearAfter:      gmFearAfter,
		ShortRestsBefore: input.ConsecutiveShortRests,
		ShortRestsAfter:  input.Outcome.State.ConsecutiveShortRests,
		RefreshRest:      input.Outcome.RefreshRest,
		RefreshLongRest:  input.Outcome.RefreshLongRest,
	}

	if input.Outcome.AdvanceCountdown && input.LongTermCountdownState != nil {
		mutation, err := ResolveCountdownMutation(CountdownMutationInput{
			Countdown: *input.LongTermCountdownState,
			Delta:     1,
			Reason:    CountdownReasonLongRest,
		})
		if err != nil {
			return RestTakePayload{}, err
		}
		payload.LongTermCountdown = &mutation.Payload
	}

	if len(input.CharacterIDs) > 0 {
		payload.CharacterStates = make([]RestCharacterStatePatch, 0, len(input.CharacterIDs))
		for _, characterID := range input.CharacterIDs {
			characterID = strings.TrimSpace(characterID)
			if characterID == "" {
				continue
			}
			payload.CharacterStates = append(payload.CharacterStates, RestCharacterStatePatch{
				CharacterID: characterID,
			})
		}
		if len(payload.CharacterStates) == 0 {
			payload.CharacterStates = nil
		}
	}

	return payload, nil
}

func restTypeToPayloadString(restType RestType) string {
	if restType == RestTypeLong {
		return "long"
	}
	return "short"
}
