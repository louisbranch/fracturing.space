package creationworkflow

import (
	"context"
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeStartingWeaponIDs(t *testing.T) {
	tests := []struct {
		name     string
		weapons  []startingWeaponSelection
		want     []string
		wantCode codes.Code
	}{
		{
			name: "two-handed primary only",
			weapons: []startingWeaponSelection{
				{ID: "weapon.greatsword", Category: "primary", Tier: 1, Burden: 2},
			},
			want:     []string{"weapon.greatsword"},
			wantCode: codes.OK,
		},
		{
			name: "one-handed primary and secondary normalize to primary first",
			weapons: []startingWeaponSelection{
				{ID: "weapon.dagger", Category: "secondary", Tier: 1, Burden: 1},
				{ID: "weapon.longsword", Category: "primary", Tier: 1, Burden: 1},
			},
			want:     []string{"weapon.longsword", "weapon.dagger"},
			wantCode: codes.OK,
		},
		{
			name: "one-handed primary without secondary",
			weapons: []startingWeaponSelection{
				{ID: "weapon.longsword", Category: "primary", Tier: 1, Burden: 1},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "two-handed primary with secondary",
			weapons: []startingWeaponSelection{
				{ID: "weapon.greatsword", Category: "primary", Tier: 1, Burden: 2},
				{ID: "weapon.dagger", Category: "secondary", Tier: 1, Burden: 1},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "secondary must be one-handed",
			weapons: []startingWeaponSelection{
				{ID: "weapon.longsword", Category: "primary", Tier: 1, Burden: 1},
				{ID: "weapon.tower-shield", Category: "secondary", Tier: 1, Burden: 2},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "primary burden must be one or two",
			weapons: []startingWeaponSelection{
				{ID: "weapon.oddity", Category: "primary", Tier: 1, Burden: 3},
				{ID: "weapon.dagger", Category: "secondary", Tier: 1, Burden: 1},
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeStartingWeaponIDs(tt.weapons)
			if tt.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tt.wantCode)
				}
				if status.Code(err) != tt.wantCode {
					t.Fatalf("error code = %v, want %v", status.Code(err), tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeStartingWeaponIDs() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("weapon ids = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("weapon ids = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestApplyEquipmentInput(t *testing.T) {
	profile := &projectionstore.DaggerheartCharacterProfile{}
	err := applyEquipmentInput(
		context.Background(),
		newTestContentStore(),
		profile,
		&daggerheartv1.DaggerheartCreationStepEquipmentInput{
			WeaponIds:    []string{"weapon-secondary-1", "weapon-primary-1"},
			ArmorId:      "armor-1",
			PotionItemId: daggerheart.StartingPotionMinorHealthID,
		},
	)
	if err != nil {
		t.Fatalf("applyEquipmentInput() error = %v", err)
	}
	if profile.Level != daggerheartprofile.PCLevelDefault {
		t.Fatalf("profile.Level = %d, want %d", profile.Level, daggerheartprofile.PCLevelDefault)
	}
	if len(profile.StartingWeaponIDs) != 2 || profile.StartingWeaponIDs[0] != "weapon-primary-1" || profile.StartingWeaponIDs[1] != "weapon-secondary-1" {
		t.Fatalf("profile.StartingWeaponIDs = %v, want [weapon-primary-1 weapon-secondary-1]", profile.StartingWeaponIDs)
	}
	if profile.StartingArmorID != "armor-1" || profile.StartingPotionItemID != daggerheart.StartingPotionMinorHealthID {
		t.Fatalf("profile equipment = armor:%q potion:%q", profile.StartingArmorID, profile.StartingPotionItemID)
	}
	if profile.Proficiency != daggerheartprofile.PCProficiency || profile.ArmorScore != 2 {
		t.Fatalf("profile proficiency/armor = (%d, %d), want (%d, 2)", profile.Proficiency, profile.ArmorScore, daggerheartprofile.PCProficiency)
	}
	if profile.MajorThreshold != 7 || profile.SevereThreshold != 13 {
		t.Fatalf("profile thresholds = (%d, %d), want (7, 13)", profile.MajorThreshold, profile.SevereThreshold)
	}
}
