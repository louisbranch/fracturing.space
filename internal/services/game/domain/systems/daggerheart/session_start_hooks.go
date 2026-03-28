package daggerheart

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

type characterReadinessEvaluator struct {
	snapshot daggerheartstate.SnapshotState
}

func (e characterReadinessEvaluator) CharacterReady(ch character.State) (bool, string) {
	profile, ok := e.snapshot.CharacterProfiles[ch.CharacterID]
	if !ok {
		return false, "daggerheart profile is missing"
	}
	return EvaluateCreationReadiness(profile)
}

type sessionStartBootstrapEmitter struct {
	snapshot daggerheartstate.SnapshotState
}

func (e sessionStartBootstrapEmitter) EmitSessionStartBootstrap(
	characters map[ids.CharacterID]character.State,
	cmd command.Command,
	now time.Time,
) ([]event.Event, error) {
	if e.snapshot.GMFear != daggerheartstate.GMFearDefault {
		return nil, nil
	}

	pcCount := 0
	for _, ch := range characters {
		if !ch.Created || ch.Deleted || ch.Kind != character.KindPC {
			continue
		}
		pcCount++
	}
	if pcCount == daggerheartstate.GMFearDefault {
		return nil, nil
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{
		Value:  pcCount,
		Reason: "campaign_start",
	})
	if err != nil {
		return nil, err
	}
	return []event.Event{{
		CampaignID:    cmd.CampaignID,
		Type:          daggerheartpayload.EventTypeGMFearChanged,
		Timestamp:     now.UTC(),
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		SceneID:       cmd.SceneID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    "campaign",
		EntityID:      string(cmd.CampaignID),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		PayloadJSON:   payloadJSON,
	}}, nil
}

func bindCharacterReadiness(
	factory module.StateFactory,
	campaignID ids.CampaignID,
	currentByKey map[module.Key]any,
) (module.CharacterReadinessEvaluator, error) {
	snapshot, err := bindSessionStartSnapshot(factory, campaignID, currentByKey)
	if err != nil {
		return nil, err
	}
	return characterReadinessEvaluator{snapshot: snapshot}, nil
}

func bindSessionStartBootstrap(
	factory module.StateFactory,
	campaignID ids.CampaignID,
	currentByKey map[module.Key]any,
) (module.SessionStartBootstrapEmitter, error) {
	snapshot, err := bindSessionStartSnapshot(factory, campaignID, currentByKey)
	if err != nil {
		return nil, err
	}
	return sessionStartBootstrapEmitter{snapshot: snapshot}, nil
}

func bindSessionStartSnapshot(
	factory module.StateFactory,
	campaignID ids.CampaignID,
	currentByKey map[module.Key]any,
) (daggerheartstate.SnapshotState, error) {
	current := currentByKey[module.Key{ID: SystemID, Version: SystemVersion}]
	if current == nil {
		if factory == nil {
			return daggerheartstate.SnapshotState{}, fmt.Errorf("daggerheart state factory is not configured")
		}
		seeded, err := factory.NewSnapshotState(campaignID)
		if err != nil {
			return daggerheartstate.SnapshotState{}, fmt.Errorf("daggerheart state factory NewSnapshotState: %w", err)
		}
		current = seeded
	}
	snapshot, err := daggerheartstate.SnapshotOrDefaultIfAbsent(current)
	if err != nil {
		return daggerheartstate.SnapshotState{}, fmt.Errorf("daggerheart state is invalid: %w", err)
	}
	return snapshot, nil
}
