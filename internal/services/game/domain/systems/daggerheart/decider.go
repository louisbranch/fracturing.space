package daggerheart

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
	eventTypeGMFearChanged                  event.Type   = "sys.daggerheart.gm_fear_changed"
	eventTypeCharacterStatePatched          event.Type   = "sys.daggerheart.character_state_patched"
	eventTypeConditionChanged               event.Type   = "sys.daggerheart.condition_changed"
	eventTypeLoadoutSwapped                 event.Type   = "sys.daggerheart.loadout_swapped"
	eventTypeRestTaken                      event.Type   = "sys.daggerheart.rest_taken"
	eventTypeCountdownCreated               event.Type   = "sys.daggerheart.countdown_created"
	eventTypeCountdownUpdated               event.Type   = "sys.daggerheart.countdown_updated"
	eventTypeCountdownDeleted               event.Type   = "sys.daggerheart.countdown_deleted"
	eventTypeDamageApplied                  event.Type   = "sys.daggerheart.damage_applied"
	eventTypeAdversaryDamageApplied         event.Type   = "sys.daggerheart.adversary_damage_applied"
	eventTypeDowntimeMoveApplied            event.Type   = "sys.daggerheart.downtime_move_applied"
	eventTypeCharacterTemporaryArmorApplied event.Type   = "sys.daggerheart.character_temporary_armor_applied"
	eventTypeAdversaryConditionChanged      event.Type   = "sys.daggerheart.adversary_condition_changed"
	eventTypeAdversaryCreated               event.Type   = "sys.daggerheart.adversary_created"
	eventTypeAdversaryUpdated               event.Type   = "sys.daggerheart.adversary_updated"
	eventTypeAdversaryDeleted               event.Type   = "sys.daggerheart.adversary_deleted"

	rejectionCodeGMFearAfterRequired           = "GM_FEAR_AFTER_REQUIRED"
	rejectionCodeGMFearOutOfRange              = "GM_FEAR_AFTER_OUT_OF_RANGE"
	rejectionCodeGMFearUnchanged               = "GM_FEAR_UNCHANGED"
	rejectionCodeCharacterStatePatchNoMutation = "CHARACTER_STATE_PATCH_NO_MUTATION"
	rejectionCodeConditionChangeNoMutation     = "CONDITION_CHANGE_NO_MUTATION"
	rejectionCodeCountdownUpdateNoMutation     = "COUNTDOWN_UPDATE_NO_MUTATION"
	rejectionCodeAdversaryConditionNoMutation  = "ADVERSARY_CONDITION_NO_MUTATION"
	rejectionCodeAdversaryCreateNoMutation     = "ADVERSARY_CREATE_NO_MUTATION"
)

// Decider handles Daggerheart system commands.
type Decider struct{}

// Decide returns the decision for a system command against current state.
func (Decider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
	snapshotState, hasSnapshot := snapshotFromState(state)
	switch cmd.Type {
	case commandTypeGMFearSet:
		var payload GMFearSetPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeGMFearChanged,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "campaign",
			EntityID:      cmd.CampaignID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeCharacterStatePatch:
		var payload CharacterStatePatchPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCharacterStatePatched,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeConditionChange:
		var payload ConditionChangePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeConditionChanged,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeHopeSpend:
		var payload HopeSpendPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCharacterStatePatched,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeStressSpend:
		var payload StressSpendPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCharacterStatePatched,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeLoadoutSwap:
		var payload LoadoutSwapPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeLoadoutSwapped,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeRestTake:
		var payload RestTakePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.RestType = strings.TrimSpace(payload.RestType)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.CampaignID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeRestTaken,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "session",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeCountdownCreate:
		var payload CountdownCreatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCountdownCreated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    entityType,
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeCountdownUpdate:
		var payload CountdownUpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if isCountdownUpdateNoMutation(snapshotState, payload) {
			// FIXME(telemetry): metric for idempotent countdown updates.
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCountdownUpdateNoMutation,
				Message: "countdown update is unchanged",
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCountdownUpdated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    entityType,
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeCountdownDelete:
		var payload CountdownDeletePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCountdownDeleted,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    entityType,
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeDamageApply:
		var payload DamageApplyPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeDamageApplied,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeAdversaryDamageApply:
		var payload AdversaryDamageApplyPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryDamageApplied,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "adversary",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeDowntimeMoveApply:
		var payload DowntimeMoveApplyPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeDowntimeMoveApplied,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeCharacterTemporaryArmorApply:
		var payload CharacterTemporaryArmorApplyPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCharacterTemporaryArmorApplied,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeAdversaryConditionChange:
		var payload AdversaryConditionChangePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryConditionChanged,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "adversary",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeAdversaryCreate:
		var payload AdversaryCreatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryCreated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "adversary",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeAdversaryUpdate:
		var payload AdversaryUpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryUpdated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "adversary",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	case commandTypeAdversaryDelete:
		var payload AdversaryDeletePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryDeleted,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "adversary",
			EntityID:      entityID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	default:
		return command.Decision{}
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
