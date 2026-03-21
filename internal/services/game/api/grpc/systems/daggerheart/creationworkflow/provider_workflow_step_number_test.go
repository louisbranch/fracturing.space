package creationworkflow

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreationStepNumber(t *testing.T) {
	tests := []struct {
		name  string
		input *daggerheartv1.DaggerheartCreationStepInput
		want  int32
		code  codes.Code
	}{
		{name: "nil input", input: nil, code: codes.InvalidArgument},
		{
			name:  "class_subclass",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{}},
			want:  daggerheart.CreationStepClassSubclass,
		},
		{
			name:  "heritage",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{}},
			want:  daggerheart.CreationStepHeritage,
		},
		{
			name:  "traits",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{}},
			want:  daggerheart.CreationStepTraits,
		},
		{
			name:  "details",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{}},
			want:  daggerheart.CreationStepDetails,
		},
		{
			name:  "equipment",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{}},
			want:  daggerheart.CreationStepEquipment,
		},
		{
			name:  "background",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{}},
			want:  daggerheart.CreationStepBackground,
		},
		{
			name:  "experiences",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{}},
			want:  daggerheart.CreationStepExperiences,
		},
		{
			name:  "domain_cards",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{}},
			want:  daggerheart.CreationStepDomainCards,
		},
		{
			name:  "connections",
			input: &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{}},
			want:  daggerheart.CreationStepConnections,
		},
		{name: "empty oneof", input: &daggerheartv1.DaggerheartCreationStepInput{}, code: codes.InvalidArgument},
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
