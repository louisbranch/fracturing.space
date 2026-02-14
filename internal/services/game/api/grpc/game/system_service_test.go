package game

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"google.golang.org/grpc/codes"
)

type testRegistrySystem struct {
	id       commonv1.GameSystem
	version  string
	name     string
	metadata systems.RegistryMetadata
}

func (t *testRegistrySystem) ID() commonv1.GameSystem {
	return t.id
}

func (t *testRegistrySystem) Version() string {
	return t.version
}

func (t *testRegistrySystem) Name() string {
	return t.name
}

func (t *testRegistrySystem) RegistryMetadata() systems.RegistryMetadata {
	return t.metadata
}

func (t *testRegistrySystem) StateFactory() systems.StateFactory {
	return nil
}

func (t *testRegistrySystem) OutcomeApplier() systems.OutcomeApplier {
	return nil
}

func TestListGameSystems_Defaults(t *testing.T) {
	registry := systems.NewRegistry()
	registry.Register(&testRegistrySystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.0.0",
		name:    "Daggerheart",
		metadata: systems.RegistryMetadata{
			ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
			OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
			Notes:               "partial support",
		},
	})
	registry.Register(&testRegistrySystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.1.0",
		name:    "Daggerheart",
		metadata: systems.RegistryMetadata{
			ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
			OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
			Notes:               "partial support",
		},
	})

	svc := NewSystemService(registry)
	resp, err := svc.ListGameSystems(context.Background(), &gamev1.ListGameSystemsRequest{})
	if err != nil {
		t.Fatalf("ListGameSystems returned error: %v", err)
	}
	if len(resp.Systems) != 2 {
		t.Fatalf("ListGameSystems returned %d systems, want 2", len(resp.Systems))
	}
	if !resp.Systems[0].IsDefault {
		t.Fatal("expected default system to be marked")
	}
	if resp.Systems[0].ImplementationStage != commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL {
		t.Fatalf("ImplementationStage = %v, want PARTIAL", resp.Systems[0].ImplementationStage)
	}
	if resp.Systems[0].OperationalStatus != commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL {
		t.Fatalf("OperationalStatus = %v, want OPERATIONAL", resp.Systems[0].OperationalStatus)
	}
	if resp.Systems[0].AccessLevel != commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA {
		t.Fatalf("AccessLevel = %v, want BETA", resp.Systems[0].AccessLevel)
	}
}

func TestGetGameSystem_DefaultVersion(t *testing.T) {
	registry := systems.NewRegistry()
	registry.Register(&testRegistrySystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.0.0",
		name:    "Daggerheart",
		metadata: systems.RegistryMetadata{
			ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
			OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
		},
	})
	registry.Register(&testRegistrySystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.1.0",
		name:    "Daggerheart",
		metadata: systems.RegistryMetadata{
			ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
			OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
		},
	})

	svc := NewSystemService(registry)
	resp, err := svc.GetGameSystem(context.Background(), &gamev1.GetGameSystemRequest{Id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART})
	if err != nil {
		t.Fatalf("GetGameSystem returned error: %v", err)
	}
	if resp.System == nil {
		t.Fatal("GetGameSystem response has nil system")
	}
	if resp.System.Version != "1.0.0" {
		t.Fatalf("GetGameSystem version = %q, want %q", resp.System.Version, "1.0.0")
	}
}

func TestGetGameSystem_NotFound(t *testing.T) {
	registry := systems.NewRegistry()
	svc := NewSystemService(registry)
	_, err := svc.GetGameSystem(context.Background(), &gamev1.GetGameSystemRequest{Id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART})
	assertStatusCode(t, err, codes.NotFound)
}
