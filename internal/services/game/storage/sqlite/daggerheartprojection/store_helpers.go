package daggerheartprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func (s *Store) validateProjectionStore(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return nil
}

func requireProjectionField(value, label string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", label)
	}
	return nil
}

func domainConditionStatesToProjection(values []rules.ConditionState) []projectionstore.DaggerheartConditionState {
	if len(values) == 0 {
		return []projectionstore.DaggerheartConditionState{}
	}
	items := make([]projectionstore.DaggerheartConditionState, 0, len(values))
	for _, value := range values {
		triggers := make([]string, 0, len(value.ClearTriggers))
		for _, trigger := range value.ClearTriggers {
			triggers = append(triggers, string(trigger))
		}
		items = append(items, projectionstore.DaggerheartConditionState{
			ID:            value.ID,
			Class:         string(value.Class),
			Standard:      value.Standard,
			Code:          value.Code,
			Label:         value.Label,
			Source:        value.Source,
			SourceID:      value.SourceID,
			ClearTriggers: triggers,
		})
	}
	return items
}

func projectionConditionStatesToDomain(values []projectionstore.DaggerheartConditionState) []rules.ConditionState {
	if len(values) == 0 {
		return []rules.ConditionState{}
	}
	items := make([]rules.ConditionState, 0, len(values))
	for _, value := range values {
		triggers := make([]rules.ConditionClearTrigger, 0, len(value.ClearTriggers))
		for _, trigger := range value.ClearTriggers {
			triggers = append(triggers, rules.ConditionClearTrigger(trigger))
		}
		items = append(items, rules.ConditionState{
			ID:            value.ID,
			Class:         rules.ConditionClass(value.Class),
			Standard:      value.Standard,
			Code:          value.Code,
			Label:         value.Label,
			Source:        value.Source,
			SourceID:      value.SourceID,
			ClearTriggers: triggers,
		})
	}
	return items
}

func decodeProjectionConditionStates(raw string) ([]projectionstore.DaggerheartConditionState, error) {
	if strings.TrimSpace(raw) == "" {
		return []projectionstore.DaggerheartConditionState{}, nil
	}
	var structured []projectionstore.DaggerheartConditionState
	if err := json.Unmarshal([]byte(raw), &structured); err != nil {
		return nil, err
	}
	return structured, nil
}

func normalizeLegacyProjectionConditionStatesJSON(raw string) (string, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return raw, false, nil
	}
	var structured []projectionstore.DaggerheartConditionState
	if err := json.Unmarshal([]byte(raw), &structured); err == nil {
		return raw, false, nil
	}
	var legacy []string
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return "", false, err
	}

	items := make([]projectionstore.DaggerheartConditionState, 0, len(legacy))
	for _, code := range legacy {
		state, err := rules.StandardConditionState(code)
		if err != nil {
			return "", false, err
		}
		items = append(items, domainConditionStatesToProjection([]rules.ConditionState{state})...)
	}
	normalized, err := json.Marshal(items)
	if err != nil {
		return "", false, fmt.Errorf("marshal normalized condition states: %w", err)
	}
	return string(normalized), true, nil
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func toNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func mapPageRows[Row any, Item any](
	rows []Row,
	pageSize int,
	rowID func(Row) string,
	mapRow func(Row) (Item, error),
) ([]Item, string, error) {
	capHint := pageSize
	if capHint > len(rows) {
		capHint = len(rows)
	}
	items := make([]Item, 0, capHint)

	for i, row := range rows {
		if i >= pageSize {
			return items, rowID(rows[pageSize-1]), nil
		}
		item, err := mapRow(row)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}

	return items, "", nil
}
