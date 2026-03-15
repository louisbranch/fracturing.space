package contenttransport

import (
	"fmt"
	"sort"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
)

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
