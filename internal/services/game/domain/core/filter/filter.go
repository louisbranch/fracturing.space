// Package filter provides AIP-160 filter expression parsing and SQL translation.
package filter

import (
	"fmt"
	"strings"
	"time"

	"go.einride.tech/aip/filtering"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// EventDeclarations returns the field declarations for event filtering.
func EventDeclarations() (*filtering.Declarations, error) {
	return filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("session_id", filtering.TypeString),
		filtering.DeclareIdent("type", filtering.TypeString),
		filtering.DeclareIdent("actor_type", filtering.TypeString),
		filtering.DeclareIdent("actor_id", filtering.TypeString),
		filtering.DeclareIdent("system_id", filtering.TypeString),
		filtering.DeclareIdent("system_version", filtering.TypeString),
		filtering.DeclareIdent("entity_type", filtering.TypeString),
		filtering.DeclareIdent("entity_id", filtering.TypeString),
		filtering.DeclareIdent("ts", filtering.TypeTimestamp),
	)
}

// SQLCondition represents a SQL WHERE clause fragment with parameters.
type SQLCondition struct {
	// Clause is the SQL WHERE clause (e.g., "session_id = ?").
	Clause string
	// Params are the positional parameters for the clause.
	Params []any
}

// fieldMapping maps filter field names to SQL column names.
var fieldMapping = map[string]string{
	"session_id":     "session_id",
	"type":           "event_type",
	"actor_type":     "actor_type",
	"actor_id":       "actor_id",
	"system_id":      "system_id",
	"system_version": "system_version",
	"entity_type":    "entity_type",
	"entity_id":      "entity_id",
	"ts":             "timestamp",
}

// ParseEventFilter parses an AIP-160 filter expression and returns a SQL condition.
// Returns an empty condition for an empty filter string.
func ParseEventFilter(filterStr string) (SQLCondition, error) {
	if strings.TrimSpace(filterStr) == "" {
		return SQLCondition{}, nil
	}

	decls, err := EventDeclarations()
	if err != nil {
		return SQLCondition{}, fmt.Errorf("create declarations: %w", err)
	}

	filter, err := filtering.ParseFilterString(filterStr, decls)
	if err != nil {
		return SQLCondition{}, fmt.Errorf("parse filter: %w", err)
	}

	return translateExpr(filter.CheckedExpr.Expr)
}

// translateExpr translates a CEL expression to a SQL condition.
func translateExpr(e *expr.Expr) (SQLCondition, error) {
	if e == nil {
		return SQLCondition{}, nil
	}

	switch kind := e.ExprKind.(type) {
	case *expr.Expr_CallExpr:
		return translateCall(kind.CallExpr)
	default:
		return SQLCondition{}, fmt.Errorf("unsupported expression type: %T", kind)
	}
}

// translateCall translates a CEL function call to a SQL condition.
func translateCall(call *expr.Expr_Call) (SQLCondition, error) {
	switch call.Function {
	case "_&&_", "AND":
		return translateAnd(call.Args)
	case "_||_", "OR":
		return translateOr(call.Args)
	case "_==_", "=":
		return translateEquals(call.Args)
	case "_!=_", "!=":
		return translateNotEquals(call.Args)
	case "_<_", "<":
		return translateLessThan(call.Args)
	case "_<=_", "<=":
		return translateLessEquals(call.Args)
	case "_>_", ">":
		return translateGreaterThan(call.Args)
	case "_>=_", ">=":
		return translateGreaterEquals(call.Args)
	default:
		return SQLCondition{}, fmt.Errorf("unsupported function: %s", call.Function)
	}
}

func translateAnd(args []*expr.Expr) (SQLCondition, error) {
	if len(args) != 2 {
		return SQLCondition{}, fmt.Errorf("AND requires 2 arguments")
	}

	left, err := translateExpr(args[0])
	if err != nil {
		return SQLCondition{}, err
	}

	right, err := translateExpr(args[1])
	if err != nil {
		return SQLCondition{}, err
	}

	return SQLCondition{
		Clause: fmt.Sprintf("(%s AND %s)", left.Clause, right.Clause),
		Params: append(left.Params, right.Params...),
	}, nil
}

func translateOr(args []*expr.Expr) (SQLCondition, error) {
	if len(args) != 2 {
		return SQLCondition{}, fmt.Errorf("OR requires 2 arguments")
	}

	left, err := translateExpr(args[0])
	if err != nil {
		return SQLCondition{}, err
	}

	right, err := translateExpr(args[1])
	if err != nil {
		return SQLCondition{}, err
	}

	return SQLCondition{
		Clause: fmt.Sprintf("(%s OR %s)", left.Clause, right.Clause),
		Params: append(left.Params, right.Params...),
	}, nil
}

