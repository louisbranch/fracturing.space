package daggerheart

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestAdapterIdentityAndGuards(t *testing.T) {
	adapter := NewAdapter(nil)
	if adapter.ID() != SystemID {
		t.Fatalf("ID() = %q, want %q", adapter.ID(), SystemID)
	}
	if adapter.Version() != SystemVersion {
		t.Fatalf("Version() = %q, want %q", adapter.Version(), SystemVersion)
	}
	if err := adapter.Apply(context.Background(), eventForAdapterGuard()); err == nil {
		t.Fatal("expected store-not-configured error from Apply")
	}
	if _, err := adapter.Snapshot(context.Background(), "camp-1"); err == nil {
		t.Fatal("expected store-not-configured error from Snapshot")
	}

	withStore := NewAdapter(newParityDaggerheartStore())
	if _, err := withStore.Snapshot(context.Background(), " "); err == nil {
		t.Fatal("expected campaign-id-required error from Snapshot")
	}
}

func TestRegistrySystem_Contract(t *testing.T) {
	system := NewRegistrySystem()
	if system.ID() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("ID() = %v, want daggerheart enum", system.ID())
	}
	if system.Version() != SystemVersion {
		t.Fatalf("Version() = %q, want %q", system.Version(), SystemVersion)
	}
	if system.Name() != "Daggerheart" {
		t.Fatalf("Name() = %q, want Daggerheart", system.Name())
	}
	metadata := system.RegistryMetadata()
	if metadata.ImplementationStage != commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE {
		t.Fatalf("implementation stage = %v, want complete", metadata.ImplementationStage)
	}
	if metadata.OperationalStatus != commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL {
		t.Fatalf("operational status = %v, want operational", metadata.OperationalStatus)
	}
	if metadata.AccessLevel != commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA {
		t.Fatalf("access level = %v, want beta", metadata.AccessLevel)
	}
	if system.StateHandlerFactory() != nil {
		t.Fatal("expected nil state handler factory")
	}
	if system.OutcomeApplier() != nil {
		t.Fatal("expected nil outcome applier")
	}
}

func TestPublicWrappers_DelegateToMechanics(t *testing.T) {
	if _, err := NormalizeDeathMove(DeathMoveAvoidDeath); err != nil {
		t.Fatalf("NormalizeDeathMove: %v", err)
	}
	if _, err := NormalizeLifeState(LifeStateAlive); err != nil {
		t.Fatalf("NormalizeLifeState: %v", err)
	}

	deathOutcome, err := ResolveDeathMove(DeathMoveInput{
		Move:      DeathMoveBlazeOfGlory,
		Level:     1,
		HP:        1,
		HPMax:     6,
		Hope:      2,
		HopeMax:   6,
		Stress:    0,
		StressMax: 6,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove: %v", err)
	}
	if deathOutcome.LifeState != LifeStateBlazeOfGlory {
		t.Fatalf("death outcome life state = %q, want %q", deathOutcome.LifeState, LifeStateBlazeOfGlory)
	}

	restOutcome, err := ResolveRestOutcome(RestState{ConsecutiveShortRests: 0}, RestTypeShort, false, 42, 3)
	if err != nil {
		t.Fatalf("ResolveRestOutcome: %v", err)
	}
	if !restOutcome.Applied {
		t.Fatal("expected rest outcome to apply")
	}

	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          6,
		HPMax:       6,
		Hope:        1,
		HopeMax:     6,
		Stress:      2,
		StressMax:   6,
		Armor:       0,
		ArmorMax:    2,
		LifeState:   LifeStateAlive,
	})
	result := ApplyDowntimeMove(state, DowntimePrepare, DowntimeOptions{PrepareWithGroup: true})
	if result.HopeAfter <= result.HopeBefore {
		t.Fatalf("expected downtime prepare to increase hope: before=%d after=%d", result.HopeBefore, result.HopeAfter)
	}
}

func eventForAdapterGuard() event.Event {
	return event.Event{
		CampaignID:  "camp-1",
		Type:        EventTypeGMFearChanged,
		PayloadJSON: []byte(`{"before":0,"after":1}`),
	}
}
