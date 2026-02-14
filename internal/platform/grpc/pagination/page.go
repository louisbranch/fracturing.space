package pagination

import "fmt"

// PageSizeConfig configures page size normalization.
type PageSizeConfig struct {
	Default int
	Max     int
}

// OrderByConfig configures order_by validation.
type OrderByConfig struct {
	Default string
	Allowed []string
}

// ClampPageSize applies defaults and limits for page sizes.
func ClampPageSize(value int32, cfg PageSizeConfig) int {
	pageSize := int(value)
	if pageSize <= 0 {
		pageSize = cfg.Default
	}
	if cfg.Max > 0 && pageSize > cfg.Max {
		pageSize = cfg.Max
	}
	if pageSize <= 0 {
		pageSize = 1
	}
	return pageSize
}

// NormalizeOrderBy validates order_by and applies defaults.
func NormalizeOrderBy(orderBy string, cfg OrderByConfig) (string, error) {
	if orderBy == "" {
		return cfg.Default, nil
	}
	for _, allowed := range cfg.Allowed {
		if orderBy == allowed {
			return orderBy, nil
		}
	}
	return "", fmt.Errorf("invalid order_by: %s", orderBy)
}
