package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// testStringsToCharacterIDs keeps payload setup local to the root tests after
// the production helper moved into damagetransport.
func testStringsToCharacterIDs(values []string) []ids.CharacterID {
	if len(values) == 0 {
		return nil
	}
	result := make([]ids.CharacterID, len(values))
	for i, value := range values {
		result[i] = ids.CharacterID(value)
	}
	return result
}
