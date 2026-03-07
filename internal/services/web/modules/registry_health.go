package modules

import "strings"

// capitalizeLabel capitalizes the first character of a module or service ID
// for display in health entries.
func capitalizeLabel(id string) string {
	if id == "" {
		return id
	}
	return strings.ToUpper(id[:1]) + id[1:]
}
