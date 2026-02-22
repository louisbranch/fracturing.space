package commandbuild

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestCoreSystem(t *testing.T) {
	cmd := CoreSystem(CoreSystemInput{
		CampaignID:   "camp-1",
		Type:         command.Type("action.roll.resolve"),
		SessionID:    "sess-1",
		RequestID:    "req-1",
		InvocationID: "inv-1",
		EntityType:   "roll",
		EntityID:     "req-1",
		PayloadJSON:  []byte(`{"ok":true}`),
	})

	if cmd.ActorType != command.ActorTypeSystem {
		t.Fatalf("actor type = %q, want %q", cmd.ActorType, command.ActorTypeSystem)
	}
	if cmd.Type != command.Type("action.roll.resolve") {
		t.Fatalf("type = %q", cmd.Type)
	}
	if cmd.SystemID != "" || cmd.SystemVersion != "" {
		t.Fatalf("expected empty system metadata for core command")
	}
}

func TestDaggerheartSystemCommand(t *testing.T) {
	cmd := DaggerheartSystemCommand(DaggerheartSystemCommandInput{
		CampaignID:   "camp-1",
		Type:         command.Type("sys.daggerheart.gm_fear.set"),
		SessionID:    "sess-1",
		RequestID:    "req-1",
		InvocationID: "inv-1",
		EntityType:   "campaign",
		EntityID:     "camp-1",
		PayloadJSON:  []byte(`{"after":3}`),
	})

	if cmd.ActorType != command.ActorTypeSystem {
		t.Fatalf("actor type = %q, want %q", cmd.ActorType, command.ActorTypeSystem)
	}
	if cmd.SystemID != daggerheart.SystemID {
		t.Fatalf("system id = %q, want %q", cmd.SystemID, daggerheart.SystemID)
	}
	if cmd.SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("system version = %q, want %q", cmd.SystemVersion, daggerheart.SystemVersion)
	}
}
