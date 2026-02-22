package filter

import (
	"testing"

	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func TestParse(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		e, err := Parse("", Fields{"name": FieldString})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e != nil {
			t.Fatal("expected nil expr for empty filter")
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		e, err := Parse("   ", Fields{"name": FieldString})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e != nil {
			t.Fatal("expected nil expr for whitespace filter")
		}
	})

	t.Run("valid string filter", func(t *testing.T) {
		e, err := Parse(`name = "foo"`, Fields{"name": FieldString})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e == nil {
			t.Fatal("expected non-nil expr")
		}
	})

	t.Run("valid int filter", func(t *testing.T) {
		e, err := Parse("level = 5", Fields{"level": FieldInt})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e == nil {
			t.Fatal("expected non-nil expr")
		}
	})

	t.Run("invalid filter syntax", func(t *testing.T) {
		_, err := Parse("!!!invalid", Fields{"name": FieldString})
		if err == nil {
			t.Fatal("expected error for invalid syntax")
		}
	})

	t.Run("unsupported field type", func(t *testing.T) {
		_, err := Parse(`x = "foo"`, Fields{"x": FieldType("complex")})
		if err == nil {
			t.Fatal("expected error for unsupported field type")
		}
	})
}

func TestEvaluate(t *testing.T) {
	resolve := func(name string) (any, bool) {
		switch name {
		case "name":
			return "alice", true
		case "level":
			return int64(5), true
		case "active":
			return true, true
		default:
			return nil, false
		}
	}

	t.Run("nil expression", func(t *testing.T) {
		ok, err := Evaluate(nil, resolve)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true for nil expression")
		}
	})

	t.Run("string equality match", func(t *testing.T) {
		e, err := Parse(`name = "alice"`, Fields{"name": FieldString})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected match")
		}
	})

	t.Run("string equality no match", func(t *testing.T) {
		e, err := Parse(`name = "bob"`, Fields{"name": FieldString})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if ok {
			t.Error("expected no match")
		}
	})

	t.Run("string inequality", func(t *testing.T) {
		e, err := Parse(`name != "bob"`, Fields{"name": FieldString})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected match for inequality")
		}
	})

	t.Run("int less than", func(t *testing.T) {
		e, err := Parse("level < 10", Fields{"level": FieldInt})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected 5 < 10")
		}
	})

	t.Run("int greater than no match", func(t *testing.T) {
		e, err := Parse("level > 10", Fields{"level": FieldInt})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if ok {
			t.Error("expected 5 > 10 to be false")
		}
	})

	t.Run("int less than or equal", func(t *testing.T) {
		e, err := Parse("level <= 5", Fields{"level": FieldInt})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected 5 <= 5")
		}
	})

	t.Run("int greater than or equal", func(t *testing.T) {
		e, err := Parse("level >= 5", Fields{"level": FieldInt})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected 5 >= 5")
		}
	})

	t.Run("AND expression", func(t *testing.T) {
		e, err := Parse(`name = "alice" AND level = 5`, Fields{
			"name":  FieldString,
			"level": FieldInt,
		})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected AND to match")
		}
	})

	t.Run("AND short circuit", func(t *testing.T) {
		e, err := Parse(`name = "bob" AND level = 5`, Fields{
			"name":  FieldString,
			"level": FieldInt,
		})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if ok {
			t.Error("expected AND to fail when first arg is false")
		}
	})

	t.Run("OR expression", func(t *testing.T) {
		e, err := Parse(`name = "bob" OR level = 5`, Fields{
			"name":  FieldString,
			"level": FieldInt,
		})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		ok, err := Evaluate(e, resolve)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !ok {
			t.Error("expected OR to match when second is true")
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		e, err := Parse(`missing = "x"`, Fields{"missing": FieldString})
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		_, err = Evaluate(e, resolve)
		if err == nil {
			t.Error("expected error for unknown field")
		}
	})

	t.Run("unsupported expression type", func(t *testing.T) {
		// A bare identifier expression is unsupported at the top level.
		e := &expr.Expr{
			ExprKind: &expr.Expr_IdentExpr{
				IdentExpr: &expr.Expr_Ident{Name: "x"},
			},
		}
		_, err := Evaluate(e, resolve)
		if err == nil {
			t.Error("expected error for unsupported expression type")
		}
	})
}

