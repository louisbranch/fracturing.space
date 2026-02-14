package filter

import (
	"reflect"
	"strings"
	"testing"

	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func TestParseEventFilter_TypeEquals(t *testing.T) {
	cond, err := ParseEventFilter(`type = "session.started"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "event_type = ?" {
		t.Errorf("expected 'event_type = ?', got %q", cond.Clause)
	}
	if len(cond.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(cond.Params))
	}
	if cond.Params[0] != "session.started" {
		t.Errorf("expected 'session.started', got %v", cond.Params[0])
	}
}

func TestParseEventFilter_Empty(t *testing.T) {
	cond, err := ParseEventFilter(" ")
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "" || cond.Params != nil {
		t.Fatalf("expected empty condition, got %+v", cond)
	}
}

func TestParseEventFilter_AndOr(t *testing.T) {
	cond, err := ParseEventFilter(`type = "session.started" AND actor_type = "gm"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "(event_type = ? AND actor_type = ?)" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
	if !reflect.DeepEqual(cond.Params, []any{"session.started", "gm"}) {
		t.Fatalf("Params = %v", cond.Params)
	}

	cond, err = ParseEventFilter(`actor_type = "gm" OR actor_type = "participant"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "(actor_type = ? OR actor_type = ?)" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
}

func TestParseEventFilter_NotEqualsAndNumeric(t *testing.T) {
	cond, err := ParseEventFilter(`actor_id != "p1"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "actor_id != ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}

	cond, err = ParseEventFilter(`ts > timestamp("2025-01-01T00:00:00Z")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "timestamp > ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
	if len(cond.Params) != 1 {
		t.Fatalf("Params len = %d", len(cond.Params))
	}
	if !strings.HasPrefix(cond.Params[0].(string), "2025-01-01T00:00:00") {
		t.Fatalf("timestamp param = %v", cond.Params[0])
	}
}

func TestParseEventFilter_InvalidField(t *testing.T) {
	_, err := ParseEventFilter(`unknown = "x"`)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestParseEventFilter_InvalidValueFunc(t *testing.T) {
	_, err := ParseEventFilter(`ts = duration("1h")`)
	if err == nil {
		t.Fatal("expected error for unsupported value function")
	}
}

func TestParseEventFilter_InvalidTimestamp(t *testing.T) {
	_, err := ParseEventFilter(`ts = timestamp("not-a-time")`)
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestParseEventFilter_LessThan(t *testing.T) {
	cond, err := ParseEventFilter(`ts < timestamp("2025-06-01T00:00:00Z")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "timestamp < ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
}

func TestParseEventFilter_LessEquals(t *testing.T) {
	cond, err := ParseEventFilter(`ts <= timestamp("2025-06-01T00:00:00Z")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "timestamp <= ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
}

func TestParseEventFilter_GreaterEquals(t *testing.T) {
	cond, err := ParseEventFilter(`ts >= timestamp("2025-06-01T00:00:00Z")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "timestamp >= ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
}

func TestParseEventFilter_AllFields(t *testing.T) {
	fields := map[string]string{
		"session_id":     "session_id",
		"type":           "event_type",
		"actor_type":     "actor_type",
		"actor_id":       "actor_id",
		"system_id":      "system_id",
		"system_version": "system_version",
		"entity_type":    "entity_type",
		"entity_id":      "entity_id",
	}
	for field, col := range fields {
		cond, err := ParseEventFilter(field + ` = "test"`)
		if err != nil {
			t.Fatalf("parse filter for %s: %v", field, err)
		}
		expected := col + " = ?"
		if cond.Clause != expected {
			t.Fatalf("field %s: expected %q, got %q", field, expected, cond.Clause)
		}
	}
}

func TestParseEventFilter_RFC3339Nano(t *testing.T) {
	cond, err := ParseEventFilter(`ts = timestamp("2025-01-01T00:00:00.123456789Z")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if !strings.Contains(cond.Params[0].(string), "2025-01-01T00:00:00") {
		t.Fatalf("expected RFC3339 formatted timestamp, got %v", cond.Params[0])
	}
}

func TestParseEventFilter_InvalidExpression(t *testing.T) {
	_, err := ParseEventFilter(`not valid syntax +++`)
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
}

func TestEventDeclarations(t *testing.T) {
	decls, err := EventDeclarations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decls == nil {
		t.Fatal("expected non-nil declarations")
	}
}

func TestParseEventFilter_NestedAndOr(t *testing.T) {
	cond, err := ParseEventFilter(`(type = "a" AND actor_type = "b") OR (type = "c" AND actor_type = "d")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "((event_type = ? AND actor_type = ?) OR (event_type = ? AND actor_type = ?))" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
	if len(cond.Params) != 4 {
		t.Fatalf("expected 4 params, got %d", len(cond.Params))
	}
}

func TestParseEventFilter_LiteralAsField(t *testing.T) {
	_, err := ParseEventFilter(`"literal" = "value"`)
	if err == nil {
		t.Fatal("expected error for non-identifier field")
	}
}

func TestParseEventFilter_TimestampWrongArgCount(t *testing.T) {
	_, err := ParseEventFilter(`ts = timestamp("2025-01-01T00:00:00Z", "extra")`)
	if err == nil {
		t.Fatal("expected error for timestamp with extra arguments")
	}
}

func TestParseEventFilter_TimestampNoOffset(t *testing.T) {
	_, err := ParseEventFilter(`ts = timestamp("2025-01-01T00:00:00")`)
	if err == nil {
		t.Fatal("expected error for timestamp without timezone")
	}
}

func TestParseEventFilter_TimestampWithOffset(t *testing.T) {
	cond, err := ParseEventFilter(`ts = timestamp("2025-01-01T00:00:00+05:00")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if len(cond.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(cond.Params))
	}
}

func TestParseEventFilter_SystemVersionField(t *testing.T) {
	cond, err := ParseEventFilter(`system_version = "1.0"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "system_version = ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
}

// --- Internal function tests for uncovered branches ---

func TestTranslateExpr_Nil(t *testing.T) {
	cond, err := translateExpr(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cond.Clause != "" {
		t.Fatalf("expected empty clause, got %q", cond.Clause)
	}
}

func TestTranslateExpr_UnsupportedType(t *testing.T) {
	e := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	_, err := translateExpr(e)
	if err == nil {
		t.Fatal("expected error for unsupported expression type")
	}
}

func TestTranslateAnd_WrongArgCount(t *testing.T) {
	_, err := translateAnd(nil)
	if err == nil {
		t.Fatal("expected error for AND with 0 arguments")
	}
}

func TestTranslateOr_WrongArgCount(t *testing.T) {
	_, err := translateOr(nil)
	if err == nil {
		t.Fatal("expected error for OR with 0 arguments")
	}
}

func TestTranslateComparison_WrongArgCount(t *testing.T) {
	_, err := translateComparison(nil, "=")
	if err == nil {
		t.Fatal("expected error for comparison with 0 arguments")
	}
}

func TestExtractFieldName_Nil(t *testing.T) {
	_, err := extractFieldName(nil)
	if err == nil {
		t.Fatal("expected error for nil expression")
	}
}

func TestExtractValue_Nil(t *testing.T) {
	_, err := extractValue(nil)
	if err == nil {
		t.Fatal("expected error for nil expression")
	}
}

func TestExtractValue_UnsupportedKind(t *testing.T) {
	e := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	_, err := extractValue(e)
	if err == nil {
		t.Fatal("expected error for unsupported kind in value position")
	}
}

func TestExtractConstValue_Nil(t *testing.T) {
	_, err := extractConstValue(nil)
	if err == nil {
		t.Fatal("expected error for nil constant")
	}
}

func TestExtractConstValue_Types(t *testing.T) {
	tests := []struct {
		name string
		c    *expr.Constant
		want any
	}{
		{"int64", &expr.Constant{ConstantKind: &expr.Constant_Int64Value{Int64Value: 42}}, int64(42)},
		{"uint64", &expr.Constant{ConstantKind: &expr.Constant_Uint64Value{Uint64Value: 42}}, uint64(42)},
		{"double", &expr.Constant{ConstantKind: &expr.Constant_DoubleValue{DoubleValue: 3.14}}, 3.14},
		{"bool", &expr.Constant{ConstantKind: &expr.Constant_BoolValue{BoolValue: true}}, true},
		{"string", &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "hello"}}, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractConstValue(tt.c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestExtractConstValue_UnsupportedType(t *testing.T) {
	c := &expr.Constant{ConstantKind: &expr.Constant_NullValue{}}
	_, err := extractConstValue(c)
	if err == nil {
		t.Fatal("expected error for unsupported constant type")
	}
}

func TestExtractTimestampValue_Nil(t *testing.T) {
	_, err := extractTimestampValue(nil)
	if err == nil {
		t.Fatal("expected error for nil timestamp argument")
	}
}

func TestExtractTimestampValue_NonConst(t *testing.T) {
	e := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	_, err := extractTimestampValue(e)
	if err == nil {
		t.Fatal("expected error for non-constant timestamp argument")
	}
}

func TestExtractTimestampValue_NonStringConst(t *testing.T) {
	e := &expr.Expr{ExprKind: &expr.Expr_ConstExpr{
		ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_Int64Value{Int64Value: 42}},
	}}
	_, err := extractTimestampValue(e)
	if err == nil {
		t.Fatal("expected error for non-string constant timestamp")
	}
}

func TestTranslateCall_UnsupportedFunction(t *testing.T) {
	call := &expr.Expr_Call{Function: "unsupported_func"}
	_, err := translateCall(call)
	if err == nil {
		t.Fatal("expected error for unsupported function")
	}
}

func TestTranslateAnd_RightSideError(t *testing.T) {
	good := &expr.Expr{ExprKind: &expr.Expr_CallExpr{CallExpr: &expr.Expr_Call{
		Function: "_==_",
		Args: []*expr.Expr{
			{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "type"}}},
			{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "a"}}}},
		},
	}}}
	bad := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	_, err := translateAnd([]*expr.Expr{good, bad})
	if err == nil {
		t.Fatal("expected error for right-side error in AND")
	}
}

func TestTranslateAnd_LeftSideError(t *testing.T) {
	bad := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	good := &expr.Expr{ExprKind: &expr.Expr_CallExpr{CallExpr: &expr.Expr_Call{
		Function: "_==_",
		Args: []*expr.Expr{
			{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "type"}}},
			{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "a"}}}},
		},
	}}}
	_, err := translateAnd([]*expr.Expr{bad, good})
	if err == nil {
		t.Fatal("expected error for left-side error in AND")
	}
}

