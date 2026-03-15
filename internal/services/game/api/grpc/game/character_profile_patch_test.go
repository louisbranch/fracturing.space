package game

import (
	"reflect"
	"strings"
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestApplyDaggerheartProfilePatch_NilPatchNoChange(t *testing.T) {
	current := validPatchProfile()

	updated, err := applyDaggerheartProfilePatch(current, nil)
	if err != nil {
		t.Fatalf("apply nil patch: %v", err)
	}
	if !reflect.DeepEqual(updated, current) {
		t.Fatalf("updated profile mismatch\n got: %#v\nwant: %#v", updated, current)
	}
}

func TestApplyDaggerheartProfilePatch_AppliesMutableFields(t *testing.T) {
	current := validPatchProfile()

	updated, err := applyDaggerheartProfilePatch(current, &daggerheartv1.DaggerheartProfile{
		HpMax:      10,
		StressMax:  wrapperspb.Int32(8),
		ArmorScore: wrapperspb.Int32(2),
		ArmorMax:   wrapperspb.Int32(4),
	})
	if err != nil {
		t.Fatalf("apply patch: %v", err)
	}
	if updated.HpMax != 10 {
		t.Fatalf("hp_max = %d, want 10", updated.HpMax)
	}
	if updated.StressMax != 8 {
		t.Fatalf("stress_max = %d, want 8", updated.StressMax)
	}
	if updated.ArmorScore != 2 {
		t.Fatalf("armor_score = %d, want 2", updated.ArmorScore)
	}
	if updated.ArmorMax != 4 {
		t.Fatalf("armor_max = %d, want 4", updated.ArmorMax)
	}
}

func TestApplyDaggerheartProfilePatch_RejectsCreationWorkflowFields(t *testing.T) {
	_, err := applyDaggerheartProfilePatch(validPatchProfile(), &daggerheartv1.DaggerheartProfile{
		ClassId: "class.guardian",
	})
	if err == nil {
		t.Fatal("expected rejection for creation workflow fields")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if !strings.Contains(err.Error(), "ApplyCharacterCreationStep") {
		t.Fatalf("error = %v, want creation workflow guidance", err)
	}
}

func TestApplyDaggerheartProfilePatch_RejectsDescriptionAsCreationWorkflowField(t *testing.T) {
	_, err := applyDaggerheartProfilePatch(validPatchProfile(), &daggerheartv1.DaggerheartProfile{
		Description: "Reserved for the creation workflow.",
	})
	if err == nil {
		t.Fatal("expected rejection for description patch")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if !strings.Contains(err.Error(), "ApplyCharacterCreationStep") {
		t.Fatalf("error = %v, want creation workflow guidance", err)
	}
}

func validPatchProfile() projectionstore.DaggerheartCharacterProfile {
	defaults := daggerheartprofile.GetDefaults("PC")
	return projectionstore.DaggerheartCharacterProfile{
		Level:           defaults.Level,
		HpMax:           defaults.HpMax,
		StressMax:       defaults.StressMax,
		Evasion:         defaults.Evasion,
		MajorThreshold:  defaults.MajorThreshold,
		SevereThreshold: defaults.SevereThreshold,
		Proficiency:     defaults.Proficiency,
		ArmorScore:      defaults.ArmorScore,
		ArmorMax:        defaults.ArmorMax,
		Agility:         defaults.Traits.Agility,
		Strength:        defaults.Traits.Strength,
		Finesse:         defaults.Traits.Finesse,
		Instinct:        defaults.Traits.Instinct,
		Presence:        defaults.Traits.Presence,
		Knowledge:       defaults.Traits.Knowledge,
	}
}
