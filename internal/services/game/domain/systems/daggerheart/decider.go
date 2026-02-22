package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

const (
	commandTypeGMFearSet                    command.Type = "sys.daggerheart.gm_fear.set"
	commandTypeCharacterStatePatch          command.Type = "sys.daggerheart.character_state.patch"
	commandTypeConditionChange              command.Type = "sys.daggerheart.condition.change"
	commandTypeHopeSpend                    command.Type = "sys.daggerheart.hope.spend"
	commandTypeStressSpend                  command.Type = "sys.daggerheart.stress.spend"
	commandTypeLoadoutSwap                  command.Type = "sys.daggerheart.loadout.swap"
	commandTypeRestTake                     command.Type = "sys.daggerheart.rest.take"
	commandTypeCountdownCreate              command.Type = "sys.daggerheart.countdown.create"
	commandTypeCountdownUpdate              command.Type = "sys.daggerheart.countdown.update"
	commandTypeCountdownDelete              command.Type = "sys.daggerheart.countdown.delete"
	commandTypeDamageApply                  command.Type = "sys.daggerheart.damage.apply"
	commandTypeAdversaryDamageApply         command.Type = "sys.daggerheart.adversary_damage.apply"
	commandTypeDowntimeMoveApply            command.Type = "sys.daggerheart.downtime_move.apply"
	commandTypeCharacterTemporaryArmorApply command.Type = "sys.daggerheart.character_temporary_armor.apply"
	commandTypeAdversaryConditionChange     command.Type = "sys.daggerheart.adversary_condition.change"
	commandTypeAdversaryCreate              command.Type = "sys.daggerheart.adversary.create"
	commandTypeAdversaryUpdate              command.Type = "sys.daggerheart.adversary.update"
	commandTypeAdversaryDelete              command.Type = "sys.daggerheart.adversary.delete"

	rejectionCodeGMFearAfterRequired             = "GM_FEAR_AFTER_REQUIRED"
	rejectionCodeGMFearOutOfRange                = "GM_FEAR_AFTER_OUT_OF_RANGE"
	rejectionCodeGMFearUnchanged                 = "GM_FEAR_UNCHANGED"
	rejectionCodeCharacterStatePatchNoMutation   = "CHARACTER_STATE_PATCH_NO_MUTATION"
	rejectionCodeConditionChangeNoMutation       = "CONDITION_CHANGE_NO_MUTATION"
	rejectionCodeConditionChangeRemoveMissing    = "CONDITION_CHANGE_REMOVE_MISSING"
	rejectionCodeCountdownUpdateNoMutation       = "COUNTDOWN_UPDATE_NO_MUTATION"
	rejectionCodeCountdownBeforeMismatch         = "COUNTDOWN_BEFORE_MISMATCH"
	rejectionCodeDamageBeforeMismatch            = "DAMAGE_BEFORE_MISMATCH"
	rejectionCodeDamageArmorSpendLimit           = "DAMAGE_ARMOR_SPEND_LIMIT"
	rejectionCodeAdversaryDamageBeforeMismatch   = "ADVERSARY_DAMAGE_BEFORE_MISMATCH"
	rejectionCodeAdversaryConditionNoMutation    = "ADVERSARY_CONDITION_NO_MUTATION"
	rejectionCodeAdversaryConditionRemoveMissing = "ADVERSARY_CONDITION_REMOVE_MISSING"
	rejectionCodeAdversaryCreateNoMutation       = "ADVERSARY_CREATE_NO_MUTATION"
	rejectionCodePayloadDecodeFailed             = "PAYLOAD_DECODE_FAILED"
	rejectionCodeCommandTypeUnsupported          = "COMMAND_TYPE_UNSUPPORTED"
)

// Decider handles Daggerheart system commands.
type Decider struct{}

// DeciderHandledCommands returns the command types this decider handles.
// Derived from daggerheartCommandDefinitions so the list stays in sync.
func (Decider) DeciderHandledCommands() []command.Type {
	return commandTypesFromDefinitions()
}

