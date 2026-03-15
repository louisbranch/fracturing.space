package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
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
	if payload.LongTermCountdown != nil {
		if rejection := countdownUpdateSnapshotRejection(snapshotState, *payload.LongTermCountdown); rejection != nil {
			return command.Reject(*rejection)
		}
		payload.LongTermCountdown.CountdownID = ids.CountdownID(strings.TrimSpace(payload.LongTermCountdown.CountdownID.String()))
		payload.LongTermCountdown.Reason = strings.TrimSpace(payload.LongTermCountdown.Reason)
	}
	// Build event payload, stripping Before fields and LongTermCountdown.
	eventPayload := RestTakenPayload{
		RestType:        payload.RestType,
		Interrupted:     payload.Interrupted,
		GMFear:          payload.GMFearAfter,
		ShortRests:      payload.ShortRestsAfter,
		RefreshRest:     payload.RefreshRest,
		RefreshLongRest: payload.RefreshLongRest,
	}
	for _, patch := range payload.CharacterStates {
		eventPayload.CharacterStates = append(eventPayload.CharacterStates, RestTakenCharacterPatch{
			CharacterID: patch.CharacterID,
			Hope:        patch.HopeAfter,
			Stress:      patch.StressAfter,
			Armor:       patch.ArmorAfter,
		})
	}
	eventPayloadJSON, _ := json.Marshal(eventPayload)
	entityID := strings.TrimSpace(cmd.EntityID)
	if entityID == "" {
		entityID = string(cmd.CampaignID)
	}
	restEvent := command.NewEvent(cmd, EventTypeRestTaken, "session", entityID, eventPayloadJSON, now().UTC())

	if payload.LongTermCountdown == nil {
		return command.Accept(restEvent)
	}
	src := *payload.LongTermCountdown
	countdownEventPayload := CountdownUpdatedPayload{
		CountdownID: src.CountdownID,
		Value:       src.After,
		Delta:       src.Delta,
		Looped:      src.Looped,
		Reason:      src.Reason,
	}
	countdownPayloadJSON, _ := json.Marshal(countdownEventPayload)
	countdownEvent := command.NewEvent(cmd, EventTypeCountdownUpdated, "countdown", countdownEventPayload.CountdownID.String(), countdownPayloadJSON, now().UTC())
	return command.Accept(restEvent, countdownEvent)
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
