package daggerheart

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDaggerheartAdversaryToProtoSession(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	proto := daggerheartAdversaryToProto(storage.DaggerheartAdversary{
		AdversaryID: "adv-1",
		CampaignID:  "camp-1",
		Name:        "Rival",
		SessionID:   "sess-1",
		HP:          4,
		HPMax:       6,
		Conditions:  []string{"hidden"},
		CreatedAt:   created,
		UpdatedAt:   updated,
	})
	if proto.GetSessionId().GetValue() != "sess-1" {
		t.Fatal("expected session id wrapper")
	}
	if proto.GetCreatedAt().AsTime().UTC() != created {
		t.Fatal("expected created time to map")
	}
	if len(proto.GetConditions()) != 1 {
		t.Fatalf("expected conditions to map, got %v", proto.GetConditions())
	}
}

func TestNormalizeAdversaryStatsDefaults(t *testing.T) {
	stats, err := normalizeAdversaryStats(adversaryStatsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.HPMax == 0 || stats.HP == 0 {
		t.Fatal("expected defaults to be populated")
	}
}

func TestNormalizeAdversaryStatsClampHP(t *testing.T) {
	current := storage.DaggerheartAdversary{HP: 10, HPMax: 10}
	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		HPMax:   wrapperspb.Int32(5),
		Current: &current,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.HP != 5 || stats.HPMax != 5 {
		t.Fatalf("expected hp to clamp to 5, got %d/%d", stats.HP, stats.HPMax)
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

func TestNormalizeAdversaryStatsRequireFields(t *testing.T) {
	if _, err := normalizeAdversaryStats(adversaryStatsInput{RequireFields: true}); err == nil {
		t.Fatal("expected error when required fields missing")
	}

	// RequireFields with valid HP and HPMax should succeed
	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		RequireFields: true,
		HP:            wrapperspb.Int32(4),
		HPMax:         wrapperspb.Int32(6),
	})
	if err != nil {
		t.Fatalf("unexpected error with valid required fields: %v", err)
	}
	if stats.HP != 4 || stats.HPMax != 6 {
		t.Fatalf("expected HP=4 HPMax=6, got HP=%d HPMax=%d", stats.HP, stats.HPMax)
	}
}

func TestDaggerheartAdversaryToProtoNoSession(t *testing.T) {
	proto := daggerheartAdversaryToProto(storage.DaggerheartAdversary{
		AdversaryID: "adv-2",
		CampaignID:  "camp-1",
		Name:        "Shadow",
		HP:          6,
		HPMax:       8,
		Stress:      2,
		StressMax:   4,
		Evasion:     10,
		Major:       7,
		Severe:      14,
		Armor:       2,
		Conditions:  []string{"vulnerable"},
	})
	if proto.GetSessionId() != nil {
		t.Fatal("expected nil session id wrapper when no session")
	}
	if proto.GetId() != "adv-2" || proto.GetName() != "Shadow" {
		t.Fatalf("proto metadata mismatch: %v", proto)
	}
	if proto.GetHp() != 6 || proto.GetHpMax() != 8 {
		t.Fatalf("proto HP mismatch: HP=%d HPMax=%d", proto.GetHp(), proto.GetHpMax())
	}
	if proto.GetStress() != 2 || proto.GetStressMax() != 4 {
		t.Fatalf("proto stress mismatch: Stress=%d StressMax=%d", proto.GetStress(), proto.GetStressMax())
	}
	if proto.GetEvasion() != 10 || proto.GetArmor() != 2 {
		t.Fatalf("proto evasion/armor mismatch")
	}
	if proto.GetMajorThreshold() != 7 || proto.GetSevereThreshold() != 14 {
		t.Fatalf("proto thresholds mismatch")
	}
	if len(proto.GetConditions()) != 1 {
		t.Fatalf("expected conditions to map, got %v", proto.GetConditions())
	}
}

func TestNormalizeAdversaryStatsExtended(t *testing.T) {
	// Negative HP should fail
	if _, err := normalizeAdversaryStats(adversaryStatsInput{HP: wrapperspb.Int32(-1)}); err == nil {
		t.Fatal("expected error for negative HP")
	}

	// HP > HPMax should fail
	if _, err := normalizeAdversaryStats(adversaryStatsInput{
		HP:    wrapperspb.Int32(10),
		HPMax: wrapperspb.Int32(5),
	}); err == nil {
		t.Fatal("expected error for HP > HPMax")
	}

	// Stress > StressMax should fail
	if _, err := normalizeAdversaryStats(adversaryStatsInput{
		Stress:    wrapperspb.Int32(5),
		StressMax: wrapperspb.Int32(3),
	}); err == nil {
		t.Fatal("expected error for Stress > StressMax")
	}

	// Negative Stress should fail
	if _, err := normalizeAdversaryStats(adversaryStatsInput{Stress: wrapperspb.Int32(-1)}); err == nil {
		t.Fatal("expected error for negative Stress")
	}

	// Negative evasion should fail
	if _, err := normalizeAdversaryStats(adversaryStatsInput{Evasion: wrapperspb.Int32(-1)}); err == nil {
		t.Fatal("expected error for negative evasion")
	}

	// Negative major should fail
	if _, err := normalizeAdversaryStats(adversaryStatsInput{Major: wrapperspb.Int32(-1)}); err == nil {
		t.Fatal("expected error for negative major")
	}

	// Stress clamping with current values
	current := storage.DaggerheartAdversary{HP: 6, HPMax: 6, Stress: 10, StressMax: 10, Major: 8, Severe: 12}
	stats, err := normalizeAdversaryStats(adversaryStatsInput{
		StressMax: wrapperspb.Int32(5),
		Current:   &current,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Stress != 5 || stats.StressMax != 5 {
		t.Fatalf("expected stress to clamp to 5, got %d/%d", stats.Stress, stats.StressMax)
	}

	// Explicit HP set with current values
	current2 := storage.DaggerheartAdversary{HP: 3, HPMax: 10, Major: 8, Severe: 12}
	stats, err = normalizeAdversaryStats(adversaryStatsInput{
		HP:      wrapperspb.Int32(7),
		Current: &current2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.HP != 7 || stats.HPMax != 10 {
		t.Fatalf("expected HP=7 HPMax=10, got HP=%d HPMax=%d", stats.HP, stats.HPMax)
	}

	// Explicit Stress set with current values
	current3 := storage.DaggerheartAdversary{HP: 6, HPMax: 6, Stress: 2, StressMax: 6, Major: 8, Severe: 12}
	stats, err = normalizeAdversaryStats(adversaryStatsInput{
		Stress:  wrapperspb.Int32(4),
		Current: &current3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Stress != 4 || stats.StressMax != 6 {
		t.Fatalf("expected Stress=4 StressMax=6, got Stress=%d StressMax=%d", stats.Stress, stats.StressMax)
	}

	// All fields explicitly set
	stats, err = normalizeAdversaryStats(adversaryStatsInput{
		HP:        wrapperspb.Int32(5),
		HPMax:     wrapperspb.Int32(10),
		Stress:    wrapperspb.Int32(2),
		StressMax: wrapperspb.Int32(6),
		Evasion:   wrapperspb.Int32(12),
		Major:     wrapperspb.Int32(8),
		Severe:    wrapperspb.Int32(16),
		Armor:     wrapperspb.Int32(3),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.HP != 5 || stats.HPMax != 10 || stats.Stress != 2 || stats.StressMax != 6 {
		t.Fatalf("stats mismatch: %+v", stats)
	}
	if stats.Evasion != 12 || stats.Major != 8 || stats.Severe != 16 || stats.Armor != 3 {
		t.Fatalf("extended stats mismatch: %+v", stats)
	}
}
