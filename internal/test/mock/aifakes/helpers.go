package aifakes

import "strings"

func normalizedLabel(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sameNormalizedOwner(left string, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}
