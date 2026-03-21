package decider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

func decideRestTake(snapshotState snapstate.SnapshotState, cmd command.Command, now func() time.Time) command.Decision {
	var p payload.RestTakePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &p); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	now = command.NowFunc(now)
	p.RestType = strings.TrimSpace(p.RestType)
	for i := range p.CountdownUpdates {
		if rejection := countdownUpdateSnapshotRejection(snapshotState, p.CountdownUpdates[i]); rejection != nil {
			return command.Reject(*rejection)
		}
		p.CountdownUpdates[i].CountdownID = ids.CountdownID(strings.TrimSpace(p.CountdownUpdates[i].CountdownID.String()))
		p.CountdownUpdates[i].Reason = strings.TrimSpace(p.CountdownUpdates[i].Reason)
	}

	eventPayload := payload.RestTakenPayload{
		RestType:        p.RestType,
		Interrupted:     p.Interrupted,
		GMFear:          p.GMFearAfter,
		ShortRests:      p.ShortRestsAfter,
		RefreshRest:     p.RefreshRest,
		RefreshLongRest: p.RefreshLongRest,
	}
	eventPayload.Participants = append(eventPayload.Participants, p.Participants...)
	eventPayloadJSON, _ := json.Marshal(eventPayload)
	entityID := strings.TrimSpace(cmd.EntityID)
	if entityID == "" {
		entityID = string(cmd.CampaignID)
	}
	restEvent := command.NewEvent(cmd, payload.EventTypeRestTaken, "session", entityID, eventPayloadJSON, now().UTC())
	events := []event.Event{restEvent}

	for _, move := range p.DowntimeMoves {
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
		events = append(events, command.NewEvent(cmd, payload.EventTypeDowntimeMoveApplied, "character", entityID, movePayloadJSON, now().UTC()))
	}

	for _, src := range p.CountdownUpdates {
		countdownEventPayload := payload.CountdownUpdatedPayload{
			CountdownID: src.CountdownID,
			Value:       src.After,
			Delta:       src.Delta,
			Looped:      src.Looped,
			Reason:      src.Reason,
		}
		countdownPayloadJSON, _ := json.Marshal(countdownEventPayload)
		events = append(events, command.NewEvent(cmd, payload.EventTypeCountdownUpdated, "countdown", countdownEventPayload.CountdownID.String(), countdownPayloadJSON, now().UTC()))
	}

	return command.Accept(events...)
}

func decideCountdownCreate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeCountdownCreated, "countdown",
		func(p *payload.CountdownCreatePayload) string { return strings.TrimSpace(p.CountdownID.String()) },
		func(p *payload.CountdownCreatePayload, _ func() time.Time) *command.Rejection {
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

func decideCountdownUpdate(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, payload.EventTypeCountdownUpdated, "countdown",
		func(p *payload.CountdownUpdatePayload) string { return strings.TrimSpace(p.CountdownID.String()) },
		func(s snapstate.SnapshotState, hasState bool, p *payload.CountdownUpdatePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := countdownUpdateSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			p.CountdownID = ids.CountdownID(strings.TrimSpace(p.CountdownID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		},
		func(_ snapstate.SnapshotState, _ bool, p payload.CountdownUpdatePayload) payload.CountdownUpdatedPayload {
			return payload.CountdownUpdatedPayload{
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
	return module.DecideFunc(cmd, payload.EventTypeCountdownDeleted, "countdown",
		func(p *payload.CountdownDeletePayload) string { return strings.TrimSpace(p.CountdownID.String()) },
		func(p *payload.CountdownDeletePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = ids.CountdownID(strings.TrimSpace(p.CountdownID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isCountdownUpdateNoMutation(snapshot snapstate.SnapshotState, p payload.CountdownUpdatePayload) bool {
	countdown, hasCountdown := snapshotCountdownState(snapshot, p.CountdownID)
	if !hasCountdown {
		return false
	}
	if countdown.Current != p.After {
		return false
	}
	if p.Looped && !countdown.Looping {
		return false
	}
	return true
}

func countdownUpdateSnapshotRejection(snapshot snapstate.SnapshotState, p payload.CountdownUpdatePayload) *command.Rejection {
	if countdown, hasCountdown := snapshotCountdownState(snapshot, p.CountdownID); hasCountdown && p.Before != countdown.Current {
		return &command.Rejection{
			Code:    rejectionCodeCountdownBeforeMismatch,
			Message: "countdown before does not match current state",
		}
	}
	if isCountdownUpdateNoMutation(snapshot, p) {
		return &command.Rejection{
			Code:    rejectionCodeCountdownUpdateNoMutation,
			Message: "countdown update is unchanged",
		}
	}
	return nil
}

func snapshotCountdownState(snapshot snapstate.SnapshotState, countdownID ids.CountdownID) (snapstate.CountdownState, bool) {
	trimmed := ids.CountdownID(strings.TrimSpace(countdownID.String()))
	if trimmed == "" {
		return snapstate.CountdownState{}, false
	}
	countdown, ok := snapshot.CountdownStates[trimmed]
	if !ok {
		return snapstate.CountdownState{}, false
	}
	countdown.CountdownID = trimmed
	countdown.CampaignID = snapshot.CampaignID
	return countdown, true
}
