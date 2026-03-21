package filter

import (
	"fmt"
	"strings"

	"go.einride.tech/aip/filtering"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// FieldType describes a supported filter field type.
type FieldType string

const (
	FieldString FieldType = "string"
	FieldInt    FieldType = "int"
)

// Fields defines filterable fields and their types.
type Fields map[string]FieldType

// Parse parses an AIP-160 filter expression for the provided fields.
func Parse(filterStr string, fields Fields) (*expr.Expr, error) {
	if strings.TrimSpace(filterStr) == "" {
		return nil, nil
	}

	decls, err := declarations(fields)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilterString(filterStr, decls)
	if err != nil {
		return nil, fmt.Errorf("parse filter: %w", err)
	}

	return filter.CheckedExpr.Expr, nil
}

func declarations(fields Fields) (*filtering.Declarations, error) {
	decls := []filtering.DeclarationOption{filtering.DeclareStandardFunctions()}
	for name, kind := range fields {
		switch kind {
		case FieldString:
			decls = append(decls, filtering.DeclareIdent(name, filtering.TypeString))
		case FieldInt:
			decls = append(decls, filtering.DeclareIdent(name, filtering.TypeInt))
		default:
			return nil, fmt.Errorf("unsupported field type for %s", name)
		}
	}

	return filtering.NewDeclarations(decls...)
}
