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
		CreatedAt:   created,
		UpdatedAt:   updated,
	})
	if proto.GetSessionId().GetValue() != "sess-1" {
		t.Fatal("expected session id wrapper")
	}
	if proto.GetCreatedAt().AsTime().UTC() != created {
		t.Fatal("expected created time to map")
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
}
