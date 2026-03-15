package workflowtransport

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// DecodeRollSystemMetadata decodes and type-checks the roll system_data payload.
func DecodeRollSystemMetadata(systemData map[string]any) (RollSystemMetadata, error) {
	metadata := RollSystemMetadata{}
	if len(systemData) == 0 {
		return metadata, nil
	}

	var err error
	if metadata.CharacterID, err = decodeOptionalStringField(systemData, KeyCharacterID); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.AdversaryID, err = decodeOptionalStringField(systemData, KeyAdversaryID); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Trait, err = decodeOptionalStringField(systemData, "trait"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.RollKind, err = decodeOptionalStringField(systemData, KeyRollKind); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Outcome, err = decodeOptionalStringField(systemData, KeyOutcome); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Flavor, err = decodeOptionalStringField(systemData, "flavor"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.BreathCountdownID, err = decodeOptionalStringField(systemData, "breath_countdown_id"); err != nil {
		return RollSystemMetadata{}, err
	}

	if metadata.HopeFear, err = decodeOptionalBoolField(systemData, KeyHopeFear); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Crit, err = decodeOptionalBoolField(systemData, KeyCrit); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.CritNegates, err = decodeOptionalBoolField(systemData, KeyCritNegates); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.GMMove, err = decodeOptionalBoolField(systemData, "gm_move"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Underwater, err = decodeOptionalBoolField(systemData, "underwater"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Critical, err = decodeOptionalBoolField(systemData, "critical"); err != nil {
		return RollSystemMetadata{}, err
	}

	if metadata.Roll, err = decodeOptionalIntField(systemData, KeyRoll); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Modifier, err = decodeOptionalIntField(systemData, KeyModifier); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Total, err = decodeOptionalIntField(systemData, KeyTotal); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.BaseTotal, err = decodeOptionalIntField(systemData, "base_total"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.CriticalBonus, err = decodeOptionalIntField(systemData, "critical_bonus"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Advantage, err = decodeOptionalIntField(systemData, "advantage"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Disadvantage, err = decodeOptionalIntField(systemData, "disadvantage"); err != nil {
		return RollSystemMetadata{}, err
	}
	if metadata.Modifiers, err = decodeOptionalModifierList(systemData, "modifiers"); err != nil {
		return RollSystemMetadata{}, err
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
	return BoolPtr(boolValue), nil
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
	return IntPtr(decoded), nil
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

func decodeOptionalModifierList(systemData map[string]any, key string) ([]RollModifierMetadata, error) {
	value, ok := systemData[key]
	if !ok || value == nil {
		return nil, nil
	}

	switch typed := value.(type) {
	case []RollModifierMetadata:
		cloned := make([]RollModifierMetadata, len(typed))
		copy(cloned, typed)
		return cloned, nil
	case []any:
		decoded := make([]RollModifierMetadata, 0, len(typed))
		for idx, entry := range typed {
			modifier, err := decodeModifierEntry(entry, key, idx)
			if err != nil {
				return nil, err
			}
			decoded = append(decoded, modifier)
		}
		return decoded, nil
	case []map[string]any:
		decoded := make([]RollModifierMetadata, 0, len(typed))
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

func decodeModifierEntry(entry any, key string, index int) (RollModifierMetadata, error) {
	switch typed := entry.(type) {
	case RollModifierMetadata:
		return typed, nil
	case map[string]any:
		return decodeModifierMap(typed, key, index)
	default:
		return RollModifierMetadata{}, fmt.Errorf("system_data.%s[%d] must be object", key, index)
	}
}

func decodeModifierMap(entry map[string]any, key string, index int) (RollModifierMetadata, error) {
	rawValue, ok := entry["value"]
	if !ok || rawValue == nil {
		return RollModifierMetadata{}, fmt.Errorf("system_data.%s[%d].value is required", key, index)
	}
	value, err := decodeIntValue(rawValue)
	if err != nil {
		return RollModifierMetadata{}, fmt.Errorf("system_data.%s[%d].value %w", key, index, err)
	}

	source := ""
	if rawSource, ok := entry["source"]; ok && rawSource != nil {
		sourceValue, ok := rawSource.(string)
		if !ok {
			return RollModifierMetadata{}, fmt.Errorf("system_data.%s[%d].source must be string", key, index)
		}
		source = strings.TrimSpace(sourceValue)
	}

	return RollModifierMetadata{Value: value, Source: source}, nil
}
