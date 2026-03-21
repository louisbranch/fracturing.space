package rules

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestAdversarySpotlightCap(t *testing.T) {
	t.Run("defaults to one", func(t *testing.T) {
		entry := contentstore.DaggerheartAdversaryEntry{}
		if got := AdversarySpotlightCap(entry); got != AdversaryDefaultSpotlightCap {
			t.Fatalf("spotlight cap = %d, want %d", got, AdversaryDefaultSpotlightCap)
		}
	})

	t.Run("uses relentless override", func(t *testing.T) {
		entry := contentstore.DaggerheartAdversaryEntry{
			RelentlessRule: &contentstore.DaggerheartAdversaryRelentlessRule{
				MaxSpotlightsPerGMTurn: 3,
			},
		}
		if got := AdversarySpotlightCap(entry); got != 3 {
			t.Fatalf("spotlight cap = %d, want 3", got)
		}
	})

	t.Run("ignores non-positive relentless values", func(t *testing.T) {
		entry := contentstore.DaggerheartAdversaryEntry{
			RelentlessRule: &contentstore.DaggerheartAdversaryRelentlessRule{
				MaxSpotlightsPerGMTurn: 0,
			},
		}
		if got := AdversarySpotlightCap(entry); got != AdversaryDefaultSpotlightCap {
			t.Fatalf("spotlight cap = %d, want %d", got, AdversaryDefaultSpotlightCap)
		}
	})
}

func TestAdversaryIsBloodied(t *testing.T) {
	tests := []struct {
		name  string
		hp    int
		hpMax int
		want  bool
	}{
		{name: "half hp", hp: 4, hpMax: 8, want: true},
		{name: "below half hp", hp: 3, hpMax: 8, want: true},
		{name: "above half hp", hp: 5, hpMax: 8, want: false},
		{name: "zero max hp", hp: 0, hpMax: 0, want: false},
		{name: "negative hp", hp: -1, hpMax: 8, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AdversaryIsBloodied(tc.hp, tc.hpMax); got != tc.want {
				t.Fatalf("AdversaryIsBloodied(%d, %d) = %t, want %t", tc.hp, tc.hpMax, got, tc.want)
			}
		})
	}
}

func TestAdversaryStandardAttack(t *testing.T) {
	standard := contentstore.DaggerheartAdversaryAttack{Name: "Standard"}
	bloodied := contentstore.DaggerheartAdversaryAttack{Name: "Bloodied"}
	entry := contentstore.DaggerheartAdversaryEntry{
		StandardAttack: standard,
		HordeRule: &contentstore.DaggerheartAdversaryHordeRule{
			BloodiedAttack: bloodied,
		},
	}

	if got := AdversaryStandardAttack(entry, 5, 8); got.Name != standard.Name {
		t.Fatalf("non-bloodied attack = %q, want %q", got.Name, standard.Name)
	}
	if got := AdversaryStandardAttack(entry, 4, 8); got.Name != bloodied.Name {
		t.Fatalf("bloodied attack = %q, want %q", got.Name, bloodied.Name)
	}

	entry.HordeRule = nil
	if got := AdversaryStandardAttack(entry, 4, 8); got.Name != standard.Name {
		t.Fatalf("attack without horde rule = %q, want %q", got.Name, standard.Name)
	}
}

func TestAdversaryIsMinion(t *testing.T) {
	if AdversaryIsMinion(contentstore.DaggerheartAdversaryEntry{}) {
		t.Fatal("expected entry without minion rule to be false")
	}
	if AdversaryIsMinion(contentstore.DaggerheartAdversaryEntry{
		MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 0},
	}) {
		t.Fatal("expected non-positive minion rule to be false")
	}
	if !AdversaryIsMinion(contentstore.DaggerheartAdversaryEntry{
		MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3},
	}) {
		t.Fatal("expected positive minion rule to be true")
	}
}

func TestAdversaryMinionSpilloverDefeats(t *testing.T) {
	entry := contentstore.DaggerheartAdversaryEntry{
		MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3},
	}

	if got := AdversaryMinionSpilloverDefeats(contentstore.DaggerheartAdversaryEntry{}, 6); got != 0 {
		t.Fatalf("spillover without minion rule = %d, want 0", got)
	}
	if got := AdversaryMinionSpilloverDefeats(entry, 0); got != 0 {
		t.Fatalf("spillover without damage = %d, want 0", got)
	}
	if got := AdversaryMinionSpilloverDefeats(entry, 2); got != 0 {
		t.Fatalf("spillover below step = %d, want 0", got)
	}
	if got := AdversaryMinionSpilloverDefeats(entry, 3); got != 1 {
		t.Fatalf("spillover at one step = %d, want 1", got)
	}
	if got := AdversaryMinionSpilloverDefeats(entry, 7); got != 2 {
		t.Fatalf("spillover at two steps = %d, want 2", got)
	}
}
