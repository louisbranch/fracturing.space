package contenttransport

import (
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func applyContentFilter[T any](items []T, filterExpr *expr.Expr, resolver func(T, string) (any, bool)) ([]T, error) {
	if filterExpr == nil {
		return items, nil
	}

	filtered := make([]T, 0, len(items))
	for _, item := range items {
		match, err := contentfilter.Evaluate(filterExpr, func(name string) (any, bool) {
			return resolver(item, name)
		})
		if err != nil {
			return nil, err
		}
		if match {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}