// Decide returns the decision for a system command against current state.
func (Decider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	snapshotState, hasSnapshot := snapshotFromState(state)
	switch cmd.Type {
	case commandTypeGMFearSet:
		var payload GMFearSetPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if payload.After == nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeGMFearAfterRequired,
				Message: "gm fear after is required",
			})
		}
		after := *payload.After
		if after < GMFearMin || after > GMFearMax {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeGMFearOutOfRange,
				Message: "gm fear after is out of range",
			})
		}
		before := GMFearDefault
		if hasSnapshot {
			before = snapshotState.GMFear
		}
		if after == before {
			// FIXME(telemetry): metric for idempotent gm fear set commands (no-op reject).
			return command.Reject(command.Rejection{
				Code:    rejectionCodeGMFearUnchanged,
				Message: "gm fear after is unchanged",
			})
		}
		if now == nil {
			now = time.Now
		}

		changed := GMFearChangedPayload{
			Before: before,
			After:  after,
			Reason: strings.TrimSpace(payload.Reason),
		}
		payloadJSON, _ := json.Marshal(changed)
		evt := command.NewEvent(cmd, EventTypeGMFearChanged, "campaign", cmd.CampaignID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeCharacterStatePatch:
		return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeCharacterStatePatched, "character",
			func(p *CharacterStatePatchPayload) string { return strings.TrimSpace(p.CharacterID) },
			func(s SnapshotState, hasState bool, p *CharacterStatePatchPayload, _ func() time.Time) *command.Rejection {
				if hasState && isCharacterStatePatchNoMutation(s, *p) {
					// FIXME(telemetry): metric for idempotent character state patch commands.
					return &command.Rejection{
						Code:    rejectionCodeCharacterStatePatchNoMutation,
						Message: "character state patch is unchanged",
					}
				}
				p.CharacterID = strings.TrimSpace(p.CharacterID)
				return nil
			}, now)
	case commandTypeConditionChange:
		return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeConditionChanged, "character",
			func(p *ConditionChangePayload) string { return strings.TrimSpace(p.CharacterID) },
			func(s SnapshotState, hasState bool, p *ConditionChangePayload, _ func() time.Time) *command.Rejection {
				if hasState {
					if hasMissingCharacterConditionRemovals(s, *p) {
						return &command.Rejection{
							Code:    rejectionCodeConditionChangeRemoveMissing,
							Message: "condition remove requires an existing condition",
						}
					}
					if isConditionChangeNoMutation(s, *p) {
						// FIXME(telemetry): metric for idempotent character condition changes.
						return &command.Rejection{
							Code:    rejectionCodeConditionChangeNoMutation,
							Message: "condition change is unchanged",
						}
					}
				}
				p.CharacterID = strings.TrimSpace(p.CharacterID)
				p.Source = strings.TrimSpace(p.Source)
				return nil
			}, now)
	case commandTypeHopeSpend:
		var payload HopeSpendPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payloadJSON, _ := json.Marshal(CharacterStatePatchedPayload{
			CharacterID: payload.CharacterID,
			HopeBefore:  &payload.Before,
			HopeAfter:   &payload.After,
		})
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeCharacterStatePatched, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeStressSpend:
		var payload StressSpendPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payloadJSON, _ := json.Marshal(CharacterStatePatchedPayload{
			CharacterID:  payload.CharacterID,
			StressBefore: &payload.Before,
			StressAfter:  &payload.After,
		})
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeCharacterStatePatched, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeLoadoutSwap:
		return module.DecideFunc(cmd, EventTypeLoadoutSwapped, "character",
			func(p *LoadoutSwapPayload) string { return strings.TrimSpace(p.CharacterID) },
			func(p *LoadoutSwapPayload, _ func() time.Time) *command.Rejection {
				p.CharacterID = strings.TrimSpace(p.CharacterID)
				p.CardID = strings.TrimSpace(p.CardID)
				p.From = strings.TrimSpace(p.From)
				p.To = strings.TrimSpace(p.To)
				return nil
			}, now)
	case commandTypeRestTake:
		var payload RestTakePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.RestType = strings.TrimSpace(payload.RestType)
		if payload.LongTermCountdown != nil {
			if rejection := countdownUpdateSnapshotRejection(snapshotState, *payload.LongTermCountdown); rejection != nil {
				return command.Reject(*rejection)
			}
			payload.LongTermCountdown.CountdownID = strings.TrimSpace(payload.LongTermCountdown.CountdownID)
			payload.LongTermCountdown.Reason = strings.TrimSpace(payload.LongTermCountdown.Reason)
		}
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.CampaignID
		}
		restEvent := command.NewEvent(cmd, EventTypeRestTaken, "session", entityID, payloadJSON, now().UTC())

		if payload.LongTermCountdown == nil {
			return command.Accept(restEvent)
		}
		countdownPayload := *payload.LongTermCountdown
		countdownPayloadJSON, _ := json.Marshal(countdownPayload)
		countdownEvent := command.NewEvent(cmd, EventTypeCountdownUpdated, "countdown", countdownPayload.CountdownID, countdownPayloadJSON, now().UTC())

		return command.Accept(restEvent, countdownEvent)
	case commandTypeCountdownCreate:
		return module.DecideFunc(cmd, EventTypeCountdownCreated, "countdown",
			func(p *CountdownCreatePayload) string { return strings.TrimSpace(p.CountdownID) },
			func(p *CountdownCreatePayload, _ func() time.Time) *command.Rejection {
				p.CountdownID = strings.TrimSpace(p.CountdownID)
				p.Name = strings.TrimSpace(p.Name)
				p.Kind = strings.TrimSpace(p.Kind)
				p.Direction = strings.TrimSpace(p.Direction)
				return nil
			}, now)
	case commandTypeCountdownUpdate:
		return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeCountdownUpdated, "countdown",
			func(p *CountdownUpdatePayload) string { return strings.TrimSpace(p.CountdownID) },
			func(s SnapshotState, hasState bool, p *CountdownUpdatePayload, _ func() time.Time) *command.Rejection {
				if hasState {
					if rejection := countdownUpdateSnapshotRejection(s, *p); rejection != nil {
						return rejection
					}
				}
				p.CountdownID = strings.TrimSpace(p.CountdownID)
				p.Reason = strings.TrimSpace(p.Reason)
				return nil
			}, now)
	case commandTypeCountdownDelete:
		return module.DecideFunc(cmd, EventTypeCountdownDeleted, "countdown",
			func(p *CountdownDeletePayload) string { return strings.TrimSpace(p.CountdownID) },
			func(p *CountdownDeletePayload, _ func() time.Time) *command.Rejection {
				p.CountdownID = strings.TrimSpace(p.CountdownID)
				p.Reason = strings.TrimSpace(p.Reason)
				return nil
			}, now)
	case commandTypeDamageApply:
		var payload DamageApplyPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if payload.ArmorSpent > 1 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeDamageArmorSpendLimit,
				Message: "damage apply can spend at most one armor slot",
			})
		}
		if hasSnapshot {
			if character, ok := snapshotCharacterState(snapshotState, payload.CharacterID); ok {
				if payload.HpBefore != nil && character.HP != *payload.HpBefore {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeDamageBeforeMismatch,
						Message: "damage before does not match current state",
					})
				}
				if payload.ArmorBefore != nil && character.Armor != *payload.ArmorBefore {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeDamageBeforeMismatch,
						Message: "damage before does not match current state",
					})
				}
			}
		}
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payload.DamageType = strings.TrimSpace(payload.DamageType)
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeDamageApplied, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeAdversaryDamageApply:
		var payload AdversaryDamageApplyPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if hasSnapshot {
			if adversary, ok := snapshotAdversaryState(snapshotState, payload.AdversaryID); ok {
				if payload.HpBefore != nil && adversary.HP != *payload.HpBefore {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeAdversaryDamageBeforeMismatch,
						Message: "adversary damage before does not match current state",
					})
				}
				if payload.ArmorBefore != nil && adversary.Armor != *payload.ArmorBefore {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeAdversaryDamageBeforeMismatch,
						Message: "adversary damage before does not match current state",
					})
				}
			}
		}
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payload.DamageType = strings.TrimSpace(payload.DamageType)
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := command.NewEvent(cmd, EventTypeAdversaryDamageApplied, "adversary", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeDowntimeMoveApply:
		return module.DecideFunc(cmd, EventTypeDowntimeMoveApplied, "character",
			func(p *DowntimeMoveApplyPayload) string { return strings.TrimSpace(p.CharacterID) },
			func(p *DowntimeMoveApplyPayload, _ func() time.Time) *command.Rejection {
				p.CharacterID = strings.TrimSpace(p.CharacterID)
				p.Move = strings.TrimSpace(p.Move)
				return nil
			}, now)
	case commandTypeCharacterTemporaryArmorApply:
		return module.DecideFunc(cmd, EventTypeCharacterTemporaryArmorApplied, "character",
			func(p *CharacterTemporaryArmorApplyPayload) string { return strings.TrimSpace(p.CharacterID) },
			func(p *CharacterTemporaryArmorApplyPayload, _ func() time.Time) *command.Rejection {
				p.CharacterID = strings.TrimSpace(p.CharacterID)
				p.Source = strings.TrimSpace(p.Source)
				p.Duration = strings.TrimSpace(p.Duration)
				p.SourceID = strings.TrimSpace(p.SourceID)
				return nil
			}, now)
	case commandTypeAdversaryConditionChange:
		return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeAdversaryConditionChanged, "adversary",
			func(p *AdversaryConditionChangePayload) string { return strings.TrimSpace(p.AdversaryID) },
			func(s SnapshotState, hasState bool, p *AdversaryConditionChangePayload, _ func() time.Time) *command.Rejection {
				if hasState {
					if hasMissingAdversaryConditionRemovals(s, *p) {
						return &command.Rejection{
							Code:    rejectionCodeAdversaryConditionRemoveMissing,
							Message: "adversary condition remove requires an existing condition",
						}
					}
					if isAdversaryConditionChangeNoMutation(s, *p) {
						// FIXME(telemetry): metric for idempotent adversary condition changes.
						return &command.Rejection{
							Code:    rejectionCodeAdversaryConditionNoMutation,
							Message: "adversary condition change is unchanged",
						}
					}
				}
				p.AdversaryID = strings.TrimSpace(p.AdversaryID)
				p.Source = strings.TrimSpace(p.Source)
				return nil
			}, now)
	case commandTypeAdversaryCreate:
		return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeAdversaryCreated, "adversary",
			func(p *AdversaryCreatePayload) string { return strings.TrimSpace(p.AdversaryID) },
			func(s SnapshotState, hasState bool, p *AdversaryCreatePayload, _ func() time.Time) *command.Rejection {
				if hasState && isAdversaryCreateNoMutation(s, *p) {
					// FIXME(telemetry): metric for idempotent adversary creation commands.
					return &command.Rejection{
						Code:    rejectionCodeAdversaryCreateNoMutation,
						Message: "adversary create is unchanged",
					}
				}
				p.AdversaryID = strings.TrimSpace(p.AdversaryID)
				p.Name = strings.TrimSpace(p.Name)
				p.Kind = strings.TrimSpace(p.Kind)
				p.SessionID = strings.TrimSpace(p.SessionID)
				p.Notes = strings.TrimSpace(p.Notes)
				return nil
			}, now)
	case commandTypeAdversaryUpdate:
		return module.DecideFunc(cmd, EventTypeAdversaryUpdated, "adversary",
			func(p *AdversaryUpdatePayload) string { return strings.TrimSpace(p.AdversaryID) },
			func(p *AdversaryUpdatePayload, _ func() time.Time) *command.Rejection {
				p.AdversaryID = strings.TrimSpace(p.AdversaryID)
				p.Name = strings.TrimSpace(p.Name)
				p.Kind = strings.TrimSpace(p.Kind)
				p.SessionID = strings.TrimSpace(p.SessionID)
				p.Notes = strings.TrimSpace(p.Notes)
				return nil
			}, now)
	case commandTypeAdversaryDelete:
		return module.DecideFunc(cmd, EventTypeAdversaryDeleted, "adversary",
			func(p *AdversaryDeletePayload) string { return strings.TrimSpace(p.AdversaryID) },
			func(p *AdversaryDeletePayload, _ func() time.Time) *command.Rejection {
				p.AdversaryID = strings.TrimSpace(p.AdversaryID)
				p.Reason = strings.TrimSpace(p.Reason)
				return nil
			}, now)
	default:
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCommandTypeUnsupported,
			Message: "command type is not supported by daggerheart decider",
		})
	}
}

