package generator

import (
	"math/rand"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestNewSeededRNGDeterministic(t *testing.T) {
	first := NewSeededRNG(42, false)
	second := NewSeededRNG(42, false)

	if first.Int63() != second.Int63() {
		t.Fatal("expected deterministic RNG for same seed")
	}
	if first.Int63() != second.Int63() {
		t.Fatal("expected deterministic RNG sequence for same seed")
	}
}

func TestGeneratorRandomRangeMinGreaterThanMax(t *testing.T) {
	gen := &Generator{rng: rand.New(rand.NewSource(1))}
	if got := gen.randomRange(5, 3); got != 5 {
		t.Fatalf("expected min when min >= max, got %d", got)
	}
}

func TestGeneratorRandomRangeInclusive(t *testing.T) {
	gen := &Generator{rng: rand.New(rand.NewSource(2))}
	for i := 0; i < 10; i++ {
		value := gen.randomRange(2, 4)
		if value < 2 || value > 4 {
			t.Fatalf("value %d out of range", value)
		}
	}
}

func TestGeneratorGameSystem(t *testing.T) {
	gen := &Generator{rng: rand.New(rand.NewSource(3))}
	if got := gen.gameSystem(); got != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected daggerheart system, got %v", got)
	}
}