func TestTranslateOr_RightSideError(t *testing.T) {
	good := &expr.Expr{ExprKind: &expr.Expr_CallExpr{CallExpr: &expr.Expr_Call{
		Function: "_==_",
		Args: []*expr.Expr{
			{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "type"}}},
			{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "a"}}}},
		},
	}}}
	bad := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	_, err := translateOr([]*expr.Expr{good, bad})
	if err == nil {
		t.Fatal("expected error for right-side error in OR")
	}
}

func TestTranslateComparison_ExtractValueError(t *testing.T) {
	args := []*expr.Expr{
		{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "type"}}},
		{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "not_a_value"}}},
	}
	_, err := translateComparison(args, "=")
	if err == nil {
		t.Fatal("expected error for extractValue failure")
	}
}

func TestTranslateOr_LeftSideError(t *testing.T) {
	bad := &expr.Expr{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "x"}}}
	good := &expr.Expr{ExprKind: &expr.Expr_CallExpr{CallExpr: &expr.Expr_Call{
		Function: "_==_",
		Args: []*expr.Expr{
			{ExprKind: &expr.Expr_IdentExpr{IdentExpr: &expr.Expr_Ident{Name: "type"}}},
			{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "a"}}}},
		},
	}}}
	_, err := translateOr([]*expr.Expr{bad, good})
	if err == nil {
		t.Fatal("expected error for left-side error in OR")
	}
}

func TestExtractValue_TimestampWrongArgCount(t *testing.T) {
	e := &expr.Expr{ExprKind: &expr.Expr_CallExpr{CallExpr: &expr.Expr_Call{
		Function: "timestamp",
		Args: []*expr.Expr{
			{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "a"}}}},
			{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "b"}}}},
		},
	}}}
	_, err := extractValue(e)
	if err == nil {
		t.Fatal("expected error for timestamp with wrong arg count")
	}
}

func TestTranslateComparison_ExtractFieldError(t *testing.T) {
	args := []*expr.Expr{
		{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "oops"}}}},
		{ExprKind: &expr.Expr_ConstExpr{ConstExpr: &expr.Constant{ConstantKind: &expr.Constant_StringValue{StringValue: "val"}}}},
	}
	_, err := translateComparison(args, "=")
	if err == nil {
		t.Fatal("expected error for extractFieldName failure")
	}
}
