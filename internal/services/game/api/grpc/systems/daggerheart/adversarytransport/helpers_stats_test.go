package adversarytransport

import (
	"testing"

	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNormalizeAdversaryStatsDefaults(t *testing.T) {
	stats, err := normalizeAdversaryStats(adversaryStatsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Evasion != daggerheartprofile.AdversaryDefaultEvasion ||
		stats.Major != daggerheartprofile.AdversaryDefaultMajor ||
		stats.Severe != daggerheartprofile.AdversaryDefaultSevere {
		t.Fatalf("unexpected defaults: %+v", stats)
	}
	if stats.HPMax == 0 || stats.HP == 0 {
		t.Fatal("expected defaults to be populated")
	}
}

func TestNormalizeAdversaryStatsValidation(t *testing.T) {
	if _, err := normalizeAdversaryStats(adversaryStatsInput{HPMax: wrapperspb.Int32(0)}); err == nil {
		t.Fatal("expected error for invalid hp_max")
	}
	if _, err := normalizeAdversaryStats(adversaryStatsInput{StressMax: wrapperspb.Int32(-1)}); err == nil {
		t.Fatal("expected error for invalid stress_max")
	}
	if _, err := normalizeAdversaryStats(adversaryStatsInput{Major: wrapperspb.Int32(5), Severe: wrapperspb.Int32(3)}); err == nil {
		t.Fatal("expected error for severe < major")
	}
	if _, err := normalizeAdversaryStats(adversaryStatsInput{Armor: wrapperspb.Int32(-1)}); err == nil {
		t.Fatal("expected error for invalid armor")
	}
}

func TestNormalizeAdversaryStatsCurrentClamp(t *testing.T) {
	current := projectionstore.DaggerheartAdversary{HP: 10, HPMax: 10, Stress: 5, StressMax: 5}
	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		HPMax:     wrapperspb.Int32(5),
		StressMax: wrapperspb.Int32(2),
		Current:   &current,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.HP != 5 || stats.HPMax != 5 {
		t.Fatalf("expected hp to clamp to 5, got %d/%d", stats.HP, stats.HPMax)
	}
	if stats.Stress != 2 || stats.StressMax != 2 {
		t.Fatalf("expected stress to clamp to 2, got %d/%d", stats.Stress, stats.StressMax)
	}
}