func TestCompareValues(t *testing.T) {
	t.Run("string equal", func(t *testing.T) {
		cmp, err := compareValues("a", "a")
		if err != nil {
			t.Fatal(err)
		}
		if cmp != 0 {
			t.Errorf("expected 0, got %d", cmp)
		}
	})

	t.Run("string less", func(t *testing.T) {
		cmp, err := compareValues("a", "b")
		if err != nil {
			t.Fatal(err)
		}
		if cmp >= 0 {
			t.Errorf("expected < 0, got %d", cmp)
		}
	})

	t.Run("string greater", func(t *testing.T) {
		cmp, err := compareValues("b", "a")
		if err != nil {
			t.Fatal(err)
		}
		if cmp <= 0 {
			t.Errorf("expected > 0, got %d", cmp)
		}
	})

	t.Run("string type mismatch", func(t *testing.T) {
		_, err := compareValues("a", 1)
		if err == nil {
			t.Error("expected type mismatch error")
		}
	})

	t.Run("int equal", func(t *testing.T) {
		cmp, err := compareValues(int(5), int64(5))
		if err != nil {
			t.Fatal(err)
		}
		if cmp != 0 {
			t.Errorf("expected 0, got %d", cmp)
		}
	})

	t.Run("int32 less", func(t *testing.T) {
		cmp, err := compareValues(int32(3), int64(5))
		if err != nil {
			t.Fatal(err)
		}
		if cmp >= 0 {
			t.Errorf("expected < 0, got %d", cmp)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		cmp, err := compareValues(uint64(10), int64(5))
		if err != nil {
			t.Fatal(err)
		}
		if cmp <= 0 {
			t.Errorf("expected > 0, got %d", cmp)
		}
	})

	t.Run("float32", func(t *testing.T) {
		cmp, err := compareValues(float32(1.5), float64(1.5))
		if err != nil {
			t.Fatal(err)
		}
		if cmp != 0 {
			t.Errorf("expected 0, got %d", cmp)
		}
	})

	t.Run("float64", func(t *testing.T) {
		cmp, err := compareValues(float64(2.0), int64(3))
		if err != nil {
			t.Fatal(err)
		}
		if cmp >= 0 {
			t.Errorf("expected < 0, got %d", cmp)
		}
	})

	t.Run("uint", func(t *testing.T) {
		cmp, err := compareValues(uint(7), int64(7))
		if err != nil {
			t.Fatal(err)
		}
		if cmp != 0 {
			t.Errorf("expected 0, got %d", cmp)
		}
	})

	t.Run("uint32", func(t *testing.T) {
		cmp, err := compareValues(uint32(4), int64(4))
		if err != nil {
			t.Fatal(err)
		}
		if cmp != 0 {
			t.Errorf("expected 0, got %d", cmp)
		}
	})

	t.Run("bool equal true", func(t *testing.T) {
		cmp, err := compareValues(true, true)
		if err != nil {
			t.Fatal(err)
		}
		if cmp != 0 {
			t.Errorf("expected 0, got %d", cmp)
		}
	})

	t.Run("bool false less than true", func(t *testing.T) {
		cmp, err := compareValues(false, true)
		if err != nil {
			t.Fatal(err)
		}
		if cmp >= 0 {
			t.Errorf("expected < 0, got %d", cmp)
		}
	})

	t.Run("bool true greater than false", func(t *testing.T) {
		cmp, err := compareValues(true, false)
		if err != nil {
			t.Fatal(err)
		}
		if cmp <= 0 {
			t.Errorf("expected > 0, got %d", cmp)
		}
	})

	t.Run("bool type mismatch", func(t *testing.T) {
		_, err := compareValues(true, "x")
		if err == nil {
			t.Error("expected type mismatch error")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := compareValues(struct{}{}, "x")
		if err == nil {
			t.Error("expected unsupported type error")
		}
	})

	t.Run("number vs non-number mismatch", func(t *testing.T) {
		_, err := compareValues(int(5), "x")
		if err == nil {
			t.Error("expected type mismatch error")
		}
	})
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  float64
		ok    bool
	}{
		{"int", int(5), 5, true},
		{"int32", int32(5), 5, true},
		{"int64", int64(5), 5, true},
		{"uint", uint(5), 5, true},
		{"uint32", uint32(5), 5, true},
		{"uint64", uint64(5), 5, true},
		{"float32", float32(2.5), 2.5, true},
		{"float64", float64(2.5), 2.5, true},
		{"string not convertible", "x", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toFloat(tc.value)
			if ok != tc.ok {
				t.Fatalf("toFloat(%v) ok = %v, want %v", tc.value, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("toFloat(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestExtractFieldName(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		_, err := extractFieldName(nil)
		if err == nil {
			t.Error("expected error for nil")
		}
	})

	t.Run("ident", func(t *testing.T) {
		e := &expr.Expr{
			ExprKind: &expr.Expr_IdentExpr{
				IdentExpr: &expr.Expr_Ident{Name: "field"},
			},
		}
		name, err := extractFieldName(e)
		if err != nil {
			t.Fatal(err)
		}
		if name != "field" {
			t.Errorf("got %q, want %q", name, "field")
		}
	})

	t.Run("non-ident", func(t *testing.T) {
		e := &expr.Expr{
			ExprKind: &expr.Expr_ConstExpr{
				ConstExpr: &expr.Constant{
					ConstantKind: &expr.Constant_StringValue{StringValue: "x"},
				},
			},
		}
		_, err := extractFieldName(e)
		if err == nil {
			t.Error("expected error for non-ident expression")
		}
	})
}

func TestExtractValue(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		_, err := extractValue(nil)
		if err == nil {
			t.Error("expected error for nil")
		}
	})

	t.Run("string constant", func(t *testing.T) {
		e := &expr.Expr{
			ExprKind: &expr.Expr_ConstExpr{
				ConstExpr: &expr.Constant{
					ConstantKind: &expr.Constant_StringValue{StringValue: "hello"},
				},
			},
		}
		v, err := extractValue(e)
		if err != nil {
			t.Fatal(err)
		}
		if v != "hello" {
			t.Errorf("got %v, want %q", v, "hello")
		}
	})

	t.Run("non-const", func(t *testing.T) {
		e := &expr.Expr{
			ExprKind: &expr.Expr_IdentExpr{
				IdentExpr: &expr.Expr_Ident{Name: "x"},
			},
		}
		_, err := extractValue(e)
		if err == nil {
			t.Error("expected error for non-const expression")
		}
	})
}

func TestExtractConstValue(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		_, err := extractConstValue(nil)
		if err == nil {
			t.Error("expected error for nil")
		}
	})

	t.Run("string", func(t *testing.T) {
		c := &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "hello"}}
		v, err := extractConstValue(c)
		if err != nil {
			t.Fatal(err)
		}
		if v != "hello" {
			t.Errorf("got %v, want %q", v, "hello")
		}
	})

	t.Run("int64", func(t *testing.T) {
		c := &expr.Constant{ConstantKind: &expr.Constant_Int64Value{Int64Value: 42}}
		v, err := extractConstValue(c)
		if err != nil {
			t.Fatal(err)
		}
		if v != int64(42) {
			t.Errorf("got %v, want 42", v)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		c := &expr.Constant{ConstantKind: &expr.Constant_Uint64Value{Uint64Value: 99}}
		v, err := extractConstValue(c)
		if err != nil {
			t.Fatal(err)
		}
		if v != uint64(99) {
			t.Errorf("got %v, want 99", v)
		}
	})

	t.Run("double", func(t *testing.T) {
		c := &expr.Constant{ConstantKind: &expr.Constant_DoubleValue{DoubleValue: 3.14}}
		v, err := extractConstValue(c)
		if err != nil {
			t.Fatal(err)
		}
		if v != 3.14 {
			t.Errorf("got %v, want 3.14", v)
		}
	})

	t.Run("bool", func(t *testing.T) {
		c := &expr.Constant{ConstantKind: &expr.Constant_BoolValue{BoolValue: true}}
		v, err := extractConstValue(c)
		if err != nil {
			t.Fatal(err)
		}
		if v != true {
			t.Errorf("got %v, want true", v)
		}
	})
}

