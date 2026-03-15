package daggerheart

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// dynamicDomainEngine appends accepted adversary write-path events to the fake event store.
type dynamicDomainEngine struct {
	store       storage.EventStore
	calls       int
	lastCommand command.Command
}

func (d *dynamicDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	d.calls++
	d.lastCommand = cmd

	var eventType event.Type
	switch cmd.Type {
	case command.Type("sys.daggerheart.adversary.create"):
		eventType = event.Type("sys.daggerheart.adversary_created")
	case command.Type("sys.daggerheart.adversary.update"):
		eventType = event.Type("sys.daggerheart.adversary_updated")
	case command.Type("sys.daggerheart.adversary.delete"):
		eventType = event.Type("sys.daggerheart.adversary_deleted")
	case command.Type("sys.daggerheart.adversary_damage.apply"):
		eventType = event.Type("sys.daggerheart.adversary_damage_applied")
	case command.Type("sys.daggerheart.adversary_condition.change"):
		eventType = event.Type("sys.daggerheart.adversary_condition_changed")
	default:
		return engine.Result{}, nil
	}

	entityID := strings.TrimSpace(cmd.EntityID)
	if entityID == "" {
		var payload struct {
			AdversaryID string `json:"adversary_id"`
		}
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		entityID = strings.TrimSpace(payload.AdversaryID)
	}

	evt := event.Event{
		CampaignID:    cmd.CampaignID,
		Type:          eventType,
		Timestamp:     time.Now().UTC(),
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    "adversary",
		EntityID:      entityID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   cmd.PayloadJSON,
	}

	result := engine.Result{Decision: command.Accept(evt)}
	if d.store == nil || len(result.Decision.Events) == 0 {
		return result, nil
	}

	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := d.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored

	return result, nil
}
