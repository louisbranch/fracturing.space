package daggerheart

import (
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type contentKeySpec struct {
	Name string
	Kind pagination.CursorValueKind
}

type contentListConfig[T any] struct {
	PageSizeConfig pagination.PageSizeConfig
	OrderByConfig  pagination.OrderByConfig
	FilterFields   contentfilter.Fields
	KeySpec        []contentKeySpec
	KeyFunc        func(T) []pagination.CursorValue
	Resolver       func(T, string) (any, bool)
	FilterHashSeed string
}

type contentListRequest struct {
	PageSize  int32
	PageToken string
	OrderBy   string
	Filter    string
	DomainID  string
}

type contentPage[T any] struct {
	Items             []T
	TotalSize         int
	NextPageToken     string
	PreviousPageToken string
}

func listContentPage[T any](items []T, req contentListRequest, cfg contentListConfig[T]) (contentPage[T], error) {
	pageSize := pagination.ClampPageSize(req.PageSize, cfg.PageSizeConfig)

	orderByInput := strings.TrimSpace(req.OrderBy)
	orderBy, err := pagination.NormalizeOrderBy(orderByInput, cfg.OrderByConfig)
	if err != nil {
		return contentPage[T]{}, err
	}
	descending := strings.HasSuffix(orderBy, " desc")

	filterStr := strings.TrimSpace(req.Filter)
	filterExpr, err := contentfilter.Parse(filterStr, cfg.FilterFields)
	if err != nil {
		return contentPage[T]{}, err
	}

	filtered, err := applyContentFilter(items, filterExpr, cfg.Resolver)
	if err != nil {
		return contentPage[T]{}, err
	}

	filterHashInput := filterStr
	if cfg.FilterHashSeed != "" {
		filterHashInput = filterHashInput + "\n" + cfg.FilterHashSeed
	}

	var cursor *pagination.Cursor
	var cursorKeys []pagination.CursorValue
	pageToken := strings.TrimSpace(req.PageToken)
	if pageToken != "" {
		decoded, err := pagination.Decode(pageToken)
		if err != nil {
			return contentPage[T]{}, err
		}
		if err := pagination.ValidateFilterHash(decoded, filterHashInput); err != nil {
			return contentPage[T]{}, err
		}
		if err := pagination.ValidateOrderHash(decoded, orderBy); err != nil {
			return contentPage[T]{}, err
		}
		cursor = &decoded
		cursorKeys, err = cursorKeysFromToken(decoded, cfg.KeySpec)
		if err != nil {
			return contentPage[T]{}, err
		}
	}

	pageItems, hasNext, hasPrev, err := paginateContent(filtered, cursor, cursorKeys, descending, pageSize, cfg.KeyFunc)
	if err != nil {
		return contentPage[T]{}, err
	}

	response := contentPage[T]{
		Items:     pageItems,
		TotalSize: len(filtered),
	}
	if len(pageItems) == 0 {
		return response, nil
	}

	if hasNext {
		last := pageItems[len(pageItems)-1]
		nextCursor := pagination.NewNextPageCursor(cfg.KeyFunc(last), descending, filterHashInput, orderBy)
		token, err := pagination.Encode(nextCursor)
		if err != nil {
			return contentPage[T]{}, err
		}
		response.NextPageToken = token
	}
	if hasPrev {
		first := pageItems[0]
		prevCursor := pagination.NewPrevPageCursor(cfg.KeyFunc(first), descending, filterHashInput, orderBy)
		token, err := pagination.Encode(prevCursor)
		if err != nil {
			return contentPage[T]{}, err
		}
		response.PreviousPageToken = token
	}

	return response, nil
}

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

func paginateContent[T any](items []T, cursor *pagination.Cursor, cursorKeys []pagination.CursorValue, descending bool, pageSize int, keyFunc func(T) []pagination.CursorValue) ([]T, bool, bool, error) {
	filtered := items
	if cursor != nil {
		filtered = make([]T, 0, len(items))
		for _, item := range items {
			key := keyFunc(item)
			cmp, err := compareCursorValues(key, cursorKeys)
			if err != nil {
				return nil, false, false, err
			}
			switch cursor.Dir {
			case pagination.DirectionForward:
				if cmp > 0 {
					filtered = append(filtered, item)
				}
			case pagination.DirectionBackward:
				if cmp < 0 {
					filtered = append(filtered, item)
				}
			default:
				return nil, false, false, fmt.Errorf("invalid cursor direction: %s", cursor.Dir)
			}
		}
	}

	ordered, err := orderItems(filtered, keyFunc, descending)
	if err != nil {
		return nil, false, false, err
	}

	hasMore := len(ordered) > pageSize
	var page []T
	var hasNext bool
	var hasPrev bool

	if cursor != nil && cursor.Reverse {
		if len(ordered) > pageSize {
			page = ordered[len(ordered)-pageSize:]
			hasPrev = true
		} else {
			page = ordered
			hasPrev = false
		}
		hasNext = true
	} else {
		if len(ordered) > pageSize {
			page = ordered[:pageSize]
		} else {
			page = ordered
		}
		hasNext = hasMore
		hasPrev = cursor != nil
	}

	return page, hasNext, hasPrev, nil
}

type keyedItem[T any] struct {
	item T
	key  []pagination.CursorValue
}

func orderItems[T any](items []T, keyFunc func(T) []pagination.CursorValue, descending bool) ([]T, error) {
	if len(items) == 0 {
		return items, nil
	}

	keyed := make([]keyedItem[T], len(items))
	baseKey := keyFunc(items[0])
	for i, item := range items {
		key := keyFunc(item)
		if err := validateKeySpec(baseKey, key); err != nil {
			return nil, err
		}
		keyed[i] = keyedItem[T]{item: item, key: key}
	}

	sort.SliceStable(keyed, func(i, j int) bool {
		cmp, err := compareCursorValues(keyed[i].key, keyed[j].key)
		if err != nil {
			return false
		}
		if descending {
			return cmp > 0
		}
		return cmp < 0
	})

	ordered := make([]T, len(keyed))
	for i, entry := range keyed {
		ordered[i] = entry.item
	}

	return ordered, nil
}

func validateKeySpec(base []pagination.CursorValue, candidate []pagination.CursorValue) error {
	if len(base) != len(candidate) {
		return fmt.Errorf("cursor key length mismatch")
	}
	for i := range base {
		if base[i].Name != candidate[i].Name {
			return fmt.Errorf("cursor key mismatch at %s", base[i].Name)
		}
		if base[i].Kind != candidate[i].Kind {
			return fmt.Errorf("cursor key kind mismatch at %s", base[i].Name)
		}
	}
	return nil
}

func cursorKeysFromToken(c pagination.Cursor, specs []contentKeySpec) ([]pagination.CursorValue, error) {
	keys := make([]pagination.CursorValue, 0, len(specs))
	for _, spec := range specs {
		switch spec.Kind {
		case pagination.CursorValueString:
			value, err := pagination.ValueString(c, spec.Name)
			if err != nil {
				return nil, err
			}
			keys = append(keys, pagination.StringValue(spec.Name, value))
		case pagination.CursorValueInt:
			value, err := pagination.ValueInt(c, spec.Name)
			if err != nil {
				return nil, err
			}
			keys = append(keys, pagination.IntValue(spec.Name, value))
		case pagination.CursorValueUint:
			value, err := pagination.ValueUint(c, spec.Name)
			if err != nil {
				return nil, err
			}
			keys = append(keys, pagination.UintValue(spec.Name, value))
		default:
			return nil, fmt.Errorf("unsupported cursor key kind for %s", spec.Name)
		}
	}
	return keys, nil
}

func compareCursorValues(left []pagination.CursorValue, right []pagination.CursorValue) (int, error) {
	if len(left) != len(right) {
		return 0, fmt.Errorf("cursor key length mismatch")
	}
	for i := range left {
		cmp, err := compareCursorValue(left[i], right[i])
		if err != nil {
			return 0, err
		}
		if cmp != 0 {
			return cmp, nil
		}
	}
	return 0, nil
}

func compareCursorValue(left pagination.CursorValue, right pagination.CursorValue) (int, error) {
	if left.Kind != right.Kind {
		return 0, fmt.Errorf("cursor value kind mismatch for %s", left.Name)
	}
	if left.Kind == pagination.CursorValueString {
		return compareStrings(left.StringValue, right.StringValue), nil
	}
	if left.Kind == pagination.CursorValueInt {
		return compareInts(left.IntValue, right.IntValue), nil
	}
	if left.Kind == pagination.CursorValueUint {
		return compareUints(left.UintValue, right.UintValue), nil
	}
	return 0, fmt.Errorf("unsupported cursor value kind for %s", left.Name)
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

func compareInts(left, right int64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareUints(left, right uint64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