func isCharacterStatePatchNoMutation(snapshot SnapshotState, payload CharacterStatePatchPayload) bool {
	character, hasCharacter := snapshotCharacterState(snapshot, payload.CharacterID)
	if !hasCharacter {
		return false
	}

	if payload.HPAfter != nil {
		if character.HP != *payload.HPAfter {
			return false
		}
	} else if payload.HPBefore != nil && character.HP == 0 && character.HP != *payload.HPBefore {
		return false
	}
	if payload.HopeAfter != nil && character.Hope != *payload.HopeAfter {
		return false
	}
	if payload.HopeMaxAfter != nil && character.HopeMax != *payload.HopeMaxAfter {
		return false
	}
	if payload.StressAfter != nil && character.Stress != *payload.StressAfter {
		return false
	}
	if payload.ArmorAfter != nil && character.Armor != *payload.ArmorAfter {
		return false
	}
	if payload.LifeStateAfter != nil && character.LifeState != *payload.LifeStateAfter {
		return false
	}

	return true
}

func isConditionChangeNoMutation(snapshot SnapshotState, payload ConditionChangePayload) bool {
	character, hasCharacter := snapshotCharacterState(snapshot, payload.CharacterID)
	if !hasCharacter {
		return false
	}

	current, err := NormalizeConditions(character.Conditions)
	if err != nil {
		return false
	}
	after, err := NormalizeConditions(payload.ConditionsAfter)
	if err != nil {
		return false
	}
	return ConditionsEqual(current, after)
}

