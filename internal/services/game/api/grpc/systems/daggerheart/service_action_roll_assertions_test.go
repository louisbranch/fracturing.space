package daggerheart

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/domain"
)

// assertResponseMatches validates response fields against expectations.
func assertResponseMatches(t *testing.T, response *pb.ActionRollResponse, seed int64, seedSource string, rollMode commonv1.RollMode, modifier int32, difficulty *int32) {
	t.Helper()

	if response == nil {
		t.Fatal("ActionRoll response is nil")
	}
	if response.GetRng() == nil {
		t.Fatal("ActionRoll rng is nil")
	}
	if response.GetRng().GetSeedUsed() != uint64(seed) {
		t.Fatalf("ActionRoll seed_used = %d, want %d", response.GetRng().GetSeedUsed(), seed)
	}
	if response.GetRng().GetRngAlgo() != random.RngAlgoMathRandV1 {
		t.Fatalf("ActionRoll rng_algo = %q, want %q", response.GetRng().GetRngAlgo(), random.RngAlgoMathRandV1)
	}
	if response.GetRng().GetSeedSource() != seedSource {
		t.Fatalf("ActionRoll seed_source = %q, want %q", response.GetRng().GetSeedSource(), seedSource)
	}
	if response.GetRng().GetRollMode() != rollMode {
		t.Fatalf("ActionRoll roll_mode = %v, want %v", response.GetRng().GetRollMode(), rollMode)
	}

	result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
		Modifier:   int(modifier),
		Difficulty: intPointer(difficulty),
		Seed:       seed,
	})
	if err != nil {
		t.Fatalf("RollAction returned error: %v", err)
	}

	if response.GetHope() != int32(result.Hope) || response.GetFear() != int32(result.Fear) {
		t.Fatalf("ActionRoll dice = (%d, %d), want (%d, %d)", response.GetHope(), response.GetFear(), result.Hope, result.Fear)
	}
	if response.GetModifier() != int32(result.Modifier) {
		t.Fatalf("ActionRoll modifier = %d, want %d", response.GetModifier(), result.Modifier)
	}
	if response.GetAdvantageDie() != int32(result.AdvantageDie) {
		t.Fatalf("ActionRoll advantage_die = %d, want %d", response.GetAdvantageDie(), result.AdvantageDie)
	}
	if response.GetAdvantageModifier() != int32(result.AdvantageModifier) {
		t.Fatalf("ActionRoll advantage_modifier = %d, want %d", response.GetAdvantageModifier(), result.AdvantageModifier)
	}
	if response.Total != int32(result.Total) {
		t.Fatalf("ActionRoll total = %d, want %d", response.Total, result.Total)
	}
	if response.IsCrit != result.IsCrit {
		t.Fatalf("ActionRoll is_crit = %t, want %t", response.IsCrit, result.IsCrit)
	}
	if response.MeetsDifficulty != result.MeetsDifficulty {
		t.Fatalf("ActionRoll meets_difficulty = %t, want %t", response.MeetsDifficulty, result.MeetsDifficulty)
	}
	if response.Outcome != wantOutcomeProto(result.Outcome) {
		t.Fatalf("ActionRoll outcome = %v, want %v", response.Outcome, wantOutcomeProto(result.Outcome))
	}
	if difficulty != nil && response.Difficulty == nil {
		t.Fatal("ActionRoll difficulty is nil, want value")
	}
	if difficulty != nil && response.Difficulty != nil && *response.Difficulty != *difficulty {
		t.Fatalf("ActionRoll difficulty = %d, want %d", *response.Difficulty, *difficulty)
	}
}
