package filter

import (
	"fmt"

	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// Resolver returns a value for a field name.
type Resolver func(name string) (any, bool)

// Evaluate evaluates a parsed filter expression against a resolver.
func Evaluate(e *expr.Expr, resolve Resolver) (bool, error) {
	if e == nil {
		return true, nil
	}

	switch kind := e.ExprKind.(type) {
	case *expr.Expr_CallExpr:
		return evalCall(kind.CallExpr, resolve)
	default:
		return false, fmt.Errorf("unsupported expression type: %T", kind)
	}
}

func evalCall(call *expr.Expr_Call, resolve Resolver) (bool, error) {
	switch call.Function {
	case "_&&_", "AND":
		return evalAnd(call.Args, resolve)
	case "_||_", "OR":
		return evalOr(call.Args, resolve)
	case "_==_", "=":
		return evalCompare(call.Args, resolve, "=")
	case "_!=_", "!=":
		return evalCompare(call.Args, resolve, "!=")
	case "_<_", "<":
		return evalCompare(call.Args, resolve, "<")
	case "_<=_", "<=":
		return evalCompare(call.Args, resolve, "<=")
	case "_>_", ">":
		return evalCompare(call.Args, resolve, ">")
	case "_>=_", ">=":
		return evalCompare(call.Args, resolve, ">=")
	default:
		return false, fmt.Errorf("unsupported function: %s", call.Function)
	}
}

func evalAnd(args []*expr.Expr, resolve Resolver) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("AND requires 2 arguments")
	}
	left, err := Evaluate(args[0], resolve)
	if err != nil || !left {
		return left, err
	}
	return Evaluate(args[1], resolve)
}

func evalOr(args []*expr.Expr, resolve Resolver) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("OR requires 2 arguments")
	}
	left, err := Evaluate(args[0], resolve)
	if err != nil {
		return false, err
	}
	if left {
		return true, nil
	}
	return Evaluate(args[1], resolve)
}

func evalCompare(args []*expr.Expr, resolve Resolver, op string) (bool, error) {
	if len(args) != 2 {
		return false, fmt.Errorf("comparison requires 2 arguments")
	}

	field, err := extractFieldName(args[0])
	if err != nil {
		return false, err
	}

	left, ok := resolve(field)
	if !ok {
		return false, fmt.Errorf("unknown field: %s", field)
	}

	right, err := extractValue(args[1])
	if err != nil {
		return false, err
	}

	cmp, err := compareValues(left, right)
	if err != nil {
		return false, err
	}

	switch op {
	case "=":
		return cmp == 0, nil
	case "!=":
		return cmp != 0, nil
	case "<":
		return cmp < 0, nil
	case "<=":
		return cmp <= 0, nil
	case ">":
		return cmp > 0, nil
	case ">=":
		return cmp >= 0, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", op)
	}
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
	default:
		return nil, fmt.Errorf("expected constant, got %T", kind)
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

func compareValues(left any, right any) (int, error) {
	switch l := left.(type) {
	case string:
		r, ok := right.(string)
		if !ok {
			return 0, fmt.Errorf("type mismatch: string vs %T", right)
		}
		return compareStrings(l, r), nil
	case int:
		return compareNumbers(float64(l), right)
	case int32:
		return compareNumbers(float64(l), right)
	case int64:
		return compareNumbers(float64(l), right)
	case uint:
		return compareNumbers(float64(l), right)
	case uint32:
		return compareNumbers(float64(l), right)
	case uint64:
		return compareNumbers(float64(l), right)
	case float32:
		return compareNumbers(float64(l), right)
	case float64:
		return compareNumbers(l, right)
	case bool:
		r, ok := right.(bool)
		if !ok {
			return 0, fmt.Errorf("type mismatch: bool vs %T", right)
		}
		return compareBools(l, r), nil
	default:
		return 0, fmt.Errorf("unsupported value type: %T", left)
	}
}

func compareNumbers(left float64, right any) (int, error) {
	r, ok := toFloat(right)
	if !ok {
		return 0, fmt.Errorf("type mismatch: number vs %T", right)
	}
	switch {
	case left < r:
		return -1, nil
	case left > r:
		return 1, nil
	default:
		return 0, nil
	}
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func compareStrings(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareBools(left, right bool) int {
	if left == right {
		return 0
	}
	if !left && right {
		return -1
	}
	return 1
}
