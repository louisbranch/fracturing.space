package daggerheartprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutDaggerheartCharacterState persists a Daggerheart character state extension.
func (s *Store) PutDaggerheartCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(state.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(state.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	conditions := state.Conditions
	if conditions == nil {
		conditions = []projectionstore.DaggerheartConditionState{}
	}
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return fmt.Errorf("encode conditions: %w", err)
	}
	temporaryArmorJSON, err := json.Marshal(state.TemporaryArmor)
	if err != nil {
		return fmt.Errorf("encode temporary armor: %w", err)
	}
	classStateJSON, err := json.Marshal(state.ClassState)
	if err != nil {
		return fmt.Errorf("encode class state: %w", err)
	}
	subclassStateJSON, err := json.Marshal(state.SubclassState)
	if err != nil {
		return fmt.Errorf("encode subclass state: %w", err)
	}
	companionStateJSON, err := json.Marshal(state.CompanionState)
	if err != nil {
		return fmt.Errorf("encode companion state: %w", err)
	}
	statModifiers := state.StatModifiers
	if statModifiers == nil {
		statModifiers = []projectionstore.DaggerheartStatModifier{}
	}
	statModifiersJSON, err := json.Marshal(statModifiers)
	if err != nil {
		return fmt.Errorf("encode stat modifiers: %w", err)
	}

	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheartstate.HopeMaxDefault
	}

	lifeState := state.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheartstate.LifeStateAlive
	}

	return s.q.PutDaggerheartCharacterState(ctx, db.PutDaggerheartCharacterStateParams{
		CampaignID:                    state.CampaignID,
		CharacterID:                   state.CharacterID,
		Hp:                            int64(state.Hp),
		Hope:                          int64(state.Hope),
		HopeMax:                       int64(hopeMax),
		Stress:                        int64(state.Stress),
		Armor:                         int64(state.Armor),
		ConditionsJson:                string(conditionsJSON),
		TemporaryArmorJson:            string(temporaryArmorJSON),
		LifeState:                     lifeState,
		ClassStateJson:                string(classStateJSON),
		SubclassStateJson:             string(subclassStateJSON),
		CompanionStateJson:            string(companionStateJSON),
		ImpenetrableUsedThisShortRest: boolToInt64(state.ImpenetrableUsedThisShortRest),
		StatModifiersJson:             string(statModifiersJSON),
	})
}

// GetDaggerheartCharacterState retrieves a Daggerheart character state extension.
func (s *Store) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetDaggerheartCharacterState(ctx, db.GetDaggerheartCharacterStateParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("get daggerheart character state: %w", err)
	}

	var conditions []projectionstore.DaggerheartConditionState
	if row.ConditionsJson != "" {
		conditions, err = decodeProjectionConditionStates(row.ConditionsJson)
		if err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode conditions: %w", err)
		}
	}
	var temporaryArmor []projectionstore.DaggerheartTemporaryArmor
	if row.TemporaryArmorJson != "" {
		if err := json.Unmarshal([]byte(row.TemporaryArmorJson), &temporaryArmor); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode temporary armor: %w", err)
		}
	}
	var classState projectionstore.DaggerheartClassState
	if row.ClassStateJson != "" {
		if err := json.Unmarshal([]byte(row.ClassStateJson), &classState); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode class state: %w", err)
		}
	}
	var subclassState *projectionstore.DaggerheartSubclassState
	if row.SubclassStateJson != "" && row.SubclassStateJson != "null" && row.SubclassStateJson != "{}" {
		var decoded projectionstore.DaggerheartSubclassState
		if err := json.Unmarshal([]byte(row.SubclassStateJson), &decoded); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode subclass state: %w", err)
		}
		subclassState = &decoded
	}
	var companionState *projectionstore.DaggerheartCompanionState
	if row.CompanionStateJson != "" && row.CompanionStateJson != "null" && row.CompanionStateJson != "{}" {
		var decoded projectionstore.DaggerheartCompanionState
		if err := json.Unmarshal([]byte(row.CompanionStateJson), &decoded); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode companion state: %w", err)
		}
		companionState = &decoded
	}

	lifeState := row.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheartstate.LifeStateAlive
	}
	var statModifiers []projectionstore.DaggerheartStatModifier
	if row.StatModifiersJson != "" && row.StatModifiersJson != "[]" {
		if err := json.Unmarshal([]byte(row.StatModifiersJson), &statModifiers); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode stat modifiers: %w", err)
		}
	}

	return projectionstore.DaggerheartCharacterState{
		CampaignID:                    row.CampaignID,
		CharacterID:                   row.CharacterID,
		Hp:                            int(row.Hp),
		Hope:                          int(row.Hope),
		HopeMax:                       int(row.HopeMax),
		Stress:                        int(row.Stress),
		Armor:                         int(row.Armor),
		TemporaryArmor:                temporaryArmor,
		Conditions:                    conditions,
		LifeState:                     lifeState,
		ClassState:                    classState,
		SubclassState:                 subclassState,
		CompanionState:                companionState,
		ImpenetrableUsedThisShortRest: row.ImpenetrableUsedThisShortRest != 0,
		StatModifiers:                 statModifiers,
	}, nil
}
