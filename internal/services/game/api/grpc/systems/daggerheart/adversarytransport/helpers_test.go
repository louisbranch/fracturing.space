package adversarytransport

import (
	"context"
	"testing"
	"time"

	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestAdversaryToProtoSession(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	proto := AdversaryToProto(projectionstore.DaggerheartAdversary{
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

func TestAdversaryToProtoNoSession(t *testing.T) {
	proto := AdversaryToProto(projectionstore.DaggerheartAdversary{
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
}

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

func TestLoadAdversaryForSession(t *testing.T) {
	store := &testDaggerheartStore{
		adversaries: map[string]projectionstore.DaggerheartAdversary{
			"adv-1": {CampaignID: "camp-1", AdversaryID: "adv-1", SessionID: "sess-1"},
		},
	}
	adversary, err := LoadAdversaryForSession(context.Background(), store, "camp-1", "sess-1", "adv-1")
	if err != nil {
		t.Fatalf("LoadAdversaryForSession returned error: %v", err)
	}
	if adversary.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", adversary.AdversaryID)
	}
	if _, err := LoadAdversaryForSession(context.Background(), store, "camp-1", "other", "adv-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	store.err = storage.ErrNotFound
	if _, err := LoadAdversaryForSession(context.Background(), store, "camp-1", "sess-1", "missing"); status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}
