package app

import (
	"sort"
	"strings"
)

// sortByName sorts items by a name key with ID tiebreaker.
func sortByName[T any](items []T, nameOf func(T) string, idOf func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(nameOf(items[i])))
		right := strings.ToLower(strings.TrimSpace(nameOf(items[j])))
		if left == right {
			return strings.TrimSpace(idOf(items[i])) < strings.TrimSpace(idOf(items[j]))
		}
		return left < right
	})
}
