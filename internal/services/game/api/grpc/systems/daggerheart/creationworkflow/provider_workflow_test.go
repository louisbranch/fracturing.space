package creationworkflow

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

var _ characterworkflow.Provider = (*CreationWorkflowProvider)(nil)