func isCountdownUpdateNoMutation(snapshot SnapshotState, payload CountdownUpdatePayload) bool {
	countdown, hasCountdown := snapshotCountdownState(snapshot, payload.CountdownID)
	if !hasCountdown {
		return false
	}
	if countdown.Current != payload.After {
		return false
	}
	if payload.Looped && !countdown.Looping {
		return false
	}
	return true
}

func countdownUpdateSnapshotRejection(snapshot SnapshotState, payload CountdownUpdatePayload) *command.Rejection {
	if countdown, hasCountdown := snapshotCountdownState(snapshot, payload.CountdownID); hasCountdown && payload.Before != countdown.Current {
		return &command.Rejection{
			Code:    rejectionCodeCountdownBeforeMismatch,
			Message: "countdown before does not match current state",
		}
	}
	if isCountdownUpdateNoMutation(snapshot, payload) {
		// FIXME(telemetry): metric for idempotent countdown updates.
		return &command.Rejection{
			Code:    rejectionCodeCountdownUpdateNoMutation,
			Message: "countdown update is unchanged",
		}
	}
	return nil
}

func isAdversaryConditionChangeNoMutation(snapshot SnapshotState, payload AdversaryConditionChangePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}

	current, err := NormalizeConditions(adversary.Conditions)
	if err != nil {
		return false
	}
	after, err := NormalizeConditions(payload.ConditionsAfter)
	if err != nil {
		return false
	}
	return ConditionsEqual(current, after)
}

