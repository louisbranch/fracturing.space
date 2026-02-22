package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
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
		var payload CharacterStatePatchPayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if isCharacterStatePatchNoMutation(snapshotState, payload) {
			// FIXME(telemetry): metric for idempotent character state patch commands.
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterStatePatchNoMutation,
				Message: "character state patch is unchanged",
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeCharacterStatePatched, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeConditionChange:
		var payload ConditionChangePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if hasMissingCharacterConditionRemovals(snapshotState, payload) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeConditionChangeRemoveMissing,
				Message: "condition remove requires an existing condition",
			})
		}
		if isConditionChangeNoMutation(snapshotState, payload) {
			// FIXME(telemetry): metric for idempotent character condition changes.
			return command.Reject(command.Rejection{
				Code:    rejectionCodeConditionChangeNoMutation,
				Message: "condition change is unchanged",
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeConditionChanged, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
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
		var payload LoadoutSwapPayload
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
		payload.CardID = strings.TrimSpace(payload.CardID)
		payload.From = strings.TrimSpace(payload.From)
		payload.To = strings.TrimSpace(payload.To)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeLoadoutSwapped, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
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
		var payload CountdownCreatePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.CountdownID = strings.TrimSpace(payload.CountdownID)
		payload.Name = strings.TrimSpace(payload.Name)
		payload.Kind = strings.TrimSpace(payload.Kind)
		payload.Direction = strings.TrimSpace(payload.Direction)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "countdown"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CountdownID
		}
		evt := command.NewEvent(cmd, EventTypeCountdownCreated, entityType, entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeCountdownUpdate:
		var payload CountdownUpdatePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if rejection := countdownUpdateSnapshotRejection(snapshotState, payload); rejection != nil {
			return command.Reject(*rejection)
		}
		if now == nil {
			now = time.Now
		}
		payload.CountdownID = strings.TrimSpace(payload.CountdownID)
		payload.Reason = strings.TrimSpace(payload.Reason)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "countdown"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CountdownID
		}
		evt := command.NewEvent(cmd, EventTypeCountdownUpdated, entityType, entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeCountdownDelete:
		var payload CountdownDeletePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.CountdownID = strings.TrimSpace(payload.CountdownID)
		payload.Reason = strings.TrimSpace(payload.Reason)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "countdown"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CountdownID
		}
		evt := command.NewEvent(cmd, EventTypeCountdownDeleted, entityType, entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
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
		var payload DowntimeMoveApplyPayload
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
		payload.Move = strings.TrimSpace(payload.Move)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeDowntimeMoveApplied, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeCharacterTemporaryArmorApply:
		var payload CharacterTemporaryArmorApplyPayload
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
		payload.Source = strings.TrimSpace(payload.Source)
		payload.Duration = strings.TrimSpace(payload.Duration)
		payload.SourceID = strings.TrimSpace(payload.SourceID)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := command.NewEvent(cmd, EventTypeCharacterTemporaryArmorApplied, "character", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeAdversaryConditionChange:
		var payload AdversaryConditionChangePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if hasMissingAdversaryConditionRemovals(snapshotState, payload) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeAdversaryConditionRemoveMissing,
				Message: "adversary condition remove requires an existing condition",
			})
		}
		if isAdversaryConditionChangeNoMutation(snapshotState, payload) {
			// FIXME(telemetry): metric for idempotent adversary condition changes.
			return command.Reject(command.Rejection{
				Code:    rejectionCodeAdversaryConditionNoMutation,
				Message: "adversary condition change is unchanged",
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := command.NewEvent(cmd, EventTypeAdversaryConditionChanged, "adversary", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeAdversaryCreate:
		var payload AdversaryCreatePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if isAdversaryCreateNoMutation(snapshotState, payload) {
			// FIXME(telemetry): metric for idempotent adversary creation commands.
			return command.Reject(command.Rejection{
				Code:    rejectionCodeAdversaryCreateNoMutation,
				Message: "adversary create is unchanged",
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payload.Name = strings.TrimSpace(payload.Name)
		payload.Kind = strings.TrimSpace(payload.Kind)
		payload.SessionID = strings.TrimSpace(payload.SessionID)
		payload.Notes = strings.TrimSpace(payload.Notes)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := command.NewEvent(cmd, EventTypeAdversaryCreated, "adversary", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeAdversaryUpdate:
		var payload AdversaryUpdatePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payload.Name = strings.TrimSpace(payload.Name)
		payload.Kind = strings.TrimSpace(payload.Kind)
		payload.SessionID = strings.TrimSpace(payload.SessionID)
		payload.Notes = strings.TrimSpace(payload.Notes)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := command.NewEvent(cmd, EventTypeAdversaryUpdated, "adversary", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
	case commandTypeAdversaryDelete:
		var payload AdversaryDeletePayload
		if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
			return command.Reject(command.Rejection{
				Code:    rejectionCodePayloadDecodeFailed,
				Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
			})
		}
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payload.Reason = strings.TrimSpace(payload.Reason)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := command.NewEvent(cmd, EventTypeAdversaryDeleted, "adversary", entityID, payloadJSON, now().UTC())

		return command.Accept(evt)
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
