package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideEnvironmentEntityCreate(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, payload.EventTypeEnvironmentEntityCreated, "environment_entity",
		func(p *payload.EnvironmentEntityCreatePayload) string {
			return normalize.ID(p.EnvironmentEntityID).String()
		},
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.EnvironmentEntityCreatePayload, _ func() time.Time) *command.Rejection {
			if hasState && isEnvironmentEntityCreateNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeEnvironmentEntityCreateNoMutation,
					Message: "environment entity create is unchanged",
				}
			}
			p.EnvironmentEntityID = normalize.ID(p.EnvironmentEntityID)
			p.EnvironmentID = normalize.String(p.EnvironmentID)
			p.Name = normalize.String(p.Name)
			p.Type = normalize.String(p.Type)
			p.SessionID = normalize.ID(p.SessionID)
			p.SceneID = normalize.ID(p.SceneID)
			p.Notes = normalize.String(p.Notes)
			return nil
		}, now)
}

func decideEnvironmentEntityUpdate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeEnvironmentEntityUpdated, "environment_entity",
		func(p *payload.EnvironmentEntityUpdatePayload) string {
			return normalize.ID(p.EnvironmentEntityID).String()
		},
		func(p *payload.EnvironmentEntityUpdatePayload, _ func() time.Time) *command.Rejection {
			p.EnvironmentEntityID = normalize.ID(p.EnvironmentEntityID)
			p.EnvironmentID = normalize.String(p.EnvironmentID)
			p.Name = normalize.String(p.Name)
			p.Type = normalize.String(p.Type)
			p.SessionID = normalize.ID(p.SessionID)
			p.SceneID = normalize.ID(p.SceneID)
			p.Notes = normalize.String(p.Notes)
			return nil
		}, now)
}

func decideEnvironmentEntityDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeEnvironmentEntityDeleted, "environment_entity",
		func(p *payload.EnvironmentEntityDeletePayload) string {
			return normalize.ID(p.EnvironmentEntityID).String()
		},
		func(p *payload.EnvironmentEntityDeletePayload, _ func() time.Time) *command.Rejection {
			p.EnvironmentEntityID = normalize.ID(p.EnvironmentEntityID)
			p.Reason = normalize.String(p.Reason)
			return nil
		}, now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isEnvironmentEntityCreateNoMutation(snapshot daggerheartstate.SnapshotState, p payload.EnvironmentEntityCreatePayload) bool {
	environmentEntity, hasEnvironmentEntity := snapshotEnvironmentEntityState(snapshot, p.EnvironmentEntityID)
	if !hasEnvironmentEntity {
		return false
	}
	return environmentEntity.EnvironmentID == normalize.String(p.EnvironmentID) &&
		environmentEntity.Name == normalize.String(p.Name) &&
		environmentEntity.Type == normalize.String(p.Type) &&
		environmentEntity.Tier == p.Tier &&
		environmentEntity.Difficulty == p.Difficulty &&
		environmentEntity.SessionID == normalize.ID(p.SessionID) &&
		environmentEntity.SceneID == normalize.ID(p.SceneID) &&
		environmentEntity.Notes == normalize.String(p.Notes)
}

func snapshotEnvironmentEntityState(snapshot daggerheartstate.SnapshotState, environmentEntityID ids.EnvironmentEntityID) (daggerheartstate.EnvironmentEntityState, bool) {
	trimmed, ok := normalize.RequireID(environmentEntityID)
	if !ok {
		return daggerheartstate.EnvironmentEntityState{}, false
	}
	environmentEntity, found := snapshot.EnvironmentStates[trimmed]
	if !found {
		return daggerheartstate.EnvironmentEntityState{}, false
	}
	environmentEntity.EnvironmentEntityID = trimmed
	environmentEntity.CampaignID = snapshot.CampaignID
	return environmentEntity, true
}
