package daggerheart

import "testing"

func TestRollDamage(t *testing.T) {
	result, err := RollDamage(DamageRollRequest{
		Dice:     []DamageDieSpec{{Sides: 8, Count: 2}},
		Modifier: 2,
		Seed:     1,
		Critical: false,
	})
	if err != nil {
		t.Fatalf("RollDamage returned error: %v", err)
	}
	if result.BaseTotal == 0 {
		t.Fatal("expected base total")
	}
	if result.CriticalBonus != 0 {
		t.Fatalf("critical bonus = %d, want 0", result.CriticalBonus)
	}
	if result.Total != result.BaseTotal {
		t.Fatalf("total = %d, want %d", result.Total, result.BaseTotal)
	}
}

func TestRollDamageCriticalAddsMaxDice(t *testing.T) {
	result, err := RollDamage(DamageRollRequest{
		Dice:     []DamageDieSpec{{Sides: 6, Count: 2}, {Sides: 4, Count: 1}},
		Modifier: 1,
		Seed:     2,
		Critical: true,
	})
	if err != nil {
		t.Fatalf("RollDamage returned error: %v", err)
	}
	if result.CriticalBonus != 16 {
		t.Fatalf("critical bonus = %d, want 16", result.CriticalBonus)
	}
	if result.Total != result.BaseTotal+result.CriticalBonus {
		t.Fatalf("total = %d, want %d", result.Total, result.BaseTotal+result.CriticalBonus)
	}
}
