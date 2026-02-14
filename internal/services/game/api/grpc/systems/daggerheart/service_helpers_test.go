package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
)

func TestStepDataToStructNil(t *testing.T) {
	data, err := stepDataToStruct(nil)
	if err != nil {
		t.Fatalf("step data: %v", err)
	}
	if len(data.GetFields()) != 0 {
		t.Fatalf("expected empty struct, got %v", data.GetFields())
	}
}

func TestNormalizeStructValue(t *testing.T) {
	input := map[string]any{
		"int":     1,
		"int32":   int32(2),
		"int64":   int64(3),
		"float":   1.5,
		"bool":    true,
		"string":  "ok",
		"list":    []any{1, "two"},
		"nested":  map[string]any{"inner": 4},
		"complex": []any{map[string]any{"x": 5}},
	}

	value, err := normalizeStructValue(input)
	if err != nil {
		t.Fatalf("normalize struct value: %v", err)
	}
	if value == nil {
		t.Fatal("expected non-nil converted value")
	}
}

func TestNormalizeStructValueRejectsNil(t *testing.T) {
	if _, err := normalizeStructValue(nil); err == nil {
		t.Fatal("expected error for nil value")
	}
}

func TestNormalizeStructValueRejectsUnsupported(t *testing.T) {
	if _, err := normalizeStructValue(struct{}{}); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestOutcomeToProtoFallback(t *testing.T) {
	if outcomeToProto(daggerheartdomain.Outcome(99)) != pb.Outcome_OUTCOME_UNSPECIFIED {
		t.Fatal("expected unspecified outcome for unknown value")
	}
}

func TestInt32Slice(t *testing.T) {
	if int32Slice(nil) != nil {
		t.Fatal("expected nil slice for nil input")
	}
	values := int32Slice([]int{1, 2, 3})
	if len(values) != 3 || values[2] != 3 {
		t.Fatalf("unexpected conversion: %v", values)
	}
}
