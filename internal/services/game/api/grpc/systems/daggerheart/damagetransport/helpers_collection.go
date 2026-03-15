package damagetransport

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

func containsString(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringsToCharacterIDs(values []string) []ids.CharacterID {
	if len(values) == 0 {
		return nil
	}
	result := make([]ids.CharacterID, len(values))
	for i, value := range values {
		result[i] = ids.CharacterID(value)
	}
	return result
}
