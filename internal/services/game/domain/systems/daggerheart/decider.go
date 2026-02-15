package daggerheart

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeGMFearSet                command.Type = "action.gm_fear.set"
	commandTypeCharacterStatePatch      command.Type = "action.character_state.patch"
	commandTypeConditionChange          command.Type = "action.condition.change"
	commandTypeHopeSpend                command.Type = "action.hope.spend"
	commandTypeStressSpend              command.Type = "action.stress.spend"
	commandTypeLoadoutSwap              command.Type = "action.loadout.swap"
	commandTypeRestTake                 command.Type = "action.rest.take"
	commandTypeAttackResolve            command.Type = "action.attack.resolve"
	commandTypeReactionResolve          command.Type = "action.reaction.resolve"
	commandTypeAdversaryRollResolve     command.Type = "action.adversary_roll.resolve"
	commandTypeAdversaryAttackResolve   command.Type = "action.adversary_attack.resolve"
	commandTypeDamageRollResolve        command.Type = "action.damage_roll.resolve"
	commandTypeGroupActionResolve       command.Type = "action.group_action.resolve"
	commandTypeTagTeamResolve           command.Type = "action.tag_team.resolve"
	commandTypeCountdownCreate          command.Type = "action.countdown.create"
	commandTypeCountdownUpdate          command.Type = "action.countdown.update"
	commandTypeCountdownDelete          command.Type = "action.countdown.delete"
	commandTypeAdversaryActionResolve   command.Type = "action.adversary_action.resolve"
	commandTypeDamageApply              command.Type = "action.damage.apply"
	commandTypeAdversaryDamageApply     command.Type = "action.adversary_damage.apply"
	commandTypeDowntimeMoveApply        command.Type = "action.downtime_move.apply"
	commandTypeDeathMoveResolve         command.Type = "action.death_move.resolve"
	commandTypeBlazeOfGloryResolve      command.Type = "action.blaze_of_glory.resolve"
	commandTypeGMMoveApply              command.Type = "action.gm_move.apply"
	commandTypeAdversaryConditionChange command.Type = "action.adversary_condition.change"
	commandTypeAdversaryCreate          command.Type = "action.adversary.create"
	commandTypeAdversaryUpdate          command.Type = "action.adversary.update"
	commandTypeAdversaryDelete          command.Type = "action.adversary.delete"
	eventTypeGMFearChanged              event.Type   = "action.gm_fear_changed"
	eventTypeCharacterStatePatched      event.Type   = "action.character_state_patched"
	eventTypeConditionChanged           event.Type   = "action.condition_changed"
	eventTypeHopeSpent                  event.Type   = "action.hope_spent"
	eventTypeStressSpent                event.Type   = "action.stress_spent"
	eventTypeLoadoutSwapped             event.Type   = "action.loadout_swapped"
	eventTypeRestTaken                  event.Type   = "action.rest_taken"
	eventTypeAttackResolved             event.Type   = "action.attack_resolved"
	eventTypeReactionResolved           event.Type   = "action.reaction_resolved"
	eventTypeAdversaryRollResolved      event.Type   = "action.adversary_roll_resolved"
	eventTypeAdversaryAttackResolved    event.Type   = "action.adversary_attack_resolved"
	eventTypeDamageRollResolved         event.Type   = "action.damage_roll_resolved"
	eventTypeGroupActionResolved        event.Type   = "action.group_action_resolved"
	eventTypeTagTeamResolved            event.Type   = "action.tag_team_resolved"
	eventTypeCountdownCreated           event.Type   = "action.countdown_created"
	eventTypeCountdownUpdated           event.Type   = "action.countdown_updated"
	eventTypeCountdownDeleted           event.Type   = "action.countdown_deleted"
	eventTypeAdversaryActionResolved    event.Type   = "action.adversary_action_resolved"
	eventTypeDamageApplied              event.Type   = "action.damage_applied"
	eventTypeAdversaryDamageApplied     event.Type   = "action.adversary_damage_applied"
	eventTypeDowntimeMoveApplied        event.Type   = "action.downtime_move_applied"
	eventTypeDeathMoveResolved          event.Type   = "action.death_move_resolved"
	eventTypeBlazeOfGloryResolved       event.Type   = "action.blaze_of_glory_resolved"
	eventTypeGMMoveApplied              event.Type   = "action.gm_move_applied"
	eventTypeAdversaryConditionChanged  event.Type   = "action.adversary_condition_changed"
	eventTypeAdversaryCreated           event.Type   = "action.adversary_created"
	eventTypeAdversaryUpdated           event.Type   = "action.adversary_updated"
	eventTypeAdversaryDeleted           event.Type   = "action.adversary_deleted"

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
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeHopeSpent,
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
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeStressSpent,
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
	case commandTypeAttackResolve:
		var payload AttackResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payload.Outcome = strings.TrimSpace(payload.Outcome)
		payload.Flavor = strings.TrimSpace(payload.Flavor)
		for i, target := range payload.Targets {
			payload.Targets[i] = strings.TrimSpace(target)
		}
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "attack"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.RequestID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAttackResolved,
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
	case commandTypeReactionResolve:
		var payload ReactionResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payload.Outcome = strings.TrimSpace(payload.Outcome)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "reaction"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.RequestID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeReactionResolved,
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
	case commandTypeAdversaryRollResolve:
		var payload AdversaryRollResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "adversary"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryRollResolved,
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
	case commandTypeAdversaryAttackResolve:
		var payload AdversaryAttackResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		for i, target := range payload.Targets {
			payload.Targets[i] = strings.TrimSpace(target)
		}
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "attack"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.RequestID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryAttackResolved,
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
	case commandTypeDamageRollResolve:
		var payload DamageRollResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "roll"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.RequestID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeDamageRollResolved,
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
	case commandTypeGroupActionResolve:
		var payload GroupActionResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.LeaderCharacterID = strings.TrimSpace(payload.LeaderCharacterID)
		for i, supporter := range payload.Supporters {
			payload.Supporters[i].CharacterID = strings.TrimSpace(supporter.CharacterID)
		}
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "group_action"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.LeaderCharacterID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeGroupActionResolved,
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
	case commandTypeTagTeamResolve:
		var payload TagTeamResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.FirstCharacterID = strings.TrimSpace(payload.FirstCharacterID)
		payload.SecondCharacterID = strings.TrimSpace(payload.SecondCharacterID)
		payload.SelectedCharacterID = strings.TrimSpace(payload.SelectedCharacterID)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "tag_team"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.SelectedCharacterID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeTagTeamResolved,
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
	case commandTypeAdversaryActionResolve:
		var payload AdversaryActionResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.AdversaryID = strings.TrimSpace(payload.AdversaryID)
		payloadJSON, _ := json.Marshal(payload)
		entityType := strings.TrimSpace(cmd.EntityType)
		if entityType == "" {
			entityType = "adversary"
		}
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.AdversaryID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeAdversaryActionResolved,
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
	case commandTypeDeathMoveResolve:
		var payload DeathMoveResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payload.Move = strings.TrimSpace(payload.Move)
		payload.LifeStateAfter = strings.TrimSpace(payload.LifeStateAfter)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeDeathMoveResolved,
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
	case commandTypeBlazeOfGloryResolve:
		var payload BlazeOfGloryResolvePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.CharacterID = strings.TrimSpace(payload.CharacterID)
		payload.LifeStateAfter = strings.TrimSpace(payload.LifeStateAfter)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = payload.CharacterID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeBlazeOfGloryResolved,
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
	case commandTypeGMMoveApply:
		var payload GMMoveApplyPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if now == nil {
			now = time.Now
		}
		payload.Move = strings.TrimSpace(payload.Move)
		payload.Description = strings.TrimSpace(payload.Description)
		payload.Severity = strings.TrimSpace(payload.Severity)
		payload.Source = strings.TrimSpace(payload.Source)
		payloadJSON, _ := json.Marshal(payload)
		entityID := strings.TrimSpace(cmd.EntityID)
		if entityID == "" {
			entityID = cmd.CampaignID
		}
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeGMMoveApplied,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "gm_move",
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
