package decider

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideRestTake(snapshotState daggerheartstate.SnapshotState, cmd command.Command, now func() time.Time) command.Decision {
	var p payload.RestTakePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &p); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	now = command.NowFunc(now)
	p.RestType = normalize.String(p.RestType)
	for i := range p.CountdownUpdates {
		if rejection := countdownUpdateSnapshotRejection(snapshotState, p.CountdownUpdates[i]); rejection != nil {
			return command.Reject(*rejection)
		}
		p.CountdownUpdates[i].CountdownID = normalize.ID(p.CountdownUpdates[i].CountdownID)
		p.CountdownUpdates[i].Reason = normalize.String(p.CountdownUpdates[i].Reason)
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
	entityID := normalize.String(cmd.EntityID)
	if entityID == "" {
		entityID = string(cmd.CampaignID)
	}
	restEvent := command.NewEvent(cmd, payload.EventTypeRestTaken, "session", entityID, eventPayloadJSON, now().UTC())
	events := []event.Event{restEvent}

	for _, move := range p.DowntimeMoves {
		move.ActorCharacterID = normalize.ID(move.ActorCharacterID)
		move.TargetCharacterID = normalize.ID(move.TargetCharacterID)
		move.CountdownID = normalize.ID(move.CountdownID)
		move.GroupID = normalize.String(move.GroupID)
		move.RestType = normalize.String(move.RestType)
		move.Move = normalize.String(move.Move)
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
		func(p *payload.CountdownCreatePayload) string { return normalize.ID(p.CountdownID).String() },
		func(p *payload.CountdownCreatePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = normalize.ID(p.CountdownID)
			p.Name = normalize.String(p.Name)
			p.Kind = normalize.String(p.Kind)
			p.Direction = normalize.String(p.Direction)
			p.Variant = normalize.String(p.Variant)
			p.TriggerEventType = normalize.String(p.TriggerEventType)
			p.LinkedCountdownID = normalize.ID(p.LinkedCountdownID)
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

func decideCountdownUpdate(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, payload.EventTypeCountdownUpdated, "countdown",
		func(p *payload.CountdownUpdatePayload) string { return normalize.ID(p.CountdownID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.CountdownUpdatePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := countdownUpdateSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			p.CountdownID = normalize.ID(p.CountdownID)
			p.Reason = normalize.String(p.Reason)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.CountdownUpdatePayload) payload.CountdownUpdatedPayload {
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
		func(p *payload.CountdownDeletePayload) string { return normalize.ID(p.CountdownID).String() },
		func(p *payload.CountdownDeletePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = normalize.ID(p.CountdownID)
			p.Reason = normalize.String(p.Reason)
			return nil
		}, now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isCountdownUpdateNoMutation(snapshot daggerheartstate.SnapshotState, p payload.CountdownUpdatePayload) bool {
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

func countdownUpdateSnapshotRejection(snapshot daggerheartstate.SnapshotState, p payload.CountdownUpdatePayload) *command.Rejection {
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

func snapshotCountdownState(snapshot daggerheartstate.SnapshotState, countdownID dhids.CountdownID) (daggerheartstate.CountdownState, bool) {
	trimmed := normalize.ID(countdownID)
	if trimmed == "" {
		return daggerheartstate.CountdownState{}, false
	}
	countdown, ok := snapshot.CountdownStates[trimmed]
	if !ok {
		return daggerheartstate.CountdownState{}, false
	}
	countdown.CountdownID = trimmed
	countdown.CampaignID = snapshot.CampaignID
	return countdown, true
}
