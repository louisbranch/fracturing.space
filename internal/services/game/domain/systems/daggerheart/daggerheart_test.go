package daggerheart

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

func TestRegistryMetadata(t *testing.T) {
	sys := systems.DefaultRegistry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	if sys == nil {
		t.Fatal("expected daggerheart system registered")
	}
	meta := sys.RegistryMetadata()
	if meta.ImplementationStage != commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL {
		t.Fatalf("ImplementationStage = %v, want PARTIAL", meta.ImplementationStage)
	}
	if meta.OperationalStatus != commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL {
		t.Fatalf("OperationalStatus = %v, want OPERATIONAL", meta.OperationalStatus)
	}
	if meta.AccessLevel != commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA {
		t.Fatalf("AccessLevel = %v, want BETA", meta.AccessLevel)
	}
	if meta.Notes != "partial support" {
		t.Fatalf("Notes = %q, want %q", meta.Notes, "partial support")
	}
}
