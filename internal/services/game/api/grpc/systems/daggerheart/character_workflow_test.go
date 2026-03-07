package daggerheart

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/workflow"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProgressFromDaggerheart(t *testing.T) {
	dhProgress := daggerheart.CreationProgress{
		Steps: []daggerheart.CreationStepProgress{
			{Step: 1, Key: "class_subclass", Complete: true},
			{Step: 2, Key: "heritage", Complete: false},
		},
		NextStep:     2,
		Ready:        false,
		UnmetReasons: []string{"heritage required"},
	}

	result := progressFromDaggerheart(dhProgress)

	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Key != "class_subclass" || !result.Steps[0].Complete {
		t.Fatalf("step 0 = %+v, want class_subclass/complete", result.Steps[0])
	}
	if result.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", result.NextStep)
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
	if len(result.UnmetReasons) != 1 || result.UnmetReasons[0] != "heritage required" {
		t.Fatalf("UnmetReasons = %v, want [heritage required]", result.UnmetReasons)
	}
}

func TestProgressFromDaggerheart_Empty(t *testing.T) {
	result := progressFromDaggerheart(daggerheart.CreationProgress{})

	if len(result.Steps) != 0 {
		t.Fatalf("steps = %d, want 0", len(result.Steps))
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
}

func TestProgressFromDaggerheart_UnmetReasonsIsolated(t *testing.T) {
	reasons := []string{"a", "b"}
	dhProgress := daggerheart.CreationProgress{UnmetReasons: reasons}

	result := progressFromDaggerheart(dhProgress)

	// Mutating the original should not affect the copy.
	reasons[0] = "mutated"
	if result.UnmetReasons[0] != "a" {
		t.Fatal("UnmetReasons not copied; mutation leaked")
	}
}

func TestCreationStepNumber(t *testing.T) {
	tests := []struct {
		name  string
		input *daggerheartv1.DaggerheartCreationStepInput
		want  int32
		code  codes.Code
	}{
		{
			name:  "nil input",
			input: nil,
			code:  codes.InvalidArgument,
		},
		{
			name: "class_subclass",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{},
			},
			want: daggerheart.CreationStepClassSubclass,
		},
		{
			name: "heritage",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{},
			},
			want: daggerheart.CreationStepHeritage,
		},
		{
			name: "traits",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{},
			},
			want: daggerheart.CreationStepTraits,
		},
		{
			name: "details",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{},
			},
			want: daggerheart.CreationStepDetails,
		},
		{
			name: "equipment",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{},
			},
			want: daggerheart.CreationStepEquipment,
		},
		{
			name: "background",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{},
			},
			want: daggerheart.CreationStepBackground,
		},
		{
			name: "experiences",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{},
			},
			want: daggerheart.CreationStepExperiences,
		},
		{
			name: "domain_cards",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{},
			},
			want: daggerheart.CreationStepDomainCards,
		},
		{
			name: "connections",
			input: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{},
			},
			want: daggerheart.CreationStepConnections,
		},
		{
			name:  "empty oneof",
			input: &daggerheartv1.DaggerheartCreationStepInput{},
			code:  codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := creationStepNumber(tt.input)
			if tt.code != codes.OK {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tt.code)
				}
				if status.Code(err) != tt.code {
					t.Fatalf("error code = %v, want %v", status.Code(err), tt.code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("step = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDefaultProfileForCharacter(t *testing.T) {
	rec := storage.CharacterRecord{ID: "ch1", Kind: character.KindPC}
	profile := defaultProfileForCharacter("c1", rec)

	if profile.CampaignID != "c1" {
		t.Fatalf("CampaignID = %q, want %q", profile.CampaignID, "c1")
	}
	if profile.CharacterID != "ch1" {
		t.Fatalf("CharacterID = %q, want %q", profile.CharacterID, "ch1")
	}
	if profile.HpMax == 0 {
		t.Fatal("HpMax should have a default > 0")
	}
}

func TestEnsureProfileDefaults_PreservesExisting(t *testing.T) {
	profile := storage.DaggerheartCharacterProfile{
		HpMax:     20,
		StressMax: 8,
		Evasion:   12,
		Level:     3,
	}
	result := ensureProfileDefaults(profile, character.KindPC)

	if result.HpMax != 20 {
		t.Fatalf("HpMax = %d, want 20 (should preserve existing)", result.HpMax)
	}
	if result.StressMax != 8 {
		t.Fatalf("StressMax = %d, want 8", result.StressMax)
	}
	if result.Evasion != 12 {
		t.Fatalf("Evasion = %d, want 12", result.Evasion)
	}
	if result.Level != 3 {
		t.Fatalf("Level = %d, want 3", result.Level)
	}
}

func TestEnsureProfileDefaults_NPC(t *testing.T) {
	profile := ensureProfileDefaults(storage.DaggerheartCharacterProfile{}, character.KindNPC)

	if profile.HpMax == 0 {
		t.Fatal("NPC HpMax should have a default > 0")
	}
}

func TestCreationStepSequenceFromWorkflowInput_NilInput(t *testing.T) {
	_, err := creationStepSequenceFromWorkflowInput(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestCreationStepSequenceFromWorkflowInput_MissingClassSubclass(t *testing.T) {
	_, err := creationStepSequenceFromWorkflowInput(&daggerheartv1.DaggerheartCreationWorkflowInput{})
	if err == nil {
		t.Fatal("expected error for missing class_subclass")
	}
}

func TestSystemProfileMap_Empty(t *testing.T) {
	m := SystemProfileMap(storage.DaggerheartCharacterProfile{})
	if m == nil {
		t.Fatal("expected non-nil map")
	}
}

var _ workflow.Provider = (*CreationWorkflowProvider)(nil)
