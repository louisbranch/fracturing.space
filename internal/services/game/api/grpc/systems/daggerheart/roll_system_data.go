package daggerheart

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

// SystemData key constants for daggerheart roll payloads.
const (
	sdKeyCharacterID = "character_id"
	sdKeyAdversaryID = "adversary_id"
	sdKeyRollKind    = "roll_kind"
	sdKeyOutcome     = "outcome"
	sdKeyHopeFear    = "hope_fear"
	sdKeyCrit        = "crit"
	sdKeyCritNegates = "crit_negates"
	sdKeyRoll        = "roll"
	sdKeyModifier    = "modifier"
	sdKeyTotal       = "total"
)

// decodeRollSystemMetadata decodes and type-checks the roll system_data payload.
func decodeRollSystemMetadata(systemData map[string]any) (rollSystemMetadata, error) {
	metadata := rollSystemMetadata{}
	if len(systemData) == 0 {
		return metadata, nil
	}

	var err error
	if metadata.CharacterID, err = decodeOptionalStringField(systemData, sdKeyCharacterID); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.AdversaryID, err = decodeOptionalStringField(systemData, sdKeyAdversaryID); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Trait, err = decodeOptionalStringField(systemData, "trait"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.RollKind, err = decodeOptionalStringField(systemData, sdKeyRollKind); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Outcome, err = decodeOptionalStringField(systemData, sdKeyOutcome); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Flavor, err = decodeOptionalStringField(systemData, "flavor"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.BreathCountdownID, err = decodeOptionalStringField(systemData, "breath_countdown_id"); err != nil {
		return rollSystemMetadata{}, err
	}

	if metadata.HopeFear, err = decodeOptionalBoolField(systemData, sdKeyHopeFear); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Crit, err = decodeOptionalBoolField(systemData, sdKeyCrit); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.CritNegates, err = decodeOptionalBoolField(systemData, sdKeyCritNegates); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.GMMove, err = decodeOptionalBoolField(systemData, "gm_move"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Underwater, err = decodeOptionalBoolField(systemData, "underwater"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Critical, err = decodeOptionalBoolField(systemData, "critical"); err != nil {
		return rollSystemMetadata{}, err
	}

	if metadata.Roll, err = decodeOptionalIntField(systemData, sdKeyRoll); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Modifier, err = decodeOptionalIntField(systemData, sdKeyModifier); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Total, err = decodeOptionalIntField(systemData, sdKeyTotal); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.BaseTotal, err = decodeOptionalIntField(systemData, "base_total"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.CriticalBonus, err = decodeOptionalIntField(systemData, "critical_bonus"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Advantage, err = decodeOptionalIntField(systemData, "advantage"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Disadvantage, err = decodeOptionalIntField(systemData, "disadvantage"); err != nil {
		return rollSystemMetadata{}, err
	}
	if metadata.Modifiers, err = decodeOptionalModifierList(systemData, "modifiers"); err != nil {
		return rollSystemMetadata{}, err
	}

	return metadata, nil
}

func decodeOptionalStringField(systemData map[string]any, key string) (string, error) {
	value, ok := systemData[key]
	if !ok || value == nil {
		return "", nil
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("system_data.%s must be string", key)
	}
	return strings.TrimSpace(stringValue), nil
}

func decodeOptionalBoolField(systemData map[string]any, key string) (*bool, error) {
	value, ok := systemData[key]
	if !ok || value == nil {
		return nil, nil
	}
	boolValue, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf("system_data.%s must be bool", key)
	}
	return boolPtr(boolValue), nil
}

func decodeOptionalIntField(systemData map[string]any, key string) (*int, error) {
	value, ok := systemData[key]
	if !ok || value == nil {
		return nil, nil
	}
	decoded, err := decodeIntValue(value)
	if err != nil {
		return nil, fmt.Errorf("system_data.%s %w", key, err)
	}
	return intPtrValue(decoded), nil
}

func decodeIntValue(value any) (int, error) {
	switch decoded := value.(type) {
	case int:
		return decoded, nil
	case int8:
		return int(decoded), nil
	case int16:
		return int(decoded), nil
	case int32:
		return int(decoded), nil
	case int64:
		if decoded > int64(math.MaxInt) || decoded < int64(math.MinInt) {
			return 0, fmt.Errorf("must fit in int")
		}
		return int(decoded), nil
	case uint:
		if decoded > uint(math.MaxInt) {
			return 0, fmt.Errorf("must fit in int")
		}
		return int(decoded), nil
	case uint8:
		return int(decoded), nil
	case uint16:
		return int(decoded), nil
	case uint32:
		return int(decoded), nil
	case uint64:
		if decoded > uint64(math.MaxInt) {
			return 0, fmt.Errorf("must fit in int")
		}
		return int(decoded), nil
	case float32:
		return decodeFloatInt(float64(decoded))
	case float64:
		return decodeFloatInt(decoded)
	case json.Number:
		asInt, err := decoded.Int64()
		if err != nil {
			return 0, fmt.Errorf("must be integer")
		}
		if asInt > int64(math.MaxInt) || asInt < int64(math.MinInt) {
			return 0, fmt.Errorf("must fit in int")
		}
		return int(asInt), nil
	case string:
		intValue, err := strconv.Atoi(strings.TrimSpace(decoded))
		if err != nil {
			return 0, fmt.Errorf("must be integer")
		}
		return intValue, nil
	default:
		return 0, fmt.Errorf("must be integer")
	}
}

func decodeFloatInt(value float64) (int, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("must be integer")
	}
	if math.Trunc(value) != value {
		return 0, fmt.Errorf("must be integer")
	}
	if value > float64(math.MaxInt) || value < float64(math.MinInt) {
		return 0, fmt.Errorf("must fit in int")
	}
	return int(value), nil
}

func decodeOptionalModifierList(systemData map[string]any, key string) ([]rollModifierMetadata, error) {
	value, ok := systemData[key]
	if !ok || value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case []rollModifierMetadata:
		cloned := make([]rollModifierMetadata, len(typed))
		copy(cloned, typed)
		return cloned, nil
	case []any:
		decoded := make([]rollModifierMetadata, 0, len(typed))
		for idx, entry := range typed {
			modifier, err := decodeModifierEntry(entry, key, idx)
			if err != nil {
				return nil, err
			}
			decoded = append(decoded, modifier)
		}
		return decoded, nil
	case []map[string]any:
		decoded := make([]rollModifierMetadata, 0, len(typed))
		for idx, entry := range typed {
			modifier, err := decodeModifierMap(entry, key, idx)
			if err != nil {
				return nil, err
			}
			decoded = append(decoded, modifier)
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("system_data.%s must be array", key)
	}
}

func decodeModifierEntry(entry any, key string, index int) (rollModifierMetadata, error) {
	switch typed := entry.(type) {
	case rollModifierMetadata:
		return typed, nil
	case map[string]any:
		return decodeModifierMap(typed, key, index)
	default:
		return rollModifierMetadata{}, fmt.Errorf("system_data.%s[%d] must be object", key, index)
	}
}

func decodeModifierMap(entry map[string]any, key string, index int) (rollModifierMetadata, error) {
	rawValue, ok := entry["value"]
	if !ok || rawValue == nil {
		return rollModifierMetadata{}, fmt.Errorf("system_data.%s[%d].value is required", key, index)
	}
	value, err := decodeIntValue(rawValue)
	if err != nil {
		return rollModifierMetadata{}, fmt.Errorf("system_data.%s[%d].value %w", key, index, err)
	}

	source := ""
	if rawSource, ok := entry["source"]; ok && rawSource != nil {
		sourceValue, ok := rawSource.(string)
		if !ok {
			return rollModifierMetadata{}, fmt.Errorf("system_data.%s[%d].source must be string", key, index)
		}
		source = strings.TrimSpace(sourceValue)
	}

	return rollModifierMetadata{Value: value, Source: source}, nil
}

func (m rollSystemMetadata) outcomeOrFallback(fallback string) string {
	if trimmed := strings.TrimSpace(m.Outcome); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func (m rollSystemMetadata) rollKindCode() string {
	return strings.TrimSpace(m.RollKind)
}

func (m rollSystemMetadata) rollKindOrDefault() pb.RollKind {
	switch m.rollKindCode() {
	case pb.RollKind_ROLL_KIND_REACTION.String():
		return pb.RollKind_ROLL_KIND_REACTION
	case pb.RollKind_ROLL_KIND_ACTION.String():
		return pb.RollKind_ROLL_KIND_ACTION
	default:
		return pb.RollKind_ROLL_KIND_ACTION
	}
}

func boolPointerValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intPointerValue(value *int) (int, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}
