package daggerheart

import "testing"

func TestResolveDamageApplication_DirectDamage(t *testing.T) {
	application, mitigated, err := ResolveDamageApplication(
		DamageTarget{HP: 6, Armor: 2, MajorThreshold: 5, SevereThreshold: 10},
		DamageApplyInput{
			Amount:       10,
			Types:        DamageTypes{Physical: true},
			Resistance:   ResistanceProfile{ResistPhysical: true},
			Direct:       true,
			AllowMassive: false,
		},
	)
	if err != nil {
		t.Fatalf("ResolveDamageApplication() error = %v", err)
	}
	if !mitigated {
		t.Fatal("mitigated = false, want true (resistance should mitigate)")
	}
	if application.HPBefore != 6 || application.HPAfter != 4 {
		t.Fatalf("hp = %d->%d, want 6->4", application.HPBefore, application.HPAfter)
	}
	if application.ArmorSpent != 0 {
		t.Fatalf("armor spent = %d, want 0 for direct damage", application.ArmorSpent)
	}
}

func TestResolveDamageApplication_ArmorMitigation(t *testing.T) {
	application, mitigated, err := ResolveDamageApplication(
		DamageTarget{HP: 6, Armor: 1, MajorThreshold: 5, SevereThreshold: 10},
		DamageApplyInput{
			Amount:       10,
			Types:        DamageTypes{Physical: true},
			Resistance:   ResistanceProfile{},
			Direct:       false,
			AllowMassive: false,
		},
	)
	if err != nil {
		t.Fatalf("ResolveDamageApplication() error = %v", err)
	}
	if !mitigated {
		t.Fatal("mitigated = false, want true (armor spend should mitigate)")
	}
	if application.ArmorBefore != 1 || application.ArmorAfter != 0 || application.ArmorSpent != 1 {
		t.Fatalf("armor transition = %d->%d spent=%d, want 1->0 spent=1", application.ArmorBefore, application.ArmorAfter, application.ArmorSpent)
	}
	if application.HPBefore != 6 || application.HPAfter != 4 {
		t.Fatalf("hp = %d->%d, want 6->4", application.HPBefore, application.HPAfter)
	}
}

func TestResolveDamageApplication_InvalidThresholds(t *testing.T) {
	_, _, err := ResolveDamageApplication(
		DamageTarget{HP: 6, Armor: 1, MajorThreshold: 10, SevereThreshold: 8},
		DamageApplyInput{Amount: 5, Types: DamageTypes{Physical: true}},
	)
	if err == nil {
		t.Fatal("expected invalid threshold error")
	}
}