func TestEvalCall_UnsupportedFunction(t *testing.T) {
	call := &expr.Expr_Call{
		Function: "unsupported_fn",
	}
	_, err := evalCall(call, func(string) (any, bool) { return nil, false })
	if err == nil {
		t.Error("expected error for unsupported function")
	}
}

func TestEvalOr_ShortCircuitTrue(t *testing.T) {
	// When the first operand is true, the second should never be evaluated.
	resolve := func(name string) (any, bool) {
		switch name {
		case "name":
			return "alice", true
		default:
			return nil, false
		}
	}
	e, err := Parse(`name = "alice" OR name = "bob"`, Fields{
		"name": FieldString,
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ok, err := Evaluate(e, resolve)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !ok {
		t.Error("expected OR to short-circuit true on first arg")
	}
}

func TestEvalOr_BothFalse(t *testing.T) {
	resolve := func(name string) (any, bool) {
		switch name {
		case "name":
			return "charlie", true
		default:
			return nil, false
		}
	}
	e, err := Parse(`name = "alice" OR name = "bob"`, Fields{
		"name": FieldString,
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ok, err := Evaluate(e, resolve)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if ok {
		t.Error("expected OR to be false when both operands are false")
	}
}

func TestEvalOr_WrongArgCount(t *testing.T) {
	call := &expr.Expr_Call{
		Function: "_||_",
		Args:     []*expr.Expr{{}}, // Only 1 arg, need 2
	}
	_, err := evalCall(call, func(string) (any, bool) { return nil, false })
	if err == nil {
		t.Error("expected error for OR with wrong arg count")
	}
}

func TestEvalAnd_WrongArgCount(t *testing.T) {
	call := &expr.Expr_Call{
		Function: "_&&_",
		Args:     []*expr.Expr{{}}, // Only 1 arg, need 2
	}
	_, err := evalCall(call, func(string) (any, bool) { return nil, false })
	if err == nil {
		t.Error("expected error for AND with wrong arg count")
	}
}

func TestEvalCompare_WrongArgCount(t *testing.T) {
	call := &expr.Expr_Call{
		Function: "_==_",
		Args:     []*expr.Expr{{}}, // Only 1 arg, need 2
	}
	_, err := evalCall(call, func(string) (any, bool) { return nil, false })
	if err == nil {
		t.Error("expected error for comparison with wrong arg count")
	}
}

func TestEvalOr_LeftError(t *testing.T) {
	// If the left side of OR evaluates to an error, the error propagates.
	resolve := func(name string) (any, bool) {
		return nil, false // unknown field causes error
	}
	e := &expr.Expr{
		ExprKind: &expr.Expr_CallExpr{
			CallExpr: &expr.Expr_Call{
				Function: "_||_",
				Args: []*expr.Expr{
					// Left: comparison with unknown field
					{ExprKind: &expr.Expr_CallExpr{
						CallExpr: &expr.Expr_Call{
							Function: "_==_",
							Args: []*expr.Expr{
								{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "unknown_field"}}},
								{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "x"}}}},
							},
						},
					}},
					// Right: doesn't matter
					{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_BoolValue{BoolValue: true}}}},
				},
			},
		},
	}
	_, err := Evaluate(e, resolve)
	if err == nil {
		t.Error("expected error when left side of OR fails")
	}
}

func TestEvalAnd_LeftError(t *testing.T) {
	// If the left side of AND evaluates to an error, the error propagates.
	resolve := func(name string) (any, bool) {
		return nil, false
	}
	e := &expr.Expr{
		ExprKind: &expr.Expr_CallExpr{
			CallExpr: &expr.Expr_Call{
				Function: "_&&_",
				Args: []*expr.Expr{
					{ExprKind: &expr.Expr_CallExpr{
						CallExpr: &expr.Expr_Call{
							Function: "_==_",
							Args: []*expr.Expr{
								{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "unknown_field"}}},
								{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "x"}}}},
							},
						},
					}},
					{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_BoolValue{BoolValue: true}}}},
				},
			},
		},
	}
	_, err := Evaluate(e, resolve)
	if err == nil {
		t.Error("expected error when left side of AND fails")
	}
}

func TestExtractConstValue_UnsupportedType(t *testing.T) {
	// A nil ConstantKind should return unsupported type error.
	c := &expr.Constant{ConstantKind: nil}
	_, err := extractConstValue(c)
	if err == nil {
		t.Error("expected error for nil constant kind")
	}
}