func translateEquals(args []*expr.Expr) (SQLCondition, error) {
	return translateComparison(args, "=")
}

func translateNotEquals(args []*expr.Expr) (SQLCondition, error) {
	return translateComparison(args, "!=")
}

func translateLessThan(args []*expr.Expr) (SQLCondition, error) {
	return translateComparison(args, "<")
}

func translateLessEquals(args []*expr.Expr) (SQLCondition, error) {
	return translateComparison(args, "<=")
}

func translateGreaterThan(args []*expr.Expr) (SQLCondition, error) {
	return translateComparison(args, ">")
}

func translateGreaterEquals(args []*expr.Expr) (SQLCondition, error) {
	return translateComparison(args, ">=")
}

func translateComparison(args []*expr.Expr, op string) (SQLCondition, error) {
	if len(args) != 2 {
		return SQLCondition{}, fmt.Errorf("comparison requires 2 arguments")
	}

	field, err := extractFieldName(args[0])
	if err != nil {
		return SQLCondition{}, err
	}

	column, ok := fieldMapping[field]
	if !ok {
		return SQLCondition{}, fmt.Errorf("unknown field: %s", field)
	}

	value, err := extractValue(args[1])
	if err != nil {
		return SQLCondition{}, err
	}

	return SQLCondition{
		Clause: fmt.Sprintf("%s %s ?", column, op),
		Params: []any{value},
	}, nil
}

func extractFieldName(e *expr.Expr) (string, error) {
	if e == nil {
		return "", fmt.Errorf("nil expression")
	}

	switch kind := e.ExprKind.(type) {
	case *expr.Expr_IdentExpr:
		return kind.IdentExpr.Name, nil
	default:
		return "", fmt.Errorf("expected identifier, got %T", kind)
	}
}

func extractValue(e *expr.Expr) (any, error) {
	if e == nil {
		return nil, fmt.Errorf("nil expression")
	}

	switch kind := e.ExprKind.(type) {
	case *expr.Expr_ConstExpr:
		return extractConstValue(kind.ConstExpr)
	case *expr.Expr_CallExpr:
		// Handle timestamp("...") function calls
		if kind.CallExpr.Function == "timestamp" && len(kind.CallExpr.Args) == 1 {
			return extractTimestampValue(kind.CallExpr.Args[0])
		}
		return nil, fmt.Errorf("unsupported function in value position: %s", kind.CallExpr.Function)
	default:
		return nil, fmt.Errorf("expected constant or timestamp, got %T", kind)
	}
}

func extractConstValue(c *expr.Constant) (any, error) {
	if c == nil {
		return nil, fmt.Errorf("nil constant")
	}

	switch kind := c.ConstantKind.(type) {
	case *expr.Constant_StringValue:
		return kind.StringValue, nil
	case *expr.Constant_Int64Value:
		return kind.Int64Value, nil
	case *expr.Constant_Uint64Value:
		return kind.Uint64Value, nil
	case *expr.Constant_DoubleValue:
		return kind.DoubleValue, nil
	case *expr.Constant_BoolValue:
		return kind.BoolValue, nil
	default:
		return nil, fmt.Errorf("unsupported constant type: %T", kind)
	}
}

func extractTimestampValue(e *expr.Expr) (string, error) {
	if e == nil {
		return "", fmt.Errorf("nil timestamp argument")
	}

	switch kind := e.ExprKind.(type) {
	case *expr.Expr_ConstExpr:
		if strVal, ok := kind.ConstExpr.ConstantKind.(*expr.Constant_StringValue); ok {
			// Parse and reformat to ensure consistent format
			t, err := time.Parse(time.RFC3339, strVal.StringValue)
			if err != nil {
				// Try RFC3339Nano
				t, err = time.Parse(time.RFC3339Nano, strVal.StringValue)
				if err != nil {
					return "", fmt.Errorf("invalid timestamp format: %s", strVal.StringValue)
				}
			}
			// Return in RFC3339Nano format for SQLite storage compatibility
			return t.UTC().Format(time.RFC3339Nano), nil
		}
		return "", fmt.Errorf("timestamp argument must be a string")
	default:
		return "", fmt.Errorf("timestamp argument must be a constant string")
	}
}
