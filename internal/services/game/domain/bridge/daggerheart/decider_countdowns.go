package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideRestTake(snapshotState SnapshotState, cmd command.Command, now func() time.Time) command.Decision {
	var payload RestTakePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	now = command.NowFunc(now)
	payload.RestType = strings.TrimSpace(payload.RestType)
	for i := range payload.CountdownUpdates {
		if rejection := countdownUpdateSnapshotRejection(snapshotState, payload.CountdownUpdates[i]); rejection != nil {
			return command.Reject(*rejection)
		}
		payload.CountdownUpdates[i].CountdownID = ids.CountdownID(strings.TrimSpace(payload.CountdownUpdates[i].CountdownID.String()))
		payload.CountdownUpdates[i].Reason = strings.TrimSpace(payload.CountdownUpdates[i].Reason)
	}

	eventPayload := RestTakenPayload{
		RestType:        payload.RestType,
		Interrupted:     payload.Interrupted,
		GMFear:          payload.GMFearAfter,
		ShortRests:      payload.ShortRestsAfter,
		RefreshRest:     payload.RefreshRest,
		RefreshLongRest: payload.RefreshLongRest,
	}
	eventPayload.Participants = append(eventPayload.Participants, payload.Participants...)
	eventPayloadJSON, _ := json.Marshal(eventPayload)
	entityID := strings.TrimSpace(cmd.EntityID)
	if entityID == "" {
		entityID = string(cmd.CampaignID)
	}
	restEvent := command.NewEvent(cmd, EventTypeRestTaken, "session", entityID, eventPayloadJSON, now().UTC())
	events := []event.Event{restEvent}

	for _, move := range payload.DowntimeMoves {
		move.ActorCharacterID = ids.CharacterID(strings.TrimSpace(move.ActorCharacterID.String()))
		move.TargetCharacterID = ids.CharacterID(strings.TrimSpace(move.TargetCharacterID.String()))
		move.CountdownID = ids.CountdownID(strings.TrimSpace(move.CountdownID.String()))
		move.GroupID = strings.TrimSpace(move.GroupID)
		move.RestType = strings.TrimSpace(move.RestType)
		move.Move = strings.TrimSpace(move.Move)
		movePayloadJSON, _ := json.Marshal(move)
		entityID := move.ActorCharacterID.String()
		if entityID == "" {
			entityID = string(cmd.CampaignID)
		}
		events = append(events, command.NewEvent(cmd, EventTypeDowntimeMoveApplied, "character", entityID, movePayloadJSON, now().UTC()))
	}

	for _, src := range payload.CountdownUpdates {
		countdownEventPayload := CountdownUpdatedPayload{
			CountdownID: src.CountdownID,
			Value:       src.After,
			Delta:       src.Delta,
			Looped:      src.Looped,
			Reason:      src.Reason,
		}
		countdownPayloadJSON, _ := json.Marshal(countdownEventPayload)
		events = append(events, command.NewEvent(cmd, EventTypeCountdownUpdated, "countdown", countdownEventPayload.CountdownID.String(), countdownPayloadJSON, now().UTC()))
	}

	return command.Accept(events...)
}

func decideCountdownCreate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeCountdownCreated, "countdown",
		func(p *CountdownCreatePayload) string { return strings.TrimSpace(p.CountdownID.String()) },
		func(p *CountdownCreatePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = ids.CountdownID(strings.TrimSpace(p.CountdownID.String()))
			p.Name = strings.TrimSpace(p.Name)
			p.Kind = strings.TrimSpace(p.Kind)
			p.Direction = strings.TrimSpace(p.Direction)
			p.Variant = strings.TrimSpace(p.Variant)
			p.TriggerEventType = strings.TrimSpace(p.TriggerEventType)
			p.LinkedCountdownID = ids.CountdownID(strings.TrimSpace(p.LinkedCountdownID.String()))
			if p.Variant == "" {
				p.Variant = "standard"
			}
			switch p.Variant {
			case "standard", "dynamic", "linked":
				// valid
			default:
				return &command.Rejection{Code: "COUNTDOWN_VARIANT_INVALID", Message: fmt.Sprintf("unknown countdown variant %q; must be standard, dynamic, or linked", p.Variant)}
			}
			if p.Variant == "dynamic" && p.TriggerEventType == "" {
				return &command.Rejection{Code: "COUNTDOWN_VARIANT_INVALID", Message: "trigger_event_type is required for dynamic countdowns"}
			}
			if p.Variant == "linked" && p.LinkedCountdownID == "" {
				return &command.Rejection{Code: "COUNTDOWN_VARIANT_INVALID", Message: "linked_countdown_id is required for linked countdowns"}
			}
			return nil
		}, now)
}

func decideCountdownUpdate(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, EventTypeCountdownUpdated, "countdown",
		func(p *CountdownUpdatePayload) string { return strings.TrimSpace(p.CountdownID.String()) },
		func(s SnapshotState, hasState bool, p *CountdownUpdatePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := countdownUpdateSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			p.CountdownID = ids.CountdownID(strings.TrimSpace(p.CountdownID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		},
		func(_ SnapshotState, _ bool, p CountdownUpdatePayload) CountdownUpdatedPayload {
			return CountdownUpdatedPayload{
				CountdownID: p.CountdownID,
				Value:       p.After,
				Delta:       p.Delta,
				Looped:      p.Looped,
				Reason:      p.Reason,
			}
		},
		now)
}

func decideCountdownDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeCountdownDeleted, "countdown",
		func(p *CountdownDeletePayload) string { return strings.TrimSpace(p.CountdownID.String()) },
		func(p *CountdownDeletePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = ids.CountdownID(strings.TrimSpace(p.CountdownID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
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
		return &command.Rejection{
			Code:    rejectionCodeCountdownUpdateNoMutation,
			Message: "countdown update is unchanged",
		}
	}
	return nil
}

func snapshotCountdownState(snapshot SnapshotState, countdownID ids.CountdownID) (CountdownState, bool) {
	trimmed := ids.CountdownID(strings.TrimSpace(countdownID.String()))
	if trimmed == "" {
		return CountdownState{}, false
	}
	countdown, ok := snapshot.CountdownStates[trimmed]
	if !ok {
		return CountdownState{}, false
	}
	countdown.CountdownID = trimmed
	countdown.CampaignID = snapshot.CampaignID
	return countdown, true
}
