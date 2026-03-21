package charactermutationtransport

import (
	"testing"

	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireDaggerheartSystemf(t *testing.T) {
	record := storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}
	if err := daggerheartguard.RequireDaggerheartSystemf(record, "unsupported %s", "operation"); err != nil {
		t.Fatalf("RequireDaggerheartSystemf returned error: %v", err)
	}

	err := daggerheartguard.RequireDaggerheartSystemf(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported %s", "operation")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestTierForLevel(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{level: 0, want: 1},
		{level: 1, want: 1},
		{level: 2, want: 2},
		{level: 4, want: 2},
		{level: 5, want: 3},
		{level: 7, want: 3},
		{level: 8, want: 4},
	}

	for _, tt := range tests {
		if got := tierForLevel(tt.level); got != tt.want {
			t.Fatalf("tierForLevel(%d) = %d, want %d", tt.level, got, tt.want)
		}
	}
}
