package contenttransport

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
)

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
