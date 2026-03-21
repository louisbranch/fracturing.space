package rules

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

func TestResolveDamageApplication_WardedReducesMagicBeforeThresholds(t *testing.T) {
	application, mitigated, err := ResolveDamageApplication(
		DamageTarget{
			HP:              6,
			Armor:           4,
			Stress:          1,
			MajorThreshold:  5,
			SevereThreshold: 10,
			ArmorRules: ArmorDamageRules{
				WardedMagicReduction:  true,
				WardedReductionAmount: 4,
			},
		},
		DamageApplyInput{
			Amount: 8,
			Types:  DamageTypes{Magic: true},
		},
	)
	if err != nil {
		t.Fatalf("ResolveDamageApplication() error = %v", err)
	}
	if !mitigated {
		t.Fatal("mitigated = false, want true")
	}
	if application.ArmorSpent != 1 {
		t.Fatalf("armor spent = %d, want 1", application.ArmorSpent)
	}
	if application.Result.Severity != DamageNone || application.HPAfter != 6 {
		t.Fatalf("application = %+v, want no hp damage after warded reduction and armor spend", application)
	}
}

func TestResolveDamageApplication_PhysicalArmorCannotMitigateMagic(t *testing.T) {
	application, mitigated, err := ResolveDamageApplication(
		DamageTarget{
			HP:              6,
			Armor:           2,
			MajorThreshold:  5,
			SevereThreshold: 10,
			ArmorRules: ArmorDamageRules{
				MitigationMode: "physical_only",
			},
		},
		DamageApplyInput{
			Amount: 10,
			Types:  DamageTypes{Magic: true},
		},
	)
	if err != nil {
		t.Fatalf("ResolveDamageApplication() error = %v", err)
	}
	if mitigated {
		t.Fatal("mitigated = true, want false when physical armor meets magic damage")
	}
	if application.ArmorSpent != 0 || application.HPAfter != 3 {
		t.Fatalf("application = %+v, want no armor spend and 3 hp after", application)
	}
}

func TestResolveDamageApplication_FortifiedPainfulArmor(t *testing.T) {
	application, mitigated, err := ResolveDamageApplication(
		DamageTarget{
			HP:              6,
			Stress:          1,
			Armor:           1,
			MajorThreshold:  5,
			SevereThreshold: 10,
			ArmorRules: ArmorDamageRules{
				SeverityReductionSteps: 2,
				StressOnMark:           true,
			},
		},
		DamageApplyInput{
			Amount: 10,
			Types:  DamageTypes{Physical: true},
		},
	)
	if err != nil {
		t.Fatalf("ResolveDamageApplication() error = %v", err)
	}
	if !mitigated {
		t.Fatal("mitigated = false, want true")
	}
	if application.Result.Severity != DamageMinor || application.HPAfter != 5 {
		t.Fatalf("application = %+v, want minor severity and 5 hp after", application)
	}
	if application.StressBefore != 1 || application.StressAfter != 2 {
		t.Fatalf("stress = %d->%d, want 1->2", application.StressBefore, application.StressAfter)
	}
}
