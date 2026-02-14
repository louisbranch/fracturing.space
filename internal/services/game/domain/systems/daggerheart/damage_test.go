package daggerheart

import (
	"errors"
	"testing"
)

func TestEvaluateDamage(t *testing.T) {
	tests := []struct {
		name            string
		amount          int
		majorThreshold  int
		severeThreshold int
		opts            DamageOptions
		wantSeverity    DamageSeverity
		wantMarks       int
		wantErr         error
	}{
		{"invalid thresholds", 5, 10, 8, DamageOptions{}, 0, 0, ErrInvalidThresholds},
		{"zero damage", 0, 5, 10, DamageOptions{}, DamageNone, 0, nil},
		{"minor damage", 4, 5, 10, DamageOptions{}, DamageMinor, 1, nil},
		{"major at threshold", 5, 5, 10, DamageOptions{}, DamageMajor, 2, nil},
		{"major between", 9, 5, 10, DamageOptions{}, DamageMajor, 2, nil},
		{"severe at threshold", 10, 5, 10, DamageOptions{}, DamageSevere, 3, nil},
		{"massive enabled", 20, 5, 10, DamageOptions{EnableMassiveDamage: true}, DamageMassive, 4, nil},
		{"massive disabled", 15, 5, 10, DamageOptions{}, DamageSevere, 3, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateDamage(tt.amount, tt.majorThreshold, tt.severeThreshold, tt.opts)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Severity != tt.wantSeverity {
				t.Fatalf("severity = %v, want %v", got.Severity, tt.wantSeverity)
			}
			if got.Marks != tt.wantMarks {
				t.Fatalf("marks = %d, want %d", got.Marks, tt.wantMarks)
			}
		})
	}
}

func TestApplyDamageMarks(t *testing.T) {
	tests := []struct {
		name      string
		currentHP int
		marks     int
		wantAfter int
	}{
		{"no marks", 6, 0, 6},
		{"negative marks", 6, -1, 6},
		{"reduce", 6, 2, 4},
		{"floor at zero", 3, 5, 0},
		{"already zero", 0, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, after := ApplyDamageMarks(tt.currentHP, tt.marks)
			if after != tt.wantAfter {
				t.Fatalf("after = %d, want %d", after, tt.wantAfter)
			}
		})
	}
}

func TestApplyDamage(t *testing.T) {
	app, err := ApplyDamage(6, 10, 5, 10, DamageOptions{})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if app.Result.Marks != 3 {
		t.Fatalf("marks = %d, want 3", app.Result.Marks)
	}
	if app.HPBefore != 6 || app.HPAfter != 3 {
		t.Fatalf("hp = %d->%d, want 6->3", app.HPBefore, app.HPAfter)
	}
}

func TestApplyResistance(t *testing.T) {
	tests := []struct {
		name   string
		amount int
		types  DamageTypes
		resist ResistanceProfile
		want   int
	}{
		{"no types", 10, DamageTypes{}, ResistanceProfile{}, 10},
		{"physical resist", 9, DamageTypes{Physical: true}, ResistanceProfile{ResistPhysical: true}, 4},
		{"physical no resist", 9, DamageTypes{Physical: true}, ResistanceProfile{}, 9},
		{"magic resist", 9, DamageTypes{Magic: true}, ResistanceProfile{ResistMagic: true}, 4},
		{"mixed resist both", 9, DamageTypes{Physical: true, Magic: true}, ResistanceProfile{ResistPhysical: true, ResistMagic: true}, 4},
		{"mixed resist one", 9, DamageTypes{Physical: true, Magic: true}, ResistanceProfile{ResistPhysical: true}, 9},
		{"immune physical", 9, DamageTypes{Physical: true}, ResistanceProfile{ImmunePhysical: true}, 0},
		{"immune mixed both", 9, DamageTypes{Physical: true, Magic: true}, ResistanceProfile{ImmunePhysical: true, ImmuneMagic: true}, 0},
		{"immune mixed one", 9, DamageTypes{Physical: true, Magic: true}, ResistanceProfile{ImmunePhysical: true}, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyResistance(tt.amount, tt.types, tt.resist)
			if got != tt.want {
				t.Fatalf("damage = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestReduceDamageWithArmor(t *testing.T) {
	result, spent := ReduceDamageWithArmor(DamageResult{Severity: DamageSevere, Marks: 3}, 1)
	if spent != 1 {
		t.Fatalf("spent = %d, want 1", spent)
	}
	if result.Severity != DamageMajor || result.Marks != 2 {
		t.Fatalf("result = %+v, want major/2", result)
	}

	result, spent = ReduceDamageWithArmor(DamageResult{Severity: DamageMinor, Marks: 1}, 1)
	if result.Marks != 0 || result.Severity != DamageNone {
		t.Fatalf("result = %+v, want none/0", result)
	}
	if spent != 1 {
		t.Fatalf("spent = %d, want 1", spent)
	}

	result, spent = ReduceDamageWithArmor(DamageResult{Severity: DamageMajor, Marks: 2}, 0)
	if spent != 0 || result.Marks != 2 {
		t.Fatalf("unexpected armor spend with no slots")
	}
}

func TestApplyDamageWithArmor(t *testing.T) {
	result := DamageResult{Severity: DamageSevere, Marks: 3}
	app := ApplyDamageWithArmor(6, 1, result)
	if app.ArmorSpent != 1 || app.ArmorAfter != 0 {
		t.Fatalf("armor = %d->%d, spent %d", app.ArmorBefore, app.ArmorAfter, app.ArmorSpent)
	}
	if app.HPAfter != 4 {
		t.Fatalf("hp after = %d, want 4", app.HPAfter)
	}
}