func hasMissingCharacterConditionRemovals(snapshot SnapshotState, payload ConditionChangePayload) bool {
	if len(payload.Removed) == 0 {
		return false
	}
	character, hasCharacter := snapshotCharacterState(snapshot, payload.CharacterID)
	if !hasCharacter {
		return false
	}
	return hasMissingConditionRemovals(character.Conditions, payload.Removed)
}

func hasMissingAdversaryConditionRemovals(snapshot SnapshotState, payload AdversaryConditionChangePayload) bool {
	if len(payload.Removed) == 0 {
		return false
	}
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}
	return hasMissingConditionRemovals(adversary.Conditions, payload.Removed)
}

func hasMissingConditionRemovals(current, removed []string) bool {
	normalizedCurrent, err := NormalizeConditions(current)
	if err != nil {
		return false
	}
	normalizedRemoved, err := NormalizeConditions(removed)
	if err != nil {
		return false
	}

	currentSet := make(map[string]struct{}, len(normalizedCurrent))
	for _, value := range normalizedCurrent {
		currentSet[value] = struct{}{}
	}
	for _, value := range normalizedRemoved {
		if _, ok := currentSet[value]; !ok {
			return true
		}
	}
	return false
}

func isAdversaryCreateNoMutation(snapshot SnapshotState, payload AdversaryCreatePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}
	return adversary.Name == strings.TrimSpace(payload.Name) &&
		adversary.Kind == strings.TrimSpace(payload.Kind) &&
		adversary.SessionID == strings.TrimSpace(payload.SessionID) &&
		adversary.Notes == strings.TrimSpace(payload.Notes) &&
		adversary.HP == payload.HP &&
		adversary.HPMax == payload.HPMax &&
		adversary.Stress == payload.Stress &&
		adversary.StressMax == payload.StressMax &&
		adversary.Evasion == payload.Evasion &&
		adversary.Major == payload.Major &&
		adversary.Severe == payload.Severe &&
		adversary.Armor == payload.Armor
}

func snapshotCharacterState(snapshot SnapshotState, characterID string) (CharacterState, bool) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return CharacterState{}, false
	}
	character, ok := snapshot.CharacterStates[characterID]
	if !ok {
		return CharacterState{}, false
	}
	character.CharacterID = characterID
	character.CampaignID = snapshot.CampaignID
	if character.LifeState == "" {
		character.LifeState = LifeStateAlive
	}
	return character, true
}

func snapshotAdversaryState(snapshot SnapshotState, adversaryID string) (AdversaryState, bool) {
	adversaryID = strings.TrimSpace(adversaryID)
	if adversaryID == "" {
		return AdversaryState{}, false
	}
	adversary, ok := snapshot.AdversaryStates[adversaryID]
	if !ok {
		return AdversaryState{}, false
	}
	adversary.AdversaryID = adversaryID
	adversary.CampaignID = snapshot.CampaignID
	return adversary, true
}

func snapshotCountdownState(snapshot SnapshotState, countdownID string) (CountdownState, bool) {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return CountdownState{}, false
	}
	countdown, ok := snapshot.CountdownStates[countdownID]
	if !ok {
		return CountdownState{}, false
	}
	countdown.CountdownID = countdownID
	countdown.CampaignID = snapshot.CampaignID
	return countdown, true
}
