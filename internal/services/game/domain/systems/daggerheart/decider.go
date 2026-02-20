package daggerheart

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeGMFearSet                command.Type = "sys.daggerheart.gm_fear.set"
	commandTypeCharacterStatePatch      command.Type = "sys.daggerheart.character_state.patch"
	commandTypeConditionChange          command.Type = "sys.daggerheart.condition.change"
	commandTypeHopeSpend                command.Type = "sys.daggerheart.hope.spend"
	commandTypeStressSpend              command.Type = "sys.daggerheart.stress.spend"
	commandTypeLoadoutSwap              command.Type = "sys.daggerheart.loadout.swap"
	commandTypeRestTake                 command.Type = "sys.daggerheart.rest.take"
	commandTypeCountdownCreate          command.Type = "sys.daggerheart.countdown.create"
	commandTypeCountdownUpdate          command.Type = "sys.daggerheart.countdown.update"
	commandTypeCountdownDelete          command.Type = "sys.daggerheart.countdown.delete"
	commandTypeDamageApply              command.Type = "sys.daggerheart.damage.apply"
	commandTypeAdversaryDamageApply     command.Type = "sys.daggerheart.adversary_damage.apply"
	commandTypeDowntimeMoveApply        command.Type = "sys.daggerheart.downtime_move.apply"
	commandTypeAdversaryConditionChange command.Type = "sys.daggerheart.adversary_condition.change"
	commandTypeAdversaryCreate          command.Type = "sys.daggerheart.adversary.create"
	commandTypeAdversaryUpdate          command.Type = "sys.daggerheart.adversary.update"
	commandTypeAdversaryDelete          command.Type = "sys.daggerheart.adversary.delete"
	eventTypeGMFearChanged              event.Type   = "sys.daggerheart.gm_fear_changed"
	eventTypeCharacterStatePatched      event.Type   = "sys.daggerheart.character_state_patched"
	eventTypeConditionChanged           event.Type   = "sys.daggerheart.condition_changed"
	eventTypeLoadoutSwapped             event.Type   = "sys.daggerheart.loadout_swapped"
	eventTypeRestTaken                  event.Type   = "sys.daggerheart.rest_taken"
	eventTypeCountdownCreated           event.Type   = "sys.daggerheart.countdown_created"
	eventTypeCountdownUpdated           event.Type   = "sys.daggerheart.countdown_updated"
	eventTypeCountdownDeleted           event.Type   = "sys.daggerheart.countdown_deleted"
	eventTypeDamageApplied              event.Type   = "sys.daggerheart.damage_applied"
	eventTypeAdversaryDamageApplied     event.Type   = "sys.daggerheart.adversary_damage_applied"
	eventTypeDowntimeMoveApplied        event.Type   = "sys.daggerheart.downtime_move_applied"
	eventTypeAdversaryConditionChanged  event.Type   = "sys.daggerheart.adversary_condition_changed"
	eventTypeAdversaryCreated           event.Type   = "sys.daggerheart.adversary_created"
	eventTypeAdversaryUpdated           event.Type   = "sys.daggerheart.adversary_updated"
	eventTypeAdversaryDeleted           event.Type   = "sys.daggerheart.adversary_deleted"

	rejectionCodeGMFearAfterRequired = "GM_FEAR_AFTER_REQUIRED"
	rejectionCodeGMFearOutOfRange    = "GM_FEAR_AFTER_OUT_OF_RANGE"
)

// Decider handles Daggerheart system commands.
type Decider struct{}

// Decide returns the decision for a system command against current state.
func (Decider) Decide(state any, cmd command.Command, now func() time.Time) command.Decision {
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
		switch current := state.(type) {
		case SnapshotState:
			before = current.GMFear
		case *SnapshotState:
			if current != nil {
				before = current.GMFear
			}
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
	case commandTypeAdversaryConditionChange:
		var payload AdversaryConditionChangePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
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
